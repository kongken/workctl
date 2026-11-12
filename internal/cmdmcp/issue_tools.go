package cmdmcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	clierrors "github.com/kongken/workctl/internal/errors"
	"github.com/kongken/workctl/internal/mcpserver"
)

func (t toolset) issueTools() []mcpserver.Tool {
	return []mcpserver.Tool{
		{
			Name:        "assign_multica_issue",
			Description: "Assign a Multica issue to a member, agent, or squad, or unassign it.",
			InputSchema: schema(`{
				"type": "object",
				"properties": {
					"issue_id": {"type": "string", "description": "Multica issue identifier or UUID"},
					"to": {"type": "string", "description": "Assignee name (fuzzy match)"},
					"to_id": {"type": "string", "description": "Assignee UUID (mutually exclusive with to)"},
					"unassign": {"type": "boolean", "description": "Remove the current assignee"},
					"start": {"type": "boolean", "description": "Start an agent run on assignment (default true)"}
				},
				"required": ["issue_id"]
			}`),
			Write:   true,
			Handler: t.assignMulticaIssue,
		},
		{
			Name:        "set_multica_issue_status",
			Description: "Change a Multica issue's status, optionally marking it as a duplicate.",
			InputSchema: schema(`{
				"type": "object",
				"properties": {
					"issue_id": {"type": "string", "description": "Multica issue identifier or UUID"},
					"status": {"type": "string", "description": "Status key: backlog, todo, in_progress, in_review, done, blocked, cancelled"},
					"duplicate_of": {"type": "string", "description": "Original issue key or UUID; status must be cancelled"},
					"start": {"type": "boolean", "description": "Start an agent run on the transition (default true)"}
				},
				"required": ["issue_id", "status"]
			}`),
			Write:   true,
			Handler: t.setMulticaIssueStatus,
		},
		{
			Name:        "cancel_multica_issue_task",
			Description: "Cancel an in-progress or queued Multica run.",
			InputSchema: schema(`{
				"type": "object",
				"properties": {
					"run_id": {"type": "string", "description": "Run UUID or short ID"},
					"issue": {"type": "string", "description": "Issue ID/key to scope short run ID resolution"}
				},
				"required": ["run_id"]
			}`),
			Write:   true,
			Handler: t.cancelMulticaIssueTask,
		},
		{
			Name:        "list_multica_issue_children",
			Description: "List a Multica issue's sub-issues grouped by stage.",
			InputSchema: schema(`{
				"type": "object",
				"properties": {
					"issue_id": {"type": "string", "description": "Multica issue identifier or UUID"}
				},
				"required": ["issue_id"]
			}`),
			Handler: t.listMulticaIssueChildren,
		},
		{
			Name:        "manage_multica_issue_comment",
			Description: "Update, resolve, unresolve, or delete a Multica issue comment.",
			InputSchema: schema(`{
				"type": "object",
				"properties": {
					"comment_id": {"type": "string", "description": "Comment UUID"},
					"action": {"type": "string", "enum": ["update", "resolve", "unresolve", "delete"]},
					"content": {"type": "string", "description": "New content for update"},
					"expected_revision": {"type": "integer", "description": "Current positive comment revision (required for update)"}
				},
				"required": ["comment_id", "action"]
			}`),
			Write:   true,
			Handler: t.manageMulticaIssueComment,
		},
		{
			Name:        "manage_multica_issue_labels",
			Description: "List, attach, or remove labels on a Multica issue.",
			InputSchema: schema(`{
				"type": "object",
				"properties": {
					"issue_id": {"type": "string", "description": "Multica issue identifier or UUID"},
					"action": {"type": "string", "enum": ["list", "add", "remove"]},
					"label_id": {"type": "string", "description": "Label UUID for add/remove"}
				},
				"required": ["issue_id", "action"]
			}`),
			Write:   true,
			Handler: t.manageMulticaIssueLabels,
		},
		{
			Name:        "manage_multica_issue_metadata",
			Description: "List, get, set, or delete per-issue metadata (KV) on a Multica issue.",
			InputSchema: schema(`{
				"type": "object",
				"properties": {
					"issue_id": {"type": "string", "description": "Multica issue identifier or UUID"},
					"action": {"type": "string", "enum": ["list", "get", "set", "delete"]},
					"key": {"type": "string", "description": "Metadata key for get/set/delete"},
					"value": {"type": "string", "description": "Metadata value for set"},
					"value_type": {"type": "string", "enum": ["string", "number", "bool"], "description": "Force value type (default: auto-infer)"}
				},
				"required": ["issue_id", "action"]
			}`),
			Write:   true,
			Handler: t.manageMulticaIssueMetadata,
		},
		{
			Name:        "manage_multica_issue_property",
			Description: "List, set, or unset custom property values on a Multica issue.",
			InputSchema: schema(`{
				"type": "object",
				"properties": {
					"issue_id": {"type": "string", "description": "Multica issue identifier or UUID"},
					"action": {"type": "string", "enum": ["list", "set", "unset"]},
					"name": {"type": "string", "description": "Property name or UUID for set/unset"},
					"value": {"type": "string", "description": "Property value for set"}
				},
				"required": ["issue_id", "action"]
			}`),
			Write:   true,
			Handler: t.manageMulticaIssueProperty,
		},
		{
			Name:        "manage_multica_issue_subscribers",
			Description: "List, add, or remove subscribers on a Multica issue.",
			InputSchema: schema(`{
				"type": "object",
				"properties": {
					"issue_id": {"type": "string", "description": "Multica issue identifier or UUID"},
					"action": {"type": "string", "enum": ["list", "add", "remove"]},
					"user": {"type": "string", "description": "Member or agent name (fuzzy match; defaults to the caller)"},
					"user_id": {"type": "string", "description": "Member or agent UUID (mutually exclusive with user)"}
				},
				"required": ["issue_id", "action"]
			}`),
			Write:   true,
			Handler: t.manageMulticaIssueSubscribers,
		},
		{
			Name:        "list_multica_issue_pull_requests",
			Description: "List pull requests linked to a Multica issue.",
			InputSchema: schema(`{
				"type": "object",
				"properties": {
					"issue_id": {"type": "string", "description": "Multica issue identifier or UUID"}
				},
				"required": ["issue_id"]
			}`),
			Handler: t.listMulticaIssuePullRequests,
		},
		{
			Name:        "reorder_multica_issue",
			Description: "Move a Multica issue within its status column.",
			InputSchema: schema(`{
				"type": "object",
				"properties": {
					"issue_id": {"type": "string", "description": "Multica issue identifier or UUID"},
					"before": {"type": "string", "description": "Place directly above this issue (same column)"},
					"after": {"type": "string", "description": "Place directly below this issue (same column)"},
					"top": {"type": "boolean", "description": "Move to the top of its column"},
					"bottom": {"type": "boolean", "description": "Move to the bottom of its column"}
				},
				"required": ["issue_id"]
			}`),
			Write:   true,
			Handler: t.reorderMulticaIssue,
		},
		{
			Name:        "rerun_multica_issue",
			Description: "Re-enqueue a Multica issue's current agent assignment as a fresh run.",
			InputSchema: schema(`{
				"type": "object",
				"properties": {
					"issue_id": {"type": "string", "description": "Multica issue identifier or UUID"}
				},
				"required": ["issue_id"]
			}`),
			Write:   true,
			Handler: t.rerunMulticaIssue,
		},
		{
			Name:        "list_multica_issue_run_messages",
			Description: "List messages for a Multica execution.",
			InputSchema: schema(`{
				"type": "object",
				"properties": {
					"run_id": {"type": "string", "description": "Run UUID or short ID"},
					"issue": {"type": "string", "description": "Issue ID/key to scope short run ID resolution"},
					"since": {"type": "integer", "description": "Only return messages after this sequence number"}
				},
				"required": ["run_id"]
			}`),
			Handler: t.listMulticaIssueRunMessages,
		},
		{
			Name:        "list_multica_issue_runs",
			Description: "List execution history for a Multica issue.",
			InputSchema: schema(`{
				"type": "object",
				"properties": {
					"issue_id": {"type": "string", "description": "Multica issue identifier or UUID"},
					"active": {"type": "boolean", "description": "Only in-flight runs instead of the full history"},
					"siblings": {"type": "boolean", "description": "Widen to the issue's sub-issue family (implies active)"}
				},
				"required": ["issue_id"]
			}`),
			Handler: t.listMulticaIssueRuns,
		},
		{
			Name:        "search_multica_issues",
			Description: "Search Multica issues by title, description, or comments.",
			InputSchema: schema(`{
				"type": "object",
				"properties": {
					"query": {"type": "string", "description": "Search query"},
					"limit": {"type": "integer", "description": "Maximum number of results (default 20)"},
					"include_closed": {"type": "boolean", "description": "Include done and cancelled issues"}
				},
				"required": ["query"]
			}`),
			Handler: t.searchMulticaIssues,
		},
		{
			Name:        "get_multica_issue_timeline",
			Description: "Read the chronological history of a Multica issue (status/assignee changes, comments, runs).",
			InputSchema: schema(`{
				"type": "object",
				"properties": {
					"issue_id": {"type": "string", "description": "Multica issue identifier or UUID"},
					"activity_only": {"type": "boolean", "description": "Drop comments and return every activity record"},
					"actions": {"type": "array", "items": {"type": "string"}, "description": "Only return these actions (implies activity_only)"},
					"since": {"type": "string", "description": "Only return entries created after this RFC3339 timestamp"}
				},
				"required": ["issue_id"]
			}`),
			Handler: t.getMulticaIssueTimeline,
		},
		{
			Name:        "get_multica_issue_usage",
			Description: "Show aggregated token usage for a Multica issue.",
			InputSchema: schema(`{
				"type": "object",
				"properties": {
					"issue_id": {"type": "string", "description": "Multica issue identifier or UUID"}
				},
				"required": ["issue_id"]
			}`),
			Handler: t.getMulticaIssueUsage,
		},
		{
			Name:        "get_multica_issue_wakeups",
			Description: "Read a Multica issue's wakeups: list them, get one, list the runs a wakeup started, or list supported event types.",
			InputSchema: schema(`{
				"type": "object",
				"properties": {
					"issue_id": {"type": "string", "description": "Multica issue identifier or UUID (omit only for events)"},
					"action": {"type": "string", "enum": ["list", "get", "runs", "events"]},
					"wakeup_id": {"type": "string", "description": "Wakeup UUID for get/runs"}
				},
				"required": ["action"]
			}`),
			Handler: t.getMulticaIssueWakeups,
		},
		{
			Name:        "manage_multica_issue_wakeup",
			Description: "Create, update, disable, delete, trigger, or check in a Multica issue wakeup.",
			InputSchema: schema(`{
				"type": "object",
				"properties": {
					"issue_id": {"type": "string", "description": "Multica issue identifier or UUID"},
					"action": {"type": "string", "enum": ["create", "update", "disable", "delete", "trigger", "checkin"]},
					"wakeup_id": {"type": "string", "description": "Wakeup UUID (required except for create)"},
					"kind": {"type": "string", "enum": ["event", "at", "every", "cron"]},
					"mode": {"type": "string", "enum": ["once", "continuous"]},
					"at": {"type": "string", "description": "Single RFC3339 timestamp"},
					"after": {"type": "string", "description": "Delay for at, e.g. 10m"},
					"every": {"type": "string", "description": "Fixed interval, e.g. 1h (minimum 1m)"},
					"cron": {"type": "string", "description": "Five-field cron expression"},
					"timezone": {"type": "string", "description": "IANA timezone for cron (default UTC)"},
					"events": {"type": "array", "items": {"type": "string"}, "description": "Event types (see action events)"},
					"agent_id": {"type": "string", "description": "Agent to wake (defaults to the authenticated agent)"},
					"instruction": {"type": "string", "description": "Instruction for the next run"},
					"max_fires": {"type": "integer", "description": "Repeating rules: stop after this many runs (1-1000)"},
					"expires_at": {"type": "string", "description": "End the rule at this RFC3339 time"},
					"expires_in": {"type": "string", "description": "End the rule after this wait, e.g. 72h"},
					"on_timeout": {"type": "string", "enum": ["end", "wake"]},
					"until_status": {"type": "string", "description": "Condition: this issue's status becomes KEY"},
					"until_children_done": {"type": "boolean", "description": "Condition: every sub-issue is closed"},
					"stage": {"type": "integer", "description": "With until_children_done: wait for this stage and earlier ones"},
					"until_assignee": {"type": "string", "description": "Condition: this issue is assigned to member|agent|squad:ID"},
					"until_issue": {"type": "string", "description": "Condition: another issue reaches until_issue_state"},
					"until_issue_state": {"type": "string", "enum": ["done", "ended", "in_review", "blocked"]},
					"until_pr": {"type": "string", "enum": ["checks", "merged", "conflicts"], "description": "Condition: a linked pull request"},
					"until_label": {"type": "string", "description": "Condition: this issue has label LABEL_ID"},
					"until_property": {"type": "string", "description": "Condition: PROPERTY_ID=VALUE"},
					"filter_actor_type": {"type": "string", "enum": ["member", "agent"]},
					"filter_actor_id": {"type": "string", "description": "Actor UUID; requires filter_actor_type"},
					"parent": {"type": "string", "description": "Comment thread for result delivery"},
					"task_id": {"type": "string", "description": "Match this specific run"},
					"note": {"type": "string", "description": "Check-in note (checkin only, up to 500 characters)"}
				},
				"required": ["issue_id", "action"]
			}`),
			Write:   true,
			Handler: t.manageMulticaIssueWakeup,
		},
	}
}

