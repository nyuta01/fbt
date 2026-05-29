package protocol

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"sync"
)

type Client struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	cancel context.CancelFunc

	mu       sync.Mutex
	nextID   int
	incoming chan incomingMessage
	readErr  chan error
}

type Options struct {
	Dir string
	Env []string
}

type JSONRPCError struct {
	Code    int            `json:"code"`
	Message string         `json:"message"`
	Data    map[string]any `json:"data,omitempty"`
}

func (e JSONRPCError) Error() string {
	return fmt.Sprintf("json-rpc error %d: %s", e.Code, e.Message)
}

type InitializeParams struct {
	Core              map[string]string `json:"core"`
	Protocol          map[string]any    `json:"protocol"`
	CapabilityRequest []string          `json:"capability_request"`
}

type InitializeResult struct {
	Runner       map[string]any `json:"runner"`
	Protocol     map[string]any `json:"protocol"`
	Capabilities map[string]any `json:"capabilities"`
}

type RunTransformParams struct {
	Mode           string         `json:"mode"`
	InvocationID   string         `json:"invocation_id"`
	TransformRunID string         `json:"transform_run_id"`
	Transform      map[string]any `json:"transform"`
	Runner         map[string]any `json:"runner,omitempty"`
	Inputs         []any          `json:"inputs,omitempty"`
	Outputs        []any          `json:"outputs,omitempty"`
	Assets         []any          `json:"assets,omitempty"`
	Model          map[string]any `json:"model,omitempty"`
	Tools          []any          `json:"tools,omitempty"`
	Policy         map[string]any `json:"policy,omitempty"`
	State          map[string]any `json:"state,omitempty"`
	Work           map[string]any `json:"work,omitempty"`
}

type RunTransformResult struct {
	Status         string         `json:"status"`
	TransformRunID string         `json:"transform_run_id"`
	Outputs        []any          `json:"outputs,omitempty"`
	Usage          map[string]any `json:"usage,omitempty"`
	Provenance     map[string]any `json:"provenance,omitempty"`
	Warnings       []string       `json:"warnings,omitempty"`
}

type Event struct {
	RequestID      string         `json:"request_id"`
	TransformRunID string         `json:"transform_run_id"`
	Time           string         `json:"time,omitempty"`
	EventType      string         `json:"event_type"`
	Level          string         `json:"level,omitempty"`
	Message        string         `json:"message,omitempty"`
	Attributes     map[string]any `json:"attributes,omitempty"`
	ToolCall       map[string]any `json:"tool_call,omitempty"`
}

type OutputCandidate struct {
	RequestID      string `json:"request_id"`
	TransformRunID string `json:"transform_run_id"`
	Outputs        []any  `json:"outputs"`
}

type RunOutcome struct {
	Result           RunTransformResult
	Events           []Event
	OutputCandidates []OutputCandidate
}

type rpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      string `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      string          `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
}

type rpcNotification struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type incomingMessage struct {
	Response     *rpcResponse
	Notification *rpcNotification
}

func Start(ctx context.Context, command string, args []string, options Options) (*Client, error) {
	processCtx, cancel := context.WithCancel(ctx)
	cmd := exec.CommandContext(processCtx, command, args...)
	cmd.Dir = options.Dir
	if len(options.Env) > 0 {
		cmd.Env = options.Env
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		cancel()
		return nil, err
	}
	client := &Client{
		cmd:      cmd,
		stdin:    stdin,
		cancel:   cancel,
		incoming: make(chan incomingMessage, 32),
		readErr:  make(chan error, 1),
	}
	go client.readLoop(stdout)
	return client, nil
}

func (c *Client) Initialize(ctx context.Context, params InitializeParams) (InitializeResult, error) {
	var result InitializeResult
	if err := c.call(ctx, "initialize", params, &result, nil); err != nil {
		return InitializeResult{}, err
	}
	if err := c.Notify("initialized", map[string]any{}); err != nil {
		return InitializeResult{}, err
	}
	return result, nil
}

