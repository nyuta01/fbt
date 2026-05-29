package protocol

import "encoding/json"

const (
	JSONRPCVersion = "2.0"
	Version        = "0.1"
	FramingJSONL   = "jsonl"
)

const (
	MethodInitialize      = "initialize"
	MethodInitialized     = "initialized"
	MethodRunTransform    = "fbt/runTransform"
	MethodValidate        = "fbt/validate"
	MethodEvent           = "fbt/event"
	MethodOutputCandidate = "fbt/outputCandidate"
	MethodHeartbeat       = "fbt/heartbeat"
	MethodCancelRequest   = "$/cancelRequest"
)

const (
	ErrorParse          = -32700
	ErrorInvalidRequest = -32600
	ErrorMethodNotFound = -32601
	ErrorInvalidParams  = -32602
	ErrorInternal       = -32603
	ErrorRunner         = -32099
)

type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      string          `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type Response struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      string    `json:"id,omitempty"`
	Result  any       `json:"result,omitempty"`
	Error   *RPCError `json:"error,omitempty"`
}

type Notification struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

type RPCError struct {
	Code    int            `json:"code"`
	Message string         `json:"message"`
	Data    map[string]any `json:"data,omitempty"`
}

func (e RPCError) Error() string {
	return e.Message
}

type InitializeParams struct {
	Core              map[string]string `json:"core,omitempty"`
	Protocol          map[string]any    `json:"protocol,omitempty"`
	CapabilityRequest []string          `json:"capability_request,omitempty"`
}

type InitializeResult struct {
	Runner       RunnerInfo   `json:"runner"`
	Protocol     ProtocolInfo `json:"protocol"`
	Capabilities Capabilities `json:"capabilities"`
}

type RunnerInfo struct {
	ID       string `json:"id,omitempty"`
	Name     string `json:"name"`
	Version  string `json:"version"`
	Language string `json:"language,omitempty"`
}

type ProtocolInfo struct {
	Version string `json:"version"`
	Framing string `json:"framing"`
}

type Capabilities struct {
	TransformTypes   []string `json:"transform_types"`
	ArtifactTypes    []string `json:"artifact_types"`
	StreamEvents     bool     `json:"stream_events,omitempty"`
	ToolCallLog      bool     `json:"tool_call_log,omitempty"`
	UsageReporting   bool     `json:"usage_reporting,omitempty"`
	CostEstimation   bool     `json:"cost_estimation,omitempty"`
	OutputCandidates bool     `json:"output_candidates"`
	SupportsDryRun   bool     `json:"supports_dry_run,omitempty"`
	SupportsCancel   bool     `json:"supports_cancel,omitempty"`
}

type RunTransformParams struct {
	Mode           string         `json:"mode"`
	InvocationID   string         `json:"invocation_id,omitempty"`
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

type DeclaredOutput struct {
	Name         string `json:"name"`
	ArtifactType string `json:"artifact_type"`
	DeclaredPath string `json:"declared_path,omitempty"`
}