func (t toolset) assignMulticaIssue(ctx context.Context, args json.RawMessage) (string, error) {
	var in struct {
		IssueID  string `json:"issue_id"`
		To       string `json:"to"`
		ToID     string `json:"to_id"`
		Unassign bool   `json:"unassign"`
		Start    *bool  `json:"start"`
	}
	if err := decodeArgs(args, &in); err != nil {
		return "", err
	}
	if strings.TrimSpace(in.IssueID) == "" {
		return "", clierrors.NewValidationError("issue_id is required")
	}
	argv := []string{"issue", "assign", in.IssueID}
	if in.To != "" {
		argv = append(argv, "--to", in.To)
	}
	if in.ToID != "" {
		argv = append(argv, "--to-id", in.ToID)
	}
	if in.Unassign {
		argv = append(argv, "--unassign")
	}
	if in.Start != nil && !*in.Start {
		argv = append(argv, "--no-start")
	}
	return t.call(ctx, argv, "", true)
}

func (t toolset) setMulticaIssueStatus(ctx context.Context, args json.RawMessage) (string, error) {
	var in struct {
		IssueID     string `json:"issue_id"`
		Status      string `json:"status"`
		DuplicateOf string `json:"duplicate_of"`
		Start       *bool  `json:"start"`
	}
	if err := decodeArgs(args, &in); err != nil {
		return "", err
	}
	if strings.TrimSpace(in.IssueID) == "" {
		return "", clierrors.NewValidationError("issue_id is required")
	}
	if strings.TrimSpace(in.Status) == "" {
		return "", clierrors.NewValidationError("status is required")
	}
	argv := []string{"issue", "status", in.IssueID, in.Status}
	if in.DuplicateOf != "" {
		argv = append(argv, "--duplicate-of", in.DuplicateOf)
	}
	if in.Start != nil && !*in.Start {
		argv = append(argv, "--no-start")
	}
	return t.call(ctx, argv, "", true)
}

