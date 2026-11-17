package cmdmcp

import (
	"context"
	"io"
	"strings"

	"github.com/kongken/workctl/internal/mcpserver"
)

// tools returns the full tool set: the bridge and core CRUD tools plus the
// remaining Multica agent and issue management commands.
func (t toolset) tools() []mcpserver.Tool {
	tools := t.coreTools()
	tools = append(tools, t.agentTools()...)
	tools = append(tools, t.issueTools()...)
	return tools
}

// call runs a Multica CLI invocation for a tool handler. argv is the full
// argument list without the binary. body, when non-empty, is streamed to
// stdin. When jsonOut is true, --output json is appended and the parsed result
// is pretty-printed; otherwise stdout is returned as text (for commands that
// do not support --output).
func (t toolset) call(ctx context.Context, argv []string, body string, jsonOut bool) (string, error) {
	client := t.multicacli()
	var stdin io.Reader
	if body != "" {
		stdin = strings.NewReader(body)
	}
	if !jsonOut {
		return client.RunCommand(ctx, argv, stdin)
	}
	raw, err := client.Run(ctx, append(argv, "--output", "json"), stdin)
	if err != nil {
		return "", err
	}
	return prettyJSON(raw)
}
