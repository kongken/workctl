// Package multicacli adapts the installed Multica CLI for workctl commands.
// Keeping this behind a small client makes a future HTTP adapter possible
// without coupling Linear commands to Multica's transport details.
package multicacli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// Client invokes the Multica CLI executable.
type Client struct {
	Command string
	Exec    Executor
}

// Executor is the process seam used by tests and future integrations.
type Executor func(ctx context.Context, command string, args []string, stdin io.Reader) ([]byte, []byte, error)

// New returns a client using the configured command or the default binary.
func New(command string) Client {
	if strings.TrimSpace(command) == "" {
		command = "multica"
	}
	return Client{Command: command}
}

// UpdateAgentInstructions replaces the persisted instructions for an agent.
func (c Client) UpdateAgentInstructions(ctx context.Context, agentID, instructions string) (json.RawMessage, error) {
	return c.run(ctx, []string{
		"agent", "update", agentID,
		"--instructions", instructions,
		"--output", "json",
	}, nil)
}

// CreateIssueInput describes a Linear issue handoff to Multica.
type CreateIssueInput struct {
	Title       string
	Description string
	AgentID     string
	ProjectID   string
	Status      string
}

// CreateIssue creates and assigns a Multica issue through the CLI.
func (c Client) CreateIssue(ctx context.Context, input CreateIssueInput) (json.RawMessage, error) {
	args := []string{
		"issue", "create",
		"--title", input.Title,
		"--description-stdin",
		"--assignee-id", input.AgentID,
		"--status", input.Status,
		"--output", "json",
	}
	if input.ProjectID != "" {
		args = append(args, "--project", input.ProjectID)
	}
	return c.run(ctx, args, strings.NewReader(input.Description))
}

func (c Client) run(ctx context.Context, args []string, stdin io.Reader) (json.RawMessage, error) {
	if c.Exec != nil {
		stdout, stderr, err := c.Exec(ctx, c.Command, args, stdin)
		if err != nil {
			message := strings.TrimSpace(string(stderr))
			if message == "" {
				message = err.Error()
			}
			return nil, fmt.Errorf("multica command failed: %s", message)
		}
		return validateJSON(stdout)
	}
	cmd := exec.CommandContext(ctx, c.Command, args...)
	cmd.Stdin = stdin
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}
		return nil, fmt.Errorf("multica command failed: %s", message)
	}
	return validateJSON(stdout.Bytes())
}

func validateJSON(raw []byte) (json.RawMessage, error) {
	data := bytes.TrimSpace(raw)
	if len(data) == 0 {
		return nil, fmt.Errorf("multica command returned no JSON")
	}
	if !json.Valid(data) {
		return nil, fmt.Errorf("multica command returned invalid JSON")
	}
	return json.RawMessage(data), nil
}