func (t toolset) cancelMulticaIssueTask(ctx context.Context, args json.RawMessage) (string, error) {
	var in struct {
		RunID string `json:"run_id"`
		Issue string `json:"issue"`
	}
	if err := decodeArgs(args, &in); err != nil {
		return "", err
	}
	if strings.TrimSpace(in.RunID) == "" {
		return "", clierrors.NewValidationError("run_id is required")
	}
	argv := []string{"issue", "cancel-task", in.RunID}
	if in.Issue != "" {
		argv = append(argv, "--issue", in.Issue)
	}
	return t.call(ctx, argv, "", true)
}

func (t toolset) listMulticaIssueChildren(ctx context.Context, args json.RawMessage) (string, error) {
	issueID, err := issueIDArg(args)
	if err != nil {
		return "", err
	}
	return t.call(ctx, []string{"issue", "children", issueID}, "", true)
}

func (t toolset) manageMulticaIssueComment(ctx context.Context, args json.RawMessage) (string, error) {
	var in struct {
		CommentID        string `json:"comment_id"`
		Action           string `json:"action"`
		Content          string `json:"content"`
		ExpectedRevision *int   `json:"expected_revision"`
	}
	if err := decodeArgs(args, &in); err != nil {
		return "", err
	}
	if strings.TrimSpace(in.CommentID) == "" {
		return "", clierrors.NewValidationError("comment_id is required")
	}
	switch in.Action {
	case "delete":
		return t.call(ctx, []string{"issue", "comment", "delete", in.CommentID}, "", false)
	case "resolve", "unresolve":
		return t.call(ctx, []string{"issue", "comment", in.Action, in.CommentID}, "", true)
	case "update":
		if strings.TrimSpace(in.Content) == "" {
			return "", clierrors.NewValidationError("content is required for update")
		}
		if in.ExpectedRevision == nil {
			return "", clierrors.NewValidationError("expected_revision is required for update")
		}
		argv := []string{"issue", "comment", "update", in.CommentID, "--expected-revision", fmt.Sprint(*in.ExpectedRevision), "--content-stdin"}
		return t.call(ctx, argv, in.Content, true)
	default:
		return "", clierrors.NewValidationError("action must be one of update, resolve, unresolve, delete")
	}
}

