package stdiojsonrpc

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/nyuta01/fbt/sdk/go/protocol"
)

type Handler struct {
	Initialize   func(context.Context, protocol.Request, *Writer) (any, error)
	Initialized  func(context.Context, protocol.Request, *Writer) error
	RunTransform func(context.Context, protocol.Request, *Writer) (any, error)
	Validate     func(context.Context, protocol.Request, *Writer) (any, error)
	Cancel       func(context.Context, protocol.Request, *Writer) error
}

type Writer struct {
	mu  sync.Mutex
	out io.Writer
}

func NewWriter(out io.Writer) *Writer {
	return &Writer{out: out}
}

func Serve(ctx context.Context, in io.Reader, out io.Writer, handler Handler) error {
	scanner := bufio.NewScanner(in)
	writer := NewWriter(out)
	for scanner.Scan() {
		if err := handleLine(ctx, scanner.Bytes(), writer, handler); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func handleLine(ctx context.Context, line []byte, writer *Writer, handler Handler) error {
	var req protocol.Request
	if err := json.Unmarshal(line, &req); err != nil {
		return writer.Error("", protocol.ErrorParse, err.Error(), nil)
	}
	if req.JSONRPC != "" && req.JSONRPC != protocol.JSONRPCVersion {
		return writer.Error(req.ID, protocol.ErrorInvalidRequest, "invalid jsonrpc version", nil)
	}

	switch req.Method {
	case protocol.MethodInitialize:
		if handler.Initialize == nil {
			return writer.Error(req.ID, protocol.ErrorMethodNotFound, "method not found", nil)
		}
		result, err := handler.Initialize(ctx, req, writer)
		return writeResult(writer, req.ID, result, err)
	case protocol.MethodInitialized:
		if handler.Initialized != nil {
			return handler.Initialized(ctx, req, writer)
		}
		return nil
	case protocol.MethodRunTransform:
		if handler.RunTransform == nil {
			return writer.Error(req.ID, protocol.ErrorMethodNotFound, "method not found", nil)
		}
		result, err := handler.RunTransform(ctx, req, writer)
		return writeResult(writer, req.ID, result, err)
	case protocol.MethodValidate:
		if handler.Validate == nil {
			return writer.Error(req.ID, protocol.ErrorMethodNotFound, "method not found", nil)
		}
		result, err := handler.Validate(ctx, req, writer)
		return writeResult(writer, req.ID, result, err)
	case protocol.MethodCancelRequest:
		if handler.Cancel != nil {
			return handler.Cancel(ctx, req, writer)
		}
		return nil
	default:
		return writer.Error(req.ID, protocol.ErrorMethodNotFound, "method not found", nil)
	}
}

func writeResult(writer *Writer, id string, result any, err error) error {
	if err != nil {
		var rpcErr protocol.RPCError
		if ok := asRPCError(err, &rpcErr); ok {
			return writer.Error(id, rpcErr.Code, rpcErr.Message, rpcErr.Data)
		}
		return writer.Error(id, protocol.ErrorRunner, err.Error(), nil)
	}
	return writer.Response(id, result)
}

func asRPCError(err error, out *protocol.RPCError) bool {
	if rpcErr, ok := err.(protocol.RPCError); ok {
		*out = rpcErr
		return true
	}
	if rpcErr, ok := err.(*protocol.RPCError); ok {
		*out = *rpcErr
		return true
	}
	return false
}

func (w *Writer) Response(id string, result any) error {
	return w.write(protocol.Response{JSONRPC: protocol.JSONRPCVersion, ID: id, Result: result})
}

func (w *Writer) Notification(method string, params any) error {
	return w.write(protocol.Notification{JSONRPC: protocol.JSONRPCVersion, Method: method, Params: params})
}

func (w *Writer) Error(id string, code int, message string, data map[string]any) error {
	return w.write(protocol.Response{
		JSONRPC: protocol.JSONRPCVersion,
		ID:      id,
		Error:   &protocol.RPCError{Code: code, Message: message, Data: data},
	})
}

func (w *Writer) write(value any) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal json-rpc message: %w", err)
	}
	_, err = fmt.Fprintf(w.out, "%s\n", data)
	return err
}
