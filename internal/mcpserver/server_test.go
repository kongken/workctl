package mcpserver_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/kongken/workctl/internal/mcpserver"
)

func testServer() *mcpserver.Server {
	return &mcpserver.Server{
		Name:    "workctl",
		Version: "test",
		Tools: []mcpserver.Tool{
			{
				Name:        "echo",
				Description: "Echo the input",
				InputSchema: json.RawMessage(`{"type":"object","properties":{"value":{"type":"string"}},"required":["value"]}`),
				Handler: func(_ context.Context, args json.RawMessage) (string, error) {
					var in struct {
						Value string `json:"value"`
					}
					if err := json.Unmarshal(args, &in); err != nil {
						return "", err
					}
					return "echo: " + in.Value, nil
				},
			},
			{
				Name:    "fail",
				Handler: func(context.Context, json.RawMessage) (string, error) { return "", errors.New("boom") },
			},
		},
	}
}

func runServer(t *testing.T, input string) []map[string]any {
	t.Helper()
	var out bytes.Buffer
	err := testServer().Serve(context.Background(), bytes.NewBufferString(input), &out)
	if err != nil {
		t.Fatalf("serve: %v", err)
	}
	var responses []map[string]any
	scanner := bufio.NewScanner(&out)
	for scanner.Scan() {
		var decoded map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &decoded); err != nil {
			t.Fatalf("decode response %q: %v", scanner.Text(), err)
		}
		responses = append(responses, decoded)
	}
	return responses
}

func TestInitializeEchoesProtocolVersion(t *testing.T) {
	responses := runServer(t, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"c","version":"1"}}}`+"\n")
	if len(responses) != 1 {
		t.Fatalf("responses = %d", len(responses))
	}
	result, ok := responses[0]["result"].(map[string]any)
	if !ok {
		t.Fatalf("missing result: %#v", responses[0])
	}
	if result["protocolVersion"] != "2024-11-05" {
		t.Fatalf("protocolVersion = %v", result["protocolVersion"])
	}
	capabilities, _ := result["capabilities"].(map[string]any)
	if _, ok := capabilities["tools"]; !ok {
		t.Fatalf("tools capability missing: %#v", capabilities)
	}
}

func TestNotificationsProduceNoResponse(t *testing.T) {
	responses := runServer(t, `{"jsonrpc":"2.0","method":"notifications/initialized"}`+"\n"+`{"jsonrpc":"2.0","id":7,"method":"ping"}`+"\n")
	if len(responses) != 1 {
		t.Fatalf("responses = %d, want 1", len(responses))
	}
	if responses[0]["id"].(float64) != 7 {
		t.Fatalf("id = %v", responses[0]["id"])
	}
}

func TestToolsList(t *testing.T) {
	responses := runServer(t, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`+"\n")
	result := responses[0]["result"].(map[string]any)
	tools := result["tools"].([]any)
	if len(tools) != 2 {
		t.Fatalf("tools = %d", len(tools))
	}
	first := tools[0].(map[string]any)
	if first["name"] != "echo" || first["description"] != "Echo the input" {
		t.Fatalf("tool = %#v", first)
	}
	if _, ok := first["inputSchema"].(map[string]any); !ok {
		t.Fatalf("inputSchema is not an object: %#v", first["inputSchema"])
	}
}

func TestToolsCallReturnsText(t *testing.T) {
	responses := runServer(t, `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"echo","arguments":{"value":"hi"}}}`+"\n")
	result := responses[0]["result"].(map[string]any)
	if result["isError"] != nil {
		t.Fatalf("unexpected error: %#v", result)
	}
	content := result["content"].([]any)
	text := content[0].(map[string]any)["text"]
	if text != "echo: hi" {
		t.Fatalf("text = %v", text)
	}
}

func TestToolsCallReportsHandlerErrors(t *testing.T) {
	responses := runServer(t, `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"fail"}}`+"\n")
	result := responses[0]["result"].(map[string]any)
	if result["isError"] != true {
		t.Fatalf("isError = %v", result["isError"])
	}
	content := result["content"].([]any)
	if content[0].(map[string]any)["text"] != "boom" {
		t.Fatalf("content = %#v", content)
	}
}

func TestUnknownToolIsProtocolError(t *testing.T) {
	responses := runServer(t, `{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"missing"}}`+"\n")
	rpcErr, ok := responses[0]["error"].(map[string]any)
	if !ok {
		t.Fatalf("missing error: %#v", responses[0])
	}
	if rpcErr["code"].(float64) != -32602 {
		t.Fatalf("code = %v", rpcErr["code"])
	}
}

func TestParseErrorIsReported(t *testing.T) {
	responses := runServer(t, "not-json\n")
	rpcErr, ok := responses[0]["error"].(map[string]any)
	if !ok {
		t.Fatalf("missing error: %#v", responses[0])
	}
	if rpcErr["code"].(float64) != -32700 {
		t.Fatalf("code = %v", rpcErr["code"])
	}
}