func (t toolset) manageMulticaIssueLabels(ctx context.Context, args json.RawMessage) (string, error) {
	var in struct {
		IssueID string `json:"issue_id"`
		Action  string `json:"action"`
		LabelID string `json:"label_id"`
	}
	if err := decodeArgs(args, &in); err != nil {
		return "", err
	}
	if strings.TrimSpace(in.IssueID) == "" {
		return "", clierrors.NewValidationError("issue_id is required")
	}
	switch in.Action {
	case "list":
		return t.call(ctx, []string{"issue", "label", "list", in.IssueID}, "", true)
	case "add", "remove":
		if strings.TrimSpace(in.LabelID) == "" {
			return "", clierrors.NewValidationError("label_id is required for " + in.Action)
		}
		return t.call(ctx, []string{"issue", "label", in.Action, in.IssueID, in.LabelID}, "", true)
	default:
		return "", clierrors.NewValidationError("action must be one of list, add, remove")
	}
}

func (t toolset) manageMulticaIssueMetadata(ctx context.Context, args json.RawMessage) (string, error) {
	var in struct {
		IssueID   string `json:"issue_id"`
		Action    string `json:"action"`
		Key       string `json:"key"`
		Value     string `json:"value"`
		ValueType string `json:"value_type"`
	}
	if err := decodeArgs(args, &in); err != nil {
		return "", err
	}
	if strings.TrimSpace(in.IssueID) == "" {
		return "", clierrors.NewValidationError("issue_id is required")
	}
	switch in.Action {
	case "list":
		return t.call(ctx, []string{"issue", "metadata", "list", in.IssueID}, "", true)
	case "get", "delete":
		if strings.TrimSpace(in.Key) == "" {
			return "", clierrors.NewValidationError("key is required for " + in.Action)
		}
		return t.call(ctx, []string{"issue", "metadata", in.Action, in.IssueID, "--key", in.Key}, "", true)
	case "set":
		if strings.TrimSpace(in.Key) == "" {
			return "", clierrors.NewValidationError("key is required for set")
		}
		argv := []string{"issue", "metadata", "set", in.IssueID, "--key", in.Key, "--value", in.Value}
		if in.ValueType != "" {
			argv = append(argv, "--type", in.ValueType)
		}
		return t.call(ctx, argv, "", true)
	default:
		return "", clierrors.NewValidationError("action must be one of list, get, set, delete")
	}
}

