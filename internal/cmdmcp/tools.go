package cmdmcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kongken/workctl/internal/delegation"
	clierrors "github.com/kongken/workctl/internal/errors"
	"github.com/kongken/workctl/internal/linear"
	"github.com/kongken/workctl/internal/mcpserver"
	"github.com/kongken/workctl/internal/multicacli"
)

// toolset holds the shared configuration and injected clients for the MCP tool
// handlers. The function fields keep the handlers testable without a real
// Multica executable or Linear endpoint.
type toolset struct {
	multicaCommand string
	client         func() multicacli.Client
	service        func() delegation.Service
}

func (t toolset) multicacli() multicacli.Client {
	if t.client != nil {
		return t.client()
	}
	return multicacli.New(t.multicaCommand)
}

func (t toolset) delegationService() delegation.Service {
	if t.service != nil {
		return t.service()
	}
	return delegation.New(t.multicaCommand)
}

func (t toolset) tools() []mcpserver.Tool {
	return []mcpserver.Tool{
		{
			Name:        "get_linear_issue",
			Description: "Read a Linear issue by identifier or UUID and return its JSON fields.",
			InputSchema: schema(`{
				"type": "object",
				"properties": {
					"issue_id": {"type": "string", "description": "Linear issue identifier (for example ENG-123) or UUID"}
				},
				"required": ["issue_id"]
			}`),
			Handler: t.getLinearIssue,
		},
		{
			Name:        "list_multica_agents",
			Description: "List Multica agents available in the selected workspace.",
			InputSchema: schema(`{
				"type": "object",
				"properties": {
					"include_archived": {"type": "boolean", "description": "Include archived agents"}
				}
			}`),
			Handler: t.listMulticaAgents,
		},
		{
			Name:        "sync_agent_prompt",
			Description: "Sync a local UTF-8 prompt file to a Multica agent's instructions.",
			InputSchema: schema(`{
				"type": "object",
				"properties": {
					"agent_id": {"type": "string", "description": "Multica agent UUID"},
					"prompt_file": {"type": "string", "description": "Path to the local UTF-8 prompt file"}
				},
				"required": ["agent_id", "prompt_file"]
			}`),
			Handler: t.syncAgentPrompt,
		},
		{
			Name:        "delegate_to_multica",
			Description: "Create a Multica issue from a Linear issue and assign it to a Multica agent. Optionally sync a local prompt file first.",
			InputSchema: schema(`{
				"type": "object",
				"properties": {
					"issue": {"type": "string", "description": "Linear issue identifier (for example ENG-123)"},
					"agent_id": {"type": "string", "description": "Multica agent UUID that receives the work"},
					"project_id": {"type": "string", "description": "Optional Multica project UUID"},
					"status": {"type": "string", "description": "Initial Multica issue status (default: todo)"},
					"prompt_file": {"type": "string", "description": "Optional local prompt file to sync before delegating"}
				},
				"required": ["issue", "agent_id"]
			}`),
			Handler: t.delegateToMultica,
		},
		{
			Name:        "get_multica_work_status",
			Description: "Read a Multica issue by identifier or UUID to check delegated work status.",
			InputSchema: schema(`{
				"type": "object",
				"properties": {
					"issue_id": {"type": "string", "description": "Multica issue identifier or UUID"}
				},
				"required": ["issue_id"]
			}`),
			Handler: t.getMulticaWorkStatus,
		},
	}
}

func (t toolset) getLinearIssue(ctx context.Context, args json.RawMessage) (string, error) {
	var in struct {
		IssueID string `json:"issue_id"`
	}
	if err := decodeArgs(args, &in); err != nil {
		return "", err
	}
	if strings.TrimSpace(in.IssueID) == "" {
		return "", clierrors.NewValidationError("issue_id is required")
	}
	issueID, err := linear.IssueIdentifier(in.IssueID)
	if err != nil {
		return "", err
	}
	if issueID == "" {
		return "", clierrors.NewValidationError("could not determine Linear issue identifier")
	}
	raw, err := linear.FetchIssueDetailsJSON(ctx, issueID)
	if err != nil {
		return "", err
	}
	return prettyJSON(raw)
}

func (t toolset) listMulticaAgents(ctx context.Context, args json.RawMessage) (string, error) {
	var in struct {
		IncludeArchived bool `json:"include_archived"`
	}
	if err := decodeArgs(args, &in); err != nil {
		return "", err
	}
	raw, err := t.multicacli().ListAgents(ctx, in.IncludeArchived)
	if err != nil {
		return "", err
	}
	return prettyJSON(raw)
}

func (t toolset) syncAgentPrompt(ctx context.Context, args json.RawMessage) (string, error) {
	var in struct {
		AgentID    string `json:"agent_id"`
		PromptFile string `json:"prompt_file"`
	}
	if err := decodeArgs(args, &in); err != nil {
		return "", err
	}
	if strings.TrimSpace(in.AgentID) == "" {
		return "", clierrors.NewValidationError("agent_id is required")
	}
	if strings.TrimSpace(in.PromptFile) == "" {
		return "", clierrors.NewValidationError("prompt_file is required")
	}
	instructions, err := delegation.ReadPromptFile(in.PromptFile)
	if err != nil {
		return "", err
	}
	raw, err := t.multicacli().UpdateAgentInstructions(ctx, in.AgentID, instructions)
	if err != nil {
		return "", err
	}
	return prettyJSON(raw)
}

func (t toolset) delegateToMultica(ctx context.Context, args json.RawMessage) (string, error) {
	var in struct {
		Issue      string `json:"issue"`
		AgentID    string `json:"agent_id"`
		ProjectID  string `json:"project_id"`
		Status     string `json:"status"`
		PromptFile string `json:"prompt_file"`
	}
	if err := decodeArgs(args, &in); err != nil {
		return "", err
	}
	if strings.TrimSpace(in.Issue) == "" {
		return "", clierrors.NewValidationError("issue is required")
	}
	if strings.TrimSpace(in.AgentID) == "" {
		return "", clierrors.NewValidationError("agent_id is required")
	}
	if strings.TrimSpace(in.Status) == "" {
		in.Status = "todo"
	}
	result, err := t.delegationService().Delegate(ctx, delegation.DelegateInput{
		Issue:      in.Issue,
		AgentID:    in.AgentID,
		ProjectID:  in.ProjectID,
		Status:     in.Status,
		PromptFile: in.PromptFile,
	})
	if err != nil {
		return "", err
	}
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (t toolset) getMulticaWorkStatus(ctx context.Context, args json.RawMessage) (string, error) {
	var in struct {
		IssueID string `json:"issue_id"`
	}
	if err := decodeArgs(args, &in); err != nil {
		return "", err
	}
	if strings.TrimSpace(in.IssueID) == "" {
		return "", clierrors.NewValidationError("issue_id is required")
	}
	raw, err := t.multicacli().GetIssue(ctx, in.IssueID)
	if err != nil {
		return "", err
	}
	return prettyJSON(raw)
}

func decodeArgs(args json.RawMessage, target any) error {
	if len(bytes.TrimSpace(args)) == 0 {
		return nil
	}
	if err := json.Unmarshal(args, target); err != nil {
		return clierrors.NewValidationError(fmt.Sprintf("invalid tool arguments: %v", err))
	}
	return nil
}

func prettyJSON(raw json.RawMessage) (string, error) {
	var buf bytes.Buffer
	if err := json.Indent(&buf, raw, "", "  "); err != nil {
		return "", fmt.Errorf("format JSON: %w", err)
	}
	return buf.String(), nil
}

func schema(raw string) json.RawMessage {
	return json.RawMessage(raw)
}
