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