func (t toolset) manageMulticaIssueProperty(ctx context.Context, args json.RawMessage) (string, error) {
	var in struct {
		IssueID string `json:"issue_id"`
		Action  string `json:"action"`
		Name    string `json:"name"`
		Value   string `json:"value"`
	}
	if err := decodeArgs(args, &in); err != nil {
		return "", err
	}
	if strings.TrimSpace(in.IssueID) == "" {
		return "", clierrors.NewValidationError("issue_id is required")
	}
	switch in.Action {
	case "list":
		return t.call(ctx, []string{"issue", "property", "list", in.IssueID}, "", true)
	case "set":
		if strings.TrimSpace(in.Name) == "" {
			return "", clierrors.NewValidationError("name is required for set")
		}
		return t.call(ctx, []string{"issue", "property", "set", in.IssueID, "--name", in.Name, "--value", in.Value}, "", true)
	case "unset":
		if strings.TrimSpace(in.Name) == "" {
			return "", clierrors.NewValidationError("name is required for unset")
		}
		return t.call(ctx, []string{"issue", "property", "unset", in.IssueID, "--name", in.Name}, "", true)
	default:
		return "", clierrors.NewValidationError("action must be one of list, set, unset")
	}
}

func (t toolset) manageMulticaIssueSubscribers(ctx context.Context, args json.RawMessage) (string, error) {
	var in struct {
		IssueID string `json:"issue_id"`
		Action  string `json:"action"`
		User    string `json:"user"`
		UserID  string `json:"user_id"`
	}
	if err := decodeArgs(args, &in); err != nil {
		return "", err
	}
	if strings.TrimSpace(in.IssueID) == "" {
		return "", clierrors.NewValidationError("issue_id is required")
	}
	switch in.Action {
	case "list":
		return t.call(ctx, []string{"issue", "subscriber", "list", in.IssueID}, "", true)
	case "add", "remove":
		argv := []string{"issue", "subscriber", in.Action, in.IssueID}
		if in.User != "" {
			argv = append(argv, "--user", in.User)
		}
		if in.UserID != "" {
			argv = append(argv, "--user-id", in.UserID)
		}
		return t.call(ctx, argv, "", true)
	default:
		return "", clierrors.NewValidationError("action must be one of list, add, remove")
	}
}

