# workctl

Command-line client for Linear and Plane, written in Go. `workctl` is the only executable: Linear commands live at the root and Plane commands under `plane`.

Plane support is a preview validated against local API fixtures. Authenticated Cloud and self-hosted deployment validation is still pending; see the [compatibility matrix](docs/compatibility.md).

## Build

```bash
go build -o workctl ./cmd/workctl
```

The installation path is:

```bash
go install github.com/kongken/workctl/cmd/workctl@latest
```

Linear commands are available directly for compatibility, or explicitly under `linear` when using the same grouped shape as Plane:

```bash
export LINEAR_API_KEY=lin_api_...
# or: workctl auth login

workctl issue mine
workctl linear issue view ENG-123 --json
```

## Plane

```bash
# Prompts for a personal access token without displaying it.
workctl plane auth login --profile cloud --workspace my-workspace

workctl plane project list --json
workctl plane --project PROJECT_UUID issue list
workctl plane issue view ENG-123 --json
workctl plane --project PROJECT_UUID issue create --title 'Fix login' --priority high
workctl plane issue comment add ENG-123 --body 'Ready for review'
```

Use `--api-version v1` with the explicit instance URL for a self-hosted v1 deployment. [Plane usage](docs/plane.md) covers authentication, work items, comments, resource management, attachments, and queries. [Configuration](docs/configuration.md) describes profile selection and credential isolation; [migration](docs/migration.md) covers existing scripts and rollback.

## Multica delegation

`workctl` can hand a Linear issue to a Multica agent through the installed
`multica` CLI. Keep the agent instructions in a local, versioned file and sync
them before delegation:

```bash
workctl multica prompt sync AGENT_UUID --file .workctl/agent-prompts/default.md
workctl multica delegate ENG-123 \
  --agent-id AGENT_UUID \
  --project PROJECT_UUID \
  --prompt-file .workctl/agent-prompts/default.md
```

The same integration exposes agent CRUD operations. Agent deletion maps to
Multica's archive operation, so it can be reversed with `restore`:

```bash
workctl multica agent list --include-archived
workctl multica agent get AGENT_UUID
workctl multica agent create --name Builder --runtime-id RUNTIME_UUID \
  --instructions-file .workctl/agent-prompts/default.md
workctl multica agent update AGENT_UUID \
  --instructions-file .workctl/agent-prompts/default.md
workctl multica agent delete AGENT_UUID
workctl multica agent restore AGENT_UUID
```

These commands return the JSON produced by the Multica CLI. Use
`--multica-command` on any command to point workctl at a wrapper or a local
test executable.

The delegate command reads the Linear issue through workctl's existing GraphQL
client, updates the selected Multica agent when `--prompt-file` is provided,
then creates an assigned Multica issue with the Linear title, description, and
source URL. It emits a JSON mapping containing both issue identifiers. The
Multica executable can be overridden with `--multica-command` for a local test
fixture or wrapper.

This is a synchronous handoff. It does not poll Multica runs or write status
back to Linear; a durable bridge can build on the emitted mapping. See
[the integration research](docs/research/workctl-multica-integration.md) for
the adapter seams, prompt rules, and follow-up work.

## MCP server

`workctl mcp serve` exposes Linear reads plus Multica agent and issue
management as Model Context Protocol tools over stdio, so an MCP client can
read a Linear issue, create or update agents, manage issues and comments, sync
an agent prompt, and dispatch the work:

```bash
workctl mcp serve
workctl mcp serve --read-only   # expose only tools that read state
```

Tools: `get_linear_issue`; agent tools `list_multica_agents`,
`get_multica_agent`, `create_multica_agent`, `update_multica_agent`,
`archive_multica_agent`, `restore_multica_agent`, `sync_agent_prompt`,
`set_multica_agent_avatar`, `copy_multica_agent`, `get_multica_agent_env`,
`set_multica_agent_env`, `manage_multica_agent_mcp`,
`manage_multica_agent_skills`, `list_multica_agent_tasks`; issue tools
`get_multica_work_status`, `list_multica_issues`, `create_multica_issue`,
`update_multica_issue`, `add_multica_issue_comment`,
`list_multica_issue_comments`, `assign_multica_issue`,
`set_multica_issue_status`, `cancel_multica_issue_task`,
`list_multica_issue_children`, `manage_multica_issue_comment`,
`manage_multica_issue_labels`, `manage_multica_issue_metadata`,
`manage_multica_issue_property`, `manage_multica_issue_subscribers`,
`list_multica_issue_pull_requests`, `reorder_multica_issue`,
`rerun_multica_issue`, `list_multica_issue_run_messages`,
`list_multica_issue_runs`, `search_multica_issues`,
`get_multica_issue_timeline`, `get_multica_issue_usage`,
`get_multica_issue_wakeups`, `manage_multica_issue_wakeup`; and
`delegate_to_multica`. Multica operations
use the installed `multica` CLI (override with `--multica-command`). See [the
MCP guide](docs/mcp.md) for client configuration and tool parameters.

## Develop

```bash
go test ./...
go vet ./...
go test -race ./internal/plane/... ./internal/planeconfig ./internal/planeauth ./internal/cmdplane ./internal/workcli
go generate ./internal/gql/
go build -o workctl ./cmd/workctl
```

GraphQL operations remain in `internal/gql/queries/`; regenerate after changing them. Plane uses a separate REST client with internal v1/v2 adapters and `httptest` fixtures. Regular tests do not contact Plane or use real credentials.

- [API research](docs/research/plane-api.md)
- [Implementation plan](docs/plans/plane-support.md)
- [Current capability and validation matrix](docs/compatibility.md)
