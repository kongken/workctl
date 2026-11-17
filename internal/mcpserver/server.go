// Package mcpserver implements a minimal Model Context Protocol server over
// newline-delimited JSON-RPC 2.0 on stdio.
//
// It implements the handshake (initialize / notifications/initialized), the
// tools capability (tools/list and tools/call), and ping. Tool execution errors
// are reported as isError tool results rather than protocol errors so clients
// can surface them to the model.
package mcpserver

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
)

// ProtocolVersion is the protocol version used when a client does not request
// one. It is the latest version this server supports.
const ProtocolVersion = "2025-06-18"

// supportedProtocolVersions lists the MCP protocol versions this server
// implements. Their initialize / tools / ping behavior is compatible with what
// this server provides. During initialization the server echoes the client's
// requested version when it is in this set and rejects the initialization when
// it is not, so it never establishes a session under a protocol contract it
// does not implement.
var supportedProtocolVersions = []string{"2025-06-18", "2025-03-26", "2024-11-05"}

// selectProtocolVersion resolves the protocol version for a session. It returns
// the negotiated version and true when the requested version is supported (or
// empty, in which case the default applies) and reports false when the client
// asked for a version this server does not implement.
func selectProtocolVersion(requested string) (string, bool) {
	if requested == "" {
		return ProtocolVersion, true
	}
	for _, version := range supportedProtocolVersions {
		if requested == version {
			return version, true
		}
	}
	return "", false
}

const maxMessageBytes = 8 << 20

// Handler runs a tool with raw JSON arguments and returns its text result.
type Handler func(ctx context.Context, args json.RawMessage) (string, error)

// Tool describes one MCP tool. Write marks a tool that mutates Multica state;
// it is metadata for callers (for example a read-only server mode) and is not
// sent to MCP clients.
type Tool struct {
	Name        string
	Description string
	InputSchema json.RawMessage
	Handler     Handler
	Write       bool
}

// Server exposes a fixed set of tools over an MCP stdio transport.
type Server struct {
	Name    string
	Version string
	Tools   []Tool
}

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

// Serve reads JSON-RPC messages from in and writes responses to out until the
// input is exhausted or the context is cancelled.
func (s *Server) Serve(ctx context.Context, in io.Reader, out io.Writer) error {
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 0, 64*1024), maxMessageBytes)
	encoder := json.NewEncoder(out)

	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return err
		}
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}

		var req rpcRequest
		if err := json.Unmarshal(line, &req); err != nil {
			if err := encoder.Encode(rpcResponse{
				JSONRPC: "2.0",
				ID:      json.RawMessage("null"),
				Error:   &rpcError{Code: -32700, Message: "parse error"},
			}); err != nil {
				return err
			}
			continue
		}

		// Notifications carry no id and must not be answered.
		if len(req.ID) == 0 {
			continue
		}

		resp := rpcResponse{JSONRPC: "2.0", ID: req.ID}
		result, rpcErr := s.handle(ctx, &req)
		if rpcErr != nil {
			resp.Error = rpcErr
		} else {
			resp.Result = result
		}
		if err := encoder.Encode(resp); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func (s *Server) handle(ctx context.Context, req *rpcRequest) (any, *rpcError) {
	switch req.Method {
	case "initialize":
		var params struct {
			ProtocolVersion string `json:"protocolVersion"`
		}
		_ = json.Unmarshal(req.Params, &params)
		version, ok := selectProtocolVersion(params.ProtocolVersion)
		if !ok {
			return nil, &rpcError{
				Code:    -32602,
				Message: fmt.Sprintf("unsupported protocol version: %s", params.ProtocolVersion),
			}
		}
		return map[string]any{
			"protocolVersion": version,
			"capabilities": map[string]any{
				"tools": map[string]any{"listChanged": false},
			},
			"serverInfo": map[string]any{"name": s.Name, "version": s.Version},
		}, nil
	case "ping":
		return map[string]any{}, nil
	case "tools/list":
		return map[string]any{"tools": s.toolList()}, nil
	case "tools/call":
		return s.callTool(ctx, req.Params)
	default:
		return nil, &rpcError{Code: -32601, Message: fmt.Sprintf("method not found: %s", req.Method)}
	}
}

func (s *Server) callTool(ctx context.Context, params json.RawMessage) (any, *rpcError) {
	var call struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(params, &call); err != nil {
		return nil, &rpcError{Code: -32602, Message: "invalid params"}
	}
	tool, ok := s.tool(call.Name)
	if !ok {
		return nil, &rpcError{Code: -32602, Message: fmt.Sprintf("unknown tool: %s", call.Name)}
	}

	args := call.Arguments
	if len(bytes.TrimSpace(args)) == 0 {
		args = json.RawMessage("{}")
	}
	text, err := tool.Handler(ctx, args)
	if err != nil {
		return map[string]any{
			"content": []map[string]any{{"type": "text", "text": err.Error()}},
			"isError": true,
		}, nil
	}
	return map[string]any{
		"content": []map[string]any{{"type": "text", "text": text}},
	}, nil
}

func (s *Server) toolList() []map[string]any {
	tools := make([]map[string]any, 0, len(s.Tools))
	for _, tool := range s.Tools {
		schema := tool.InputSchema
		if len(schema) == 0 {
			schema = json.RawMessage(`{"type":"object"}`)
		}
		tools = append(tools, map[string]any{
			"name":        tool.Name,
			"description": tool.Description,
			"inputSchema": schema,
		})
	}
	return tools
}

func (s *Server) tool(name string) (Tool, bool) {
	for _, tool := range s.Tools {
		if tool.Name == name {
			return tool, true
		}
	}
	return Tool{}, false
}
