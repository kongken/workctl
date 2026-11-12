package cmdmcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	clierrors "github.com/kongken/workctl/internal/errors"
	"github.com/kongken/workctl/internal/mcpserver"
)

func (t toolset) agentTools() []mcpserver.Tool {
	return []mcpserver.Tool{
		{
			Name:        "set_multica_agent_avatar",
			Description: "Upload an avatar image for a Multica agent.",
			InputSchema: schema(`{
				"type": "object",
				"properties": {
					"agent_id": {"type": "string", "description": "Multica agent UUID"},
					"file": {"type": "string", "description": "Path to the avatar image file"}
				},
				"required": ["agent_id", "file"]
			}`),
			Write:   true,
			Handler: t.setMulticaAgentAvatar,
		},
		{
			Name:        "copy_multica_agent",
			Description: "Copy a Multica agent into a new agent, optionally on a different runtime.",
			InputSchema: schema(`{
				"type": "object",
				"properties": {
					"source_agent_id": {"type": "string", "description": "Agent UUID to copy"},
					"name": {"type": "string", "description": "Name for the new agent"},
					"runtime_id": {"type": "string", "description": "Target runtime UUID (default: same as source)"},
					"description": {"type": "string"},
					"instructions": {"type": "string"},
					"model": {"type": "string", "description": "Required when runtime_id changes the runtime"},
					"max_concurrent_tasks": {"type": "integer"},
					"no_skills": {"type": "boolean", "description": "Do not copy the source agent's skill assignments"}
				},
				"required": ["source_agent_id"]
			}`),
			Write:   true,
			Handler: t.copyMulticaAgent,
		},
		{
			Name:        "get_multica_agent_env",
			Description: "Read a Multica agent's custom environment variables (audited).",
			InputSchema: schema(`{
				"type": "object",
				"properties": {
					"agent_id": {"type": "string", "description": "Multica agent UUID"}
				},
				"required": ["agent_id"]
			}`),
			Handler: t.getMulticaAgentEnv,
		},
		{
			Name:        "set_multica_agent_env",
			Description: "Replace a Multica agent's custom environment variables. Pass a JSON object string; values equal to **** preserve existing entries.",
			InputSchema: schema(`{
				"type": "object",
				"properties": {
					"agent_id": {"type": "string", "description": "Multica agent UUID"},
					"env": {"type": "string", "description": "Replacement custom_env as a JSON object, e.g. {\"KEY\":\"value\"}"}
				},
				"required": ["agent_id", "env"]
			}`),
			Write:   true,
			Handler: t.setMulticaAgentEnv,
		},
		{
			Name:        "manage_multica_agent_mcp",
			Description: "List, add, remove, enable, or disable the workspace MCP servers assigned to a Multica agent.",
			InputSchema: schema(`{
				"type": "object",
				"properties": {
					"agent_id": {"type": "string", "description": "Multica agent UUID"},
					"action": {"type": "string", "enum": ["list", "add", "remove", "enable", "disable"]},
					"server_id": {"type": "string", "description": "Workspace MCP server UUID (required except for list)"}
				},
				"required": ["agent_id", "action"]
			}`),
			Write:   true,
			Handler: t.manageMulticaAgentMCP,
		},
		{
			Name:        "manage_multica_agent_skills",
			Description: "List, add, or replace the skills assigned to a Multica agent.",
			InputSchema: schema(`{
				"type": "object",
				"properties": {
					"agent_id": {"type": "string", "description": "Multica agent UUID"},
					"action": {"type": "string", "enum": ["list", "add", "set"]},
					"skill_ids": {"type": "array", "items": {"type": "string"}, "description": "Skill IDs for add/set"}
				},
				"required": ["agent_id", "action"]
			}`),
			Write:   true,
			Handler: t.manageMulticaAgentSkills,
		},
		{
			Name:        "list_multica_agent_tasks",
			Description: "List runs for a Multica agent.",
			InputSchema: schema(`{
				"type": "object",
				"properties": {
					"agent_id": {"type": "string", "description": "Multica agent UUID"},
					"limit": {"type": "integer", "description": "Maximum runs per page (1-200)"},
					"before": {"type": "string", "description": "Cursor from the previous page"}
				},
				"required": ["agent_id"]
			}`),
			Handler: t.listMulticaAgentTasks,
		},
	}
}

func (t toolset) setMulticaAgentAvatar(ctx context.Context, args json.RawMessage) (string, error) {
	var in struct {
		AgentID string `json:"agent_id"`
		File    string `json:"file"`
	}
	if err := decodeArgs(args, &in); err != nil {
		return "", err
	}
	if strings.TrimSpace(in.AgentID) == "" {
		return "", clierrors.NewValidationError("agent_id is required")
	}
	if strings.TrimSpace(in.File) == "" {
		return "", clierrors.NewValidationError("file is required")
	}
	return t.call(ctx, []string{"agent", "avatar", in.AgentID, "--file", in.File}, "", true)
}