func (t toolset) listMulticaIssuePullRequests(ctx context.Context, args json.RawMessage) (string, error) {
	issueID, err := issueIDArg(args)
	if err != nil {
		return "", err
	}
	return t.call(ctx, []string{"issue", "pull-requests", issueID}, "", true)
}

func (t toolset) reorderMulticaIssue(ctx context.Context, args json.RawMessage) (string, error) {
	var in struct {
		IssueID string `json:"issue_id"`
		Before  string `json:"before"`
		After   string `json:"after"`
		Top     bool   `json:"top"`
		Bottom  bool   `json:"bottom"`
	}
	if err := decodeArgs(args, &in); err != nil {
		return "", err
	}
	if strings.TrimSpace(in.IssueID) == "" {
		return "", clierrors.NewValidationError("issue_id is required")
	}
	argv := []string{"issue", "reorder", in.IssueID}
	if in.Before != "" {
		argv = append(argv, "--before", in.Before)
	}
	if in.After != "" {
		argv = append(argv, "--after", in.After)
	}
	if in.Top {
		argv = append(argv, "--top")
	}
	if in.Bottom {
		argv = append(argv, "--bottom")
	}
	return t.call(ctx, argv, "", true)
}

func (t toolset) rerunMulticaIssue(ctx context.Context, args json.RawMessage) (string, error) {
	issueID, err := issueIDArg(args)
	if err != nil {
		return "", err
	}
	return t.call(ctx, []string{"issue", "rerun", issueID}, "", true)
}

func (t toolset) listMulticaIssueRunMessages(ctx context.Context, args json.RawMessage) (string, error) {
	var in struct {
		RunID string `json:"run_id"`
		Issue string `json:"issue"`
		Since int    `json:"since"`
	}
	if err := decodeArgs(args, &in); err != nil {
		return "", err
	}
	if strings.TrimSpace(in.RunID) == "" {
		return "", clierrors.NewValidationError("run_id is required")
	}
	argv := []string{"issue", "run-messages", in.RunID}
	if in.Issue != "" {
		argv = append(argv, "--issue", in.Issue)
	}
	if in.Since > 0 {
		argv = append(argv, "--since", fmt.Sprint(in.Since))
	}
	return t.call(ctx, argv, "", true)
}

func (t toolset) listMulticaIssueRuns(ctx context.Context, args json.RawMessage) (string, error) {
	var in struct {
		IssueID  string `json:"issue_id"`
		Active   bool   `json:"active"`
		Siblings bool   `json:"siblings"`
	}
	if err := decodeArgs(args, &in); err != nil {
		return "", err
	}
	if strings.TrimSpace(in.IssueID) == "" {
		return "", clierrors.NewValidationError("issue_id is required")
	}
	argv := []string{"issue", "runs", in.IssueID}
	if in.Active {
		argv = append(argv, "--active")
	}
	if in.Siblings {
		argv = append(argv, "--siblings")
	}
	return t.call(ctx, argv, "", true)
}