func (c *Client) RunTransform(ctx context.Context, params RunTransformParams) (RunOutcome, error) {
	var result RunTransformResult
	outcome := RunOutcome{}
	if err := c.call(ctx, "fbt/runTransform", params, &result, &outcome); err != nil {
		return outcome, err
	}
	outcome.Result = result
	return outcome, nil
}

func (c *Client) Cancel(id string, reason string) error {
	return c.Notify("$/cancelRequest", map[string]any{"id": id, "reason": reason})
}

func (c *Client) Notify(method string, params any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	message := map[string]any{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
	}
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}
	_, err = c.stdin.Write(append(data, '\n'))
	return err
}

func (c *Client) Close() error {
	if c.cancel != nil {
		c.cancel()
	}
	if c.stdin != nil {
		_ = c.stdin.Close()
	}
	if c.cmd != nil && c.cmd.Process != nil {
		_ = c.cmd.Process.Kill()
		return c.cmd.Wait()
	}
	return nil
}

func (c *Client) call(ctx context.Context, method string, params any, result any, outcome *RunOutcome) error {
	id := c.nextRequestID()
	request := rpcRequest{JSONRPC: "2.0", ID: id, Method: method, Params: params}
	if err := c.writeRequest(request); err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			_ = c.Cancel(id, ctx.Err().Error())
			return ctx.Err()
		case err := <-c.readErr:
			if err == nil {
				return io.EOF
			}
			return err
		case message := <-c.incoming:
			if message.Notification != nil {
				if outcome != nil {
					collectNotification(outcome, message.Notification)
				}
				continue
			}
			if message.Response == nil || message.Response.ID != id {
				continue
			}
			if message.Response.Error != nil {
				return *message.Response.Error
			}
			if result == nil {
				return nil
			}
			return json.Unmarshal(message.Response.Result, result)
		}
	}
}

func (c *Client) nextRequestID() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.nextID++
	return fmt.Sprintf("req_%d", c.nextID)
}

func (c *Client) writeRequest(request rpcRequest) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	data, err := json.Marshal(request)
	if err != nil {
		return err
	}
	_, err = c.stdin.Write(append(data, '\n'))
	return err
}

func (c *Client) readLoop(stdout io.Reader) {
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		message, err := decodeIncoming(scanner.Bytes())
		if err != nil {
			c.readErr <- err
			return
		}
		c.incoming <- message
	}
	if err := scanner.Err(); err != nil {
		c.readErr <- err
		return
	}
	c.readErr <- nil
}

func decodeIncoming(data []byte) (incomingMessage, error) {
	var probe struct {
		ID     *string `json:"id"`
		Method string  `json:"method"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		return incomingMessage{}, err
	}
	if probe.ID != nil {
		var response rpcResponse
		if err := json.Unmarshal(data, &response); err != nil {
			return incomingMessage{}, err
		}
		if response.JSONRPC != "2.0" {
			return incomingMessage{}, errors.New("invalid jsonrpc version")
		}
		return incomingMessage{Response: &response}, nil
	}
	if probe.Method != "" {
		var notification rpcNotification
		if err := json.Unmarshal(data, &notification); err != nil {
			return incomingMessage{}, err
		}
		if notification.JSONRPC != "2.0" {
			return incomingMessage{}, errors.New("invalid jsonrpc version")
		}
		return incomingMessage{Notification: &notification}, nil
	}
	return incomingMessage{}, errors.New("message is neither response nor notification")
}

func collectNotification(outcome *RunOutcome, notification *rpcNotification) {
	switch notification.Method {
	case "fbt/event":
		var event Event
		if err := json.Unmarshal(notification.Params, &event); err == nil {
			outcome.Events = append(outcome.Events, event)
		}
	case "fbt/outputCandidate":
		var candidate OutputCandidate
		if err := json.Unmarshal(notification.Params, &candidate); err == nil {
			outcome.OutputCandidates = append(outcome.OutputCandidates, candidate)
		}
	}
}