func (t toolset) copyMulticaAgent(ctx context.Context, args json.RawMessage) (string, error) {
	var in struct {
		SourceAgentID      string `json:"source_agent_id"`
		Name               string `json:"name"`
		RuntimeID          string `json:"runtime_id"`
		Description        string `json:"description"`
		Instructions       string `json:"instructions"`
		Model              string `json:"model"`
		MaxConcurrentTasks *int32 `json:"max_concurrent_tasks"`
		NoSkills           bool   `json:"no_skills"`
	}
	if err := decodeArgs(args, &in); err != nil {
		return "", err
	}
	if strings.TrimSpace(in.SourceAgentID) == "" {
		return "", clierrors.NewValidationError("source_agent_id is required")
	}
	argv := []string{"agent", "copy", in.SourceAgentID}
	if in.Name != "" {
		argv = append(argv, "--name", in.Name)
	}
	if in.RuntimeID != "" {
		argv = append(argv, "--runtime-id", in.RuntimeID)
	}
	if in.Description != "" {
		argv = append(argv, "--description", in.Description)
	}
	if in.Instructions != "" {
		argv = append(argv, "--instructions", in.Instructions)
	}
	if in.Model != "" {
		argv = append(argv, "--model", in.Model)
	}
	if in.MaxConcurrentTasks != nil {
		argv = append(argv, "--max-concurrent-tasks", fmt.Sprint(*in.MaxConcurrentTasks))
	}
	if in.NoSkills {
		argv = append(argv, "--no-skills")
	}
	return t.call(ctx, argv, "", true)
}

func (t toolset) getMulticaAgentEnv(ctx context.Context, args json.RawMessage) (string, error) {
	agentID, err := agentIDArg(args)
	if err != nil {
		return "", err
	}
	return t.call(ctx, []string{"agent", "env", "get", agentID}, "", true)
}

func (t toolset) setMulticaAgentEnv(ctx context.Context, args json.RawMessage) (string, error) {
	var in struct {
		AgentID string `json:"agent_id"`
		Env     string `json:"env"`
	}
	if err := decodeArgs(args, &in); err != nil {
		return "", err
	}
	if strings.TrimSpace(in.AgentID) == "" {
		return "", clierrors.NewValidationError("agent_id is required")
	}
	if strings.TrimSpace(in.Env) == "" {
		return "", clierrors.NewValidationError("env is required")
	}
	if !json.Valid([]byte(in.Env)) {
		return "", clierrors.NewValidationError("env must be a JSON object")
	}
	return t.call(ctx, []string{"agent", "env", "set", in.AgentID, "--custom-env", in.Env}, "", true)
}

func (t toolset) manageMulticaAgentMCP(ctx context.Context, args json.RawMessage) (string, error) {
	var in struct {
		AgentID  string `json:"agent_id"`
		Action   string `json:"action"`
		ServerID string `json:"server_id"`
	}
	if err := decodeArgs(args, &in); err != nil {
		return "", err
	}
	if strings.TrimSpace(in.AgentID) == "" {
		return "", clierrors.NewValidationError("agent_id is required")
	}
	switch in.Action {
	case "list":
		return t.call(ctx, []string{"agent", "mcp", "list", in.AgentID}, "", true)
	case "add", "remove", "enable", "disable":
		if strings.TrimSpace(in.ServerID) == "" {
			return "", clierrors.NewValidationError("server_id is required for " + in.Action)
		}
		return t.call(ctx, []string{"agent", "mcp", in.Action, in.AgentID, in.ServerID}, "", true)
	default:
		return "", clierrors.NewValidationError("action must be one of list, add, remove, enable, disable")
	}
}

func (t toolset) manageMulticaAgentSkills(ctx context.Context, args json.RawMessage) (string, error) {
	var in struct {
		AgentID  string   `json:"agent_id"`
		Action   string   `json:"action"`
		SkillIDs []string `json:"skill_ids"`
	}
	if err := decodeArgs(args, &in); err != nil {
		return "", err
	}
	if strings.TrimSpace(in.AgentID) == "" {
		return "", clierrors.NewValidationError("agent_id is required")
	}
	switch in.Action {
	case "list":
		return t.call(ctx, []string{"agent", "skills", "list", in.AgentID}, "", true)
	case "add", "set":
		if len(in.SkillIDs) == 0 {
			return "", clierrors.NewValidationError("skill_ids is required for " + in.Action)
		}
		return t.call(ctx, []string{"agent", "skills", in.Action, in.AgentID, "--skill-ids", strings.Join(in.SkillIDs, ",")}, "", true)
	default:
		return "", clierrors.NewValidationError("action must be one of list, add, set")
	}
}

func (t toolset) listMulticaAgentTasks(ctx context.Context, args json.RawMessage) (string, error) {
	var in struct {
		AgentID string `json:"agent_id"`
		Limit   int    `json:"limit"`
		Before  string `json:"before"`
	}
	if err := decodeArgs(args, &in); err != nil {
		return "", err
	}
	if strings.TrimSpace(in.AgentID) == "" {
		return "", clierrors.NewValidationError("agent_id is required")
	}
	argv := []string{"agent", "tasks", in.AgentID}
	if in.Limit > 0 {
		argv = append(argv, "--limit", fmt.Sprint(in.Limit))
	}
	if in.Before != "" {
		argv = append(argv, "--before", in.Before)
	}
	return t.call(ctx, argv, "", true)
}