func (t toolset) searchMulticaIssues(ctx context.Context, args json.RawMessage) (string, error) {
	var in struct {
		Query         string `json:"query"`
		Limit         int    `json:"limit"`
		IncludeClosed bool   `json:"include_closed"`
	}
	if err := decodeArgs(args, &in); err != nil {
		return "", err
	}
	if strings.TrimSpace(in.Query) == "" {
		return "", clierrors.NewValidationError("query is required")
	}
	argv := []string{"issue", "search", in.Query}
	if in.Limit > 0 {
		argv = append(argv, "--limit", fmt.Sprint(in.Limit))
	}
	if in.IncludeClosed {
		argv = append(argv, "--include-closed")
	}
	return t.call(ctx, argv, "", true)
}

func (t toolset) getMulticaIssueTimeline(ctx context.Context, args json.RawMessage) (string, error) {
	var in struct {
		IssueID      string   `json:"issue_id"`
		ActivityOnly bool     `json:"activity_only"`
		Actions      []string `json:"actions"`
		Since        string   `json:"since"`
	}
	if err := decodeArgs(args, &in); err != nil {
		return "", err
	}
	if strings.TrimSpace(in.IssueID) == "" {
		return "", clierrors.NewValidationError("issue_id is required")
	}
	argv := []string{"issue", "timeline", in.IssueID}
	if in.ActivityOnly {
		argv = append(argv, "--activity-only")
	}
	if len(in.Actions) > 0 {
		argv = append(argv, "--action", strings.Join(in.Actions, ","))
	}
	if in.Since != "" {
		argv = append(argv, "--since", in.Since)
	}
	return t.call(ctx, argv, "", true)
}

func (t toolset) getMulticaIssueUsage(ctx context.Context, args json.RawMessage) (string, error) {
	issueID, err := issueIDArg(args)
	if err != nil {
		return "", err
	}
	return t.call(ctx, []string{"issue", "usage", issueID}, "", true)
}

func (t toolset) getMulticaIssueWakeups(ctx context.Context, args json.RawMessage) (string, error) {
	var in struct {
		IssueID  string `json:"issue_id"`
		Action   string `json:"action"`
		WakeupID string `json:"wakeup_id"`
	}
	if err := decodeArgs(args, &in); err != nil {
		return "", err
	}
	switch in.Action {
	case "events":
		return t.call(ctx, []string{"issue", "wakeup", "events"}, "", false)
	case "list":
		if strings.TrimSpace(in.IssueID) == "" {
			return "", clierrors.NewValidationError("issue_id is required for list")
		}
		return t.call(ctx, []string{"issue", "wakeup", "list", in.IssueID}, "", true)
	case "get", "runs":
		if strings.TrimSpace(in.IssueID) == "" {
			return "", clierrors.NewValidationError("issue_id is required for " + in.Action)
		}
		if strings.TrimSpace(in.WakeupID) == "" {
			return "", clierrors.NewValidationError("wakeup_id is required for " + in.Action)
		}
		return t.call(ctx, []string{"issue", "wakeup", in.Action, in.IssueID, in.WakeupID}, "", true)
	default:
		return "", clierrors.NewValidationError("action must be one of list, get, runs, events")
	}
}

type wakeupOptions struct {
	Kind              string   `json:"kind"`
	Mode              string   `json:"mode"`
	At                string   `json:"at"`
	After             string   `json:"after"`
	Every             string   `json:"every"`
	Cron              string   `json:"cron"`
	Timezone          string   `json:"timezone"`
	Events            []string `json:"events"`
	AgentID           string   `json:"agent_id"`
	Instruction       string   `json:"instruction"`
	MaxFires          *int32   `json:"max_fires"`
	ExpiresAt         string   `json:"expires_at"`
	ExpiresIn         string   `json:"expires_in"`
	OnTimeout         string   `json:"on_timeout"`
	UntilStatus       string   `json:"until_status"`
	UntilChildrenDone bool     `json:"until_children_done"`
	Stage             *int32   `json:"stage"`
	UntilAssignee     string   `json:"until_assignee"`
	UntilIssue        string   `json:"until_issue"`
	UntilIssueState   string   `json:"until_issue_state"`
	UntilPR           string   `json:"until_pr"`
	UntilLabel        string   `json:"until_label"`
	UntilProperty     string   `json:"until_property"`
	FilterActorType   string   `json:"filter_actor_type"`
	FilterActorID     string   `json:"filter_actor_id"`
	Parent            string   `json:"parent"`
	TaskID            string   `json:"task_id"`
}

