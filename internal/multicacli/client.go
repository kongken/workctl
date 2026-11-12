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

// AgentCreateInput contains the writable fields supported by the agent
// lifecycle commands. Name and RuntimeID are required by Multica.
type AgentCreateInput struct {
	Name               string
	RuntimeID          string
	Description        string
	Instructions       string
	Model              string
	MaxConcurrentTasks *int32
}

// AgentUpdateInput uses pointers so callers can distinguish an omitted field
// from an explicit empty value (which clears that field in Multica).
type AgentUpdateInput struct {
	Name               *string
	Description        *string
	Instructions       *string
	Model              *string
	Status             *string
	RuntimeID          *string
	MaxConcurrentTasks *int32
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
	return c.UpdateAgent(ctx, agentID, AgentUpdateInput{Instructions: &instructions})
}

// ListAgents returns all visible agents, optionally including archived agents.
func (c Client) ListAgents(ctx context.Context, includeArchived bool) (json.RawMessage, error) {
	args := []string{"agent", "list"}
	if includeArchived {
		args = append(args, "--include-archived")
	}
	args = append(args, "--output", "json")
	return c.run(ctx, args, nil)
}

// GetAgent returns one agent by UUID.
func (c Client) GetAgent(ctx context.Context, agentID string) (json.RawMessage, error) {
	return c.run(ctx, []string{"agent", "get", agentID, "--output", "json"}, nil)
}

// CreateAgent creates a Multica agent.
func (c Client) CreateAgent(ctx context.Context, input AgentCreateInput) (json.RawMessage, error) {
	args := []string{"agent", "create", "--name", input.Name, "--runtime-id", input.RuntimeID}
	appendAgentCreateOptions(&args, input)
	args = append(args, "--output", "json")
	return c.run(ctx, args, nil)
}

// UpdateAgent updates the fields present in input.
func (c Client) UpdateAgent(ctx context.Context, agentID string, input AgentUpdateInput) (json.RawMessage, error) {
	args := []string{"agent", "update", agentID}
	appendAgentUpdateOptions(&args, input)
	args = append(args, "--output", "json")
	return c.run(ctx, args, nil)
}

// ArchiveAgent is the delete operation exposed by workctl. Multica retains
// archived agents so they can be restored later.
func (c Client) ArchiveAgent(ctx context.Context, agentID string) (json.RawMessage, error) {
	return c.run(ctx, []string{"agent", "archive", agentID, "--output", "json"}, nil)
}

// RestoreAgent restores an archived agent.
func (c Client) RestoreAgent(ctx context.Context, agentID string) (json.RawMessage, error) {
	return c.run(ctx, []string{"agent", "restore", agentID, "--output", "json"}, nil)
}

func appendAgentCreateOptions(args *[]string, input AgentCreateInput) {
	if input.Description != "" {
		*args = append(*args, "--description", input.Description)
	}
	if input.Instructions != "" {
		*args = append(*args, "--instructions", input.Instructions)
	}
	if input.Model != "" {
		*args = append(*args, "--model", input.Model)
	}
	if input.MaxConcurrentTasks != nil {
		*args = append(*args, "--max-concurrent-tasks", fmt.Sprint(*input.MaxConcurrentTasks))
	}
}

func appendAgentUpdateOptions(args *[]string, input AgentUpdateInput) {
	if input.Name != nil {
		*args = append(*args, "--name", *input.Name)
	}
	if input.Description != nil {
		*args = append(*args, "--description", *input.Description)
	}
	if input.Instructions != nil {
		*args = append(*args, "--instructions", *input.Instructions)
	}
	if input.Model != nil {
		*args = append(*args, "--model", *input.Model)
	}
	if input.Status != nil {
		*args = append(*args, "--status", *input.Status)
	}
	if input.RuntimeID != nil {
		*args = append(*args, "--runtime-id", *input.RuntimeID)
	}
	if input.MaxConcurrentTasks != nil {
		*args = append(*args, "--max-concurrent-tasks", fmt.Sprint(*input.MaxConcurrentTasks))
	}
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
