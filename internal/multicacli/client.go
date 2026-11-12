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

// GetIssue returns one issue by UUID or identifier.
func (c Client) GetIssue(ctx context.Context, issueID string) (json.RawMessage, error) {
	return c.run(ctx, []string{"issue", "get", issueID, "--output", "json"}, nil)
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

// CreateIssueInput describes an issue to create through Multica.
// AssigneeID and ParentID accept a member, agent, or squad reference.
type CreateIssueInput struct {
	Title       string
	Description string
	AssigneeID  string
	ProjectID   string
	Status      string
	Priority    string
	ParentID    string
}

// CreateIssue creates a Multica issue through the CLI.
func (c Client) CreateIssue(ctx context.Context, input CreateIssueInput) (json.RawMessage, error) {
	args := []string{"issue", "create", "--title", input.Title, "--description-stdin", "--output", "json"}
	args = appendIssueCreateOptions(args, input)
	return c.run(ctx, args, strings.NewReader(input.Description))
}

func appendIssueCreateOptions(args []string, input CreateIssueInput) []string {
	if input.AssigneeID != "" {
		args = append(args, "--assignee-id", input.AssigneeID)
	}
	if input.ProjectID != "" {
		args = append(args, "--project", input.ProjectID)
	}
	if input.Status != "" {
		args = append(args, "--status", input.Status)
	}
	if input.Priority != "" {
		args = append(args, "--priority", input.Priority)
	}
	if input.ParentID != "" {
		args = append(args, "--parent", input.ParentID)
	}
	return args
}

// IssueListInput filters ListIssues.
type IssueListInput struct {
	AssigneeID string
	Project    string
	Status     string
	Limit      int
	Offset     int
}

// ListIssues returns issues matching the filters as JSON.
func (c Client) ListIssues(ctx context.Context, input IssueListInput) (json.RawMessage, error) {
	args := []string{"issue", "list", "--output", "json"}
	if input.AssigneeID != "" {
		args = append(args, "--assignee-id", input.AssigneeID)
	}
	if input.Project != "" {
		args = append(args, "--project", input.Project)
	}
	if input.Status != "" {
		args = append(args, "--status", input.Status)
	}
	if input.Limit > 0 {
		args = append(args, "--limit", fmt.Sprint(input.Limit))
	}
	if input.Offset > 0 {
		args = append(args, "--offset", fmt.Sprint(input.Offset))
	}
	return c.run(ctx, args, nil)
}

// IssueUpdateInput uses pointers so callers can distinguish an omitted field
// from an explicit empty value (which clears that field in Multica).
type IssueUpdateInput struct {
	Title       *string
	Description *string
	AssigneeID  *string
	ProjectID   *string
	Status      *string
	Priority    *string
	ParentID    *string
}

// UpdateIssue applies the fields present in input to an issue.
func (c Client) UpdateIssue(ctx context.Context, issueID string, input IssueUpdateInput) (json.RawMessage, error) {
	args := []string{"issue", "update", issueID, "--output", "json"}
	if input.Title != nil {
		args = append(args, "--title", *input.Title)
	}
	if input.Description != nil {
		args = append(args, "--description-stdin")
	}
	if input.AssigneeID != nil {
		args = append(args, "--assignee-id", *input.AssigneeID)
	}
	if input.ProjectID != nil {
		args = append(args, "--project", *input.ProjectID)
	}
	if input.Status != nil {
		args = append(args, "--status", *input.Status)
	}
	if input.Priority != nil {
		args = append(args, "--priority", *input.Priority)
	}
	if input.ParentID != nil {
		args = append(args, "--parent", *input.ParentID)
	}
	var stdin io.Reader
	if input.Description != nil {
		stdin = strings.NewReader(*input.Description)
	}
	return c.run(ctx, args, stdin)
}

// AddComment posts a comment on an issue. A non-empty parentID replies in that
// comment thread.
func (c Client) AddComment(ctx context.Context, issueID, content, parentID string) (json.RawMessage, error) {
	if strings.TrimSpace(content) == "" {
		return nil, fmt.Errorf("comment content is required")
	}
	args := []string{"issue", "comment", "add", issueID, "--content-stdin", "--output", "json"}
	if parentID != "" {
		args = append(args, "--parent", parentID)
	}
	return c.run(ctx, args, strings.NewReader(content))
}

// ListComments returns an issue's comments as JSON. rootsOnly returns top-level
// threads only; recent caps the number of threads when greater than zero.
func (c Client) ListComments(ctx context.Context, issueID string, rootsOnly bool, recent int) (json.RawMessage, error) {
	args := []string{"issue", "comment", "list", issueID, "--output", "json"}
	if rootsOnly {
		args = append(args, "--roots-only")
	}
	if recent > 0 {
		args = append(args, "--recent", fmt.Sprint(recent))
	}
	return c.run(ctx, args, nil)
}

// Run executes an arbitrary Multica CLI invocation and returns its JSON output.
// Tool handlers use it to reach commands beyond the typed helpers.
func (c Client) Run(ctx context.Context, args []string, stdin io.Reader) (json.RawMessage, error) {
	return c.run(ctx, args, stdin)
}

// RunCommand executes an arbitrary Multica CLI invocation and returns stdout as
// trimmed text. Use it for commands that do not support --output json.
func (c Client) RunCommand(ctx context.Context, args []string, stdin io.Reader) (string, error) {
	stdout, err := c.runBytes(ctx, args, stdin)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(stdout)), nil
}

func (c Client) run(ctx context.Context, args []string, stdin io.Reader) (json.RawMessage, error) {
	stdout, err := c.runBytes(ctx, args, stdin)
	if err != nil {
		return nil, err
	}
	return validateJSON(stdout)
}

func (c Client) runBytes(ctx context.Context, args []string, stdin io.Reader) ([]byte, error) {
	if c.Exec != nil {
		stdout, stderr, err := c.Exec(ctx, c.Command, args, stdin)
		if err != nil {
			message := strings.TrimSpace(string(stderr))
			if message == "" {
				message = err.Error()
			}
			return nil, fmt.Errorf("multica command failed: %s", message)
		}
		return stdout, nil
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
	return stdout.Bytes(), nil
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