func (in wakeupOptions) argv() []string {
	var argv []string
	add := func(flag, value string) {
		if value != "" {
			argv = append(argv, flag, value)
		}
	}
	add("--kind", in.Kind)
	add("--mode", in.Mode)
	add("--at", in.At)
	add("--after", in.After)
	add("--every", in.Every)
	add("--cron", in.Cron)
	add("--timezone", in.Timezone)
	if len(in.Events) > 0 {
		argv = append(argv, "--event", strings.Join(in.Events, ","))
	}
	add("--agent-id", in.AgentID)
	add("--instruction", in.Instruction)
	if in.MaxFires != nil {
		argv = append(argv, "--max-fires", fmt.Sprint(*in.MaxFires))
	}
	add("--expires-at", in.ExpiresAt)
	add("--expires-in", in.ExpiresIn)
	add("--on-timeout", in.OnTimeout)
	add("--until-status", in.UntilStatus)
	if in.UntilChildrenDone {
		argv = append(argv, "--until-children-done")
	}
	if in.Stage != nil {
		argv = append(argv, "--stage", fmt.Sprint(*in.Stage))
	}
	add("--until-assignee", in.UntilAssignee)
	add("--until-issue", in.UntilIssue)
	add("--until-issue-state", in.UntilIssueState)
	add("--until-pr", in.UntilPR)
	add("--until-label", in.UntilLabel)
	add("--until-property", in.UntilProperty)
	add("--filter-actor-type", in.FilterActorType)
	add("--filter-actor-id", in.FilterActorID)
	add("--parent", in.Parent)
	add("--task-id", in.TaskID)
	return argv
}

func (t toolset) manageMulticaIssueWakeup(ctx context.Context, args json.RawMessage) (string, error) {
	var in struct {
		wakeupOptions
		IssueID  string `json:"issue_id"`
		Action   string `json:"action"`
		WakeupID string `json:"wakeup_id"`
		Note     string `json:"note"`
	}
	if err := decodeArgs(args, &in); err != nil {
		return "", err
	}
	if strings.TrimSpace(in.IssueID) == "" {
		return "", clierrors.NewValidationError("issue_id is required")
	}
	switch in.Action {
	case "create":
		argv := append([]string{"issue", "wakeup", "create", in.IssueID}, in.wakeupOptions.argv()...)
		return t.call(ctx, argv, "", true)
	case "update":
		if strings.TrimSpace(in.WakeupID) == "" {
			return "", clierrors.NewValidationError("wakeup_id is required for update")
		}
		argv := append([]string{"issue", "wakeup", "update", in.IssueID, in.WakeupID}, in.wakeupOptions.argv()...)
		return t.call(ctx, argv, "", true)
	case "disable", "delete", "trigger":
		if strings.TrimSpace(in.WakeupID) == "" {
			return "", clierrors.NewValidationError("wakeup_id is required for " + in.Action)
		}
		return t.call(ctx, []string{"issue", "wakeup", in.Action, in.IssueID, in.WakeupID}, "", true)
	case "checkin":
		if strings.TrimSpace(in.WakeupID) == "" {
			return "", clierrors.NewValidationError("wakeup_id is required for checkin")
		}
		argv := []string{"issue", "wakeup", "checkin", in.IssueID, in.WakeupID}
		if in.Note != "" {
			argv = append(argv, "--note", in.Note)
		}
		return t.call(ctx, argv, "", false)
	default:
		return "", clierrors.NewValidationError("action must be one of create, update, disable, delete, trigger, checkin")
	}
}

func issueIDArg(args json.RawMessage) (string, error) {
	var in struct {
		IssueID string `json:"issue_id"`
	}
	if err := decodeArgs(args, &in); err != nil {
		return "", err
	}
	if strings.TrimSpace(in.IssueID) == "" {
		return "", clierrors.NewValidationError("issue_id is required")
	}
	return in.IssueID, nil
}
