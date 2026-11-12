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
			Name:        "get_multica_agent",
			Description: "Read one Multica agent by UUID.",
			InputSchema: schema(`{
				"type": "object",
				"properties": {
					"agent_id": {"type": "string", "description": "Multica agent UUID"}
				},
				"required": ["agent_id"]
			}`),
			Handler: t.getMulticaAgent,
		},
		{
			Name:        "create_multica_agent",
			Description: "Create a Multica agent on a runtime.",
			InputSchema: schema(`{
				"type": "object",
				"properties": {
					"name": {"type": "string", "description": "Agent name"},
					"runtime_id": {"type": "string", "description": "Runtime UUID the agent runs on"},
					"description": {"type": "string", "description": "Agent description"},
					"instructions": {"type": "string", "description": "Agent instructions"},
					"model": {"type": "string", "description": "Model identifier"},
					"max_concurrent_tasks": {"type": "integer", "description": "Maximum concurrent runs (1-50)"}
				},
				"required": ["name", "runtime_id"]
			}`),
			Write:   true,
			Handler: t.createMulticaAgent,
		},
		{
			Name:        "update_multica_agent",
			Description: "Update a Multica agent. Only the provided fields change; an empty string clears a field.",
			InputSchema: schema(`{
				"type": "object",
				"properties": {
					"agent_id": {"type": "string", "description": "Multica agent UUID"},
					"name": {"type": "string"},
					"description": {"type": "string"},
					"instructions": {"type": "string"},
					"model": {"type": "string"},
					"status": {"type": "string"},
					"runtime_id": {"type": "string"},
					"max_concurrent_tasks": {"type": "integer"}
				},
				"required": ["agent_id"]
			}`),
			Write:   true,
			Handler: t.updateMulticaAgent,
		},
		{
			Name:        "archive_multica_agent",
			Description: "Archive a Multica agent (reversible; use restore_multica_agent to undo).",
			InputSchema: schema(`{
				"type": "object",
				"properties": {
					"agent_id": {"type": "string", "description": "Multica agent UUID"}
				},
				"required": ["agent_id"]
			}`),
			Write:   true,
			Handler: t.archiveMulticaAgent,
		},
		{
			Name:        "restore_multica_agent",
			Description: "Restore an archived Multica agent.",
			InputSchema: schema(`{
				"type": "object",
				"properties": {
					"agent_id": {"type": "string", "description": "Multica agent UUID"}
				},
				"required": ["agent_id"]
			}`),
			Write:   true,
			Handler: t.restoreMulticaAgent,
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
			Write:   true,
			Handler: t.syncAgentPrompt,
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
		{
			Name:        "list_multica_issues",
			Description: "List Multica issues, optionally filtered by assignee, project, or status.",
			InputSchema: schema(`{
				"type": "object",
				"properties": {
					"assignee_id": {"type": "string", "description": "Assignee UUID (member, agent, or squad)"},
					"project": {"type": "string", "description": "Project UUID"},
					"status": {"type": "string", "description": "Status key, for example todo or in_progress"},
					"limit": {"type": "integer", "description": "Page size (1-100)"},
					"offset": {"type": "integer", "description": "Number of issues to skip"}
				}
			}`),
			Handler: t.listMulticaIssues,
		},
		{
			Name:        "create_multica_issue",
			Description: "Create a Multica issue, optionally assigned and linked to a project or parent.",
			InputSchema: schema(`{
				"type": "object",
				"properties": {
					"title": {"type": "string", "description": "Issue title"},
					"description": {"type": "string", "description": "Issue description (Markdown)"},
					"assignee_id": {"type": "string", "description": "Assignee UUID (member, agent, or squad)"},
					"project_id": {"type": "string", "description": "Project UUID"},
					"status": {"type": "string", "description": "Initial status key (default: Multica's own default)"},
					"priority": {"type": "string", "description": "Issue priority"},
					"parent_id": {"type": "string", "description": "Parent issue identifier or UUID"}
				},
				"required": ["title"]
			}`),
			Write:   true,
			Handler: t.createMulticaIssue,
		},
		{
			Name:        "update_multica_issue",
			Description: "Update a Multica issue. Only the provided fields change; an empty string clears a field.",
			InputSchema: schema(`{
				"type": "object",
				"properties": {
					"issue_id": {"type": "string", "description": "Multica issue identifier or UUID"},
					"title": {"type": "string"},
					"description": {"type": "string"},
					"assignee_id": {"type": "string"},
					"project_id": {"type": "string"},
					"status": {"type": "string"},
					"priority": {"type": "string"},
					"parent_id": {"type": "string"}
				},
				"required": ["issue_id"]
			}`),
			Write:   true,
			Handler: t.updateMulticaIssue,
		},
		{
			Name:        "add_multica_issue_comment",
			Description: "Post a comment on a Multica issue, optionally replying in a thread.",
			InputSchema: schema(`{
				"type": "object",
				"properties": {
					"issue_id": {"type": "string", "description": "Multica issue identifier or UUID"},
					"content": {"type": "string", "description": "Comment body (Markdown)"},
					"parent_id": {"type": "string", "description": "Comment UUID to reply under"}
				},
				"required": ["issue_id", "content"]
			}`),
			Write:   true,
			Handler: t.addMulticaIssueComment,
		},
		{
			Name:        "list_multica_issue_comments",
			Description: "List comments on a Multica issue.",
			InputSchema: schema(`{
				"type": "object",
				"properties": {
					"issue_id": {"type": "string", "description": "Multica issue identifier or UUID"},
					"roots_only": {"type": "boolean", "description": "Return only top-level threads"},
					"recent": {"type": "integer", "description": "Return only the N most recently active threads"}
				},
				"required": ["issue_id"]
			}`),
			Handler: t.listMulticaIssueComments,
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
			Write:   true,
			Handler: t.delegateToMultica,
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

func (t toolset) getMulticaAgent(ctx context.Context, args json.RawMessage) (string, error) {
	agentID, err := agentIDArg(args)
	if err != nil {
		return "", err
	}
	raw, err := t.multicacli().GetAgent(ctx, agentID)
	if err != nil {
		return "", err
	}
	return prettyJSON(raw)
}

func (t toolset) createMulticaAgent(ctx context.Context, args json.RawMessage) (string, error) {
	var in struct {
		Name               string `json:"name"`
		RuntimeID          string `json:"runtime_id"`
		Description        string `json:"description"`
		Instructions       string `json:"instructions"`
		Model              string `json:"model"`
		MaxConcurrentTasks *int32 `json:"max_concurrent_tasks"`
	}
	if err := decodeArgs(args, &in); err != nil {
		return "", err
	}
	if strings.TrimSpace(in.Name) == "" {
		return "", clierrors.NewValidationError("name is required")
	}
	if strings.TrimSpace(in.RuntimeID) == "" {
		return "", clierrors.NewValidationError("runtime_id is required")
	}
	raw, err := t.multicacli().CreateAgent(ctx, multicacli.AgentCreateInput{
		Name:               in.Name,
		RuntimeID:          in.RuntimeID,
		Description:        in.Description,
		Instructions:       in.Instructions,
		Model:              in.Model,
		MaxConcurrentTasks: in.MaxConcurrentTasks,
	})
	if err != nil {
		return "", err
	}
	return prettyJSON(raw)
}

func (t toolset) updateMulticaAgent(ctx context.Context, args json.RawMessage) (string, error) {
	var in struct {
		AgentID            string  `json:"agent_id"`
		Name               *string `json:"name"`
		Description        *string `json:"description"`
		Instructions       *string `json:"instructions"`
		Model              *string `json:"model"`
		Status             *string `json:"status"`
		RuntimeID          *string `json:"runtime_id"`
		MaxConcurrentTasks *int32  `json:"max_concurrent_tasks"`
	}
	if err := decodeArgs(args, &in); err != nil {
		return "", err
	}
	if strings.TrimSpace(in.AgentID) == "" {
		return "", clierrors.NewValidationError("agent_id is required")
	}
	raw, err := t.multicacli().UpdateAgent(ctx, in.AgentID, multicacli.AgentUpdateInput{
		Name:               in.Name,
		Description:        in.Description,
		Instructions:       in.Instructions,
		Model:              in.Model,
		Status:             in.Status,
		RuntimeID:          in.RuntimeID,
		MaxConcurrentTasks: in.MaxConcurrentTasks,
	})
	if err != nil {
		return "", err
	}
	return prettyJSON(raw)
}

func (t toolset) archiveMulticaAgent(ctx context.Context, args json.RawMessage) (string, error) {
	agentID, err := agentIDArg(args)
	if err != nil {
		return "", err
	}
	raw, err := t.multicacli().ArchiveAgent(ctx, agentID)
	if err != nil {
		return "", err
	}
	return prettyJSON(raw)
}

func (t toolset) restoreMulticaAgent(ctx context.Context, args json.RawMessage) (string, error) {
	agentID, err := agentIDArg(args)
	if err != nil {
		return "", err
	}
	raw, err := t.multicacli().RestoreAgent(ctx, agentID)
	if err != nil {
		return "", err
	}
	return prettyJSON(raw)
}

func agentIDArg(args json.RawMessage) (string, error) {
	var in struct {
		AgentID string `json:"agent_id"`
	}
	if err := decodeArgs(args, &in); err != nil {
		return "", err
	}
	if strings.TrimSpace(in.AgentID) == "" {
		return "", clierrors.NewValidationError("agent_id is required")
	}
	return in.AgentID, nil
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

func (t toolset) listMulticaIssues(ctx context.Context, args json.RawMessage) (string, error) {
	var in struct {
		AssigneeID string `json:"assignee_id"`
		Project    string `json:"project"`
		Status     string `json:"status"`
		Limit      int    `json:"limit"`
		Offset     int    `json:"offset"`
	}
	if err := decodeArgs(args, &in); err != nil {
		return "", err
	}
	raw, err := t.multicacli().ListIssues(ctx, multicacli.IssueListInput{
		AssigneeID: in.AssigneeID,
		Project:    in.Project,
		Status:     in.Status,
		Limit:      in.Limit,
		Offset:     in.Offset,
	})
	if err != nil {
		return "", err
	}
	return prettyJSON(raw)
}

func (t toolset) createMulticaIssue(ctx context.Context, args json.RawMessage) (string, error) {
	var in struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		AssigneeID  string `json:"assignee_id"`
		ProjectID   string `json:"project_id"`
		Status      string `json:"status"`
		Priority    string `json:"priority"`
		ParentID    string `json:"parent_id"`
	}
	if err := decodeArgs(args, &in); err != nil {
		return "", err
	}
	if strings.TrimSpace(in.Title) == "" {
		return "", clierrors.NewValidationError("title is required")
	}
	raw, err := t.multicacli().CreateIssue(ctx, multicacli.CreateIssueInput{
		Title:       in.Title,
		Description: in.Description,
		AssigneeID:  in.AssigneeID,
		ProjectID:   in.ProjectID,
		Status:      in.Status,
		Priority:    in.Priority,
		ParentID:    in.ParentID,
	})
	if err != nil {
		return "", err
	}
	return prettyJSON(raw)
}

func (t toolset) updateMulticaIssue(ctx context.Context, args json.RawMessage) (string, error) {
	var in struct {
		IssueID     string  `json:"issue_id"`
		Title       *string `json:"title"`
		Description *string `json:"description"`
		AssigneeID  *string `json:"assignee_id"`
		ProjectID   *string `json:"project_id"`
		Status      *string `json:"status"`
		Priority    *string `json:"priority"`
		ParentID    *string `json:"parent_id"`
	}
	if err := decodeArgs(args, &in); err != nil {
		return "", err
	}
	if strings.TrimSpace(in.IssueID) == "" {
		return "", clierrors.NewValidationError("issue_id is required")
	}
	raw, err := t.multicacli().UpdateIssue(ctx, in.IssueID, multicacli.IssueUpdateInput{
		Title:       in.Title,
		Description: in.Description,
		AssigneeID:  in.AssigneeID,
		ProjectID:   in.ProjectID,
		Status:      in.Status,
		Priority:    in.Priority,
		ParentID:    in.ParentID,
	})
	if err != nil {
		return "", err
	}
	return prettyJSON(raw)
}

func (t toolset) addMulticaIssueComment(ctx context.Context, args json.RawMessage) (string, error) {
	var in struct {
		IssueID  string `json:"issue_id"`
		Content  string `json:"content"`
		ParentID string `json:"parent_id"`
	}
	if err := decodeArgs(args, &in); err != nil {
		return "", err
	}
	if strings.TrimSpace(in.IssueID) == "" {
		return "", clierrors.NewValidationError("issue_id is required")
	}
	raw, err := t.multicacli().AddComment(ctx, in.IssueID, in.Content, in.ParentID)
	if err != nil {
		return "", err
	}
	return prettyJSON(raw)
}

func (t toolset) listMulticaIssueComments(ctx context.Context, args json.RawMessage) (string, error) {
	var in struct {
		IssueID   string `json:"issue_id"`
		RootsOnly bool   `json:"roots_only"`
		Recent    int    `json:"recent"`
	}
	if err := decodeArgs(args, &in); err != nil {
		return "", err
	}
	if strings.TrimSpace(in.IssueID) == "" {
		return "", clierrors.NewValidationError("issue_id is required")
	}
	raw, err := t.multicacli().ListComments(ctx, in.IssueID, in.RootsOnly, in.Recent)
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
