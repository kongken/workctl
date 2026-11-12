# MCP 服务

`workctl mcp serve` 在 stdio 上启动一个 Model Context Protocol 服务，把 Linear 读取与 Multica 委派桥接暴露为 MCP tools，便于其他 MCP client（例如读取 Linear issue 的 agent）发现并派发 Multica 开发任务。

服务只做同步桥接：读取 Linear issue、按需同步本地 prompt 文件、创建并分配 Multica issue，然后返回映射结果。它不会轮询 Multica run，也不会把状态回写 Linear。

## 启动

```bash
workctl mcp serve
```

MCP client 使用 stdio 传输启动该命令即可。示例配置：

```json
{
  "mcpServers": {
    "workctl": {
      "command": "workctl",
      "args": ["mcp", "serve"]
    }
  }
}
```

Multica 相关 tool 通过本机已安装的 `multica` CLI 执行，因此请先 `multica login` 并选定 workspace；也可以用 `--multica-command` 指向包装脚本或测试用可执行文件。Linear 读取沿用现有凭据（`LINEAR_API_KEY`、项目配置或 `workctl auth login`）。

```bash
workctl mcp serve --multica-command /usr/local/bin/multica
```

## Tools

| Tool | 参数 | 说明 |
| --- | --- | --- |
| `get_linear_issue` | `issue_id`（必需） | 按 identifier（如 `ENG-123`）或 UUID 读取 Linear issue 的 JSON 字段。 |
| `list_multica_agents` | `include_archived`（可选） | 列出当前 Multica workspace 的 agent。 |
| `sync_agent_prompt` | `agent_id`、`prompt_file`（必需） | 读取本地 UTF-8 prompt 文件并更新指定 agent 的 instructions。 |
| `delegate_to_multica` | `issue`、`agent_id`（必需）；`project_id`、`status`、`prompt_file`（可选） | 读取 Linear issue，可选先同步 prompt，再创建并分配给 Multica agent 的 issue。 |
| `get_multica_work_status` | `issue_id`（必需） | 读取 Multica issue，用于检查委派任务状态。 |

`delegate_to_multica` 返回：

```json
{
  "linear_issue": "ENG-123",
  "linear_url": "https://linear.app/acme/issue/ENG-123",
  "prompt_file": ".workctl/agent-prompts/default.md",
  "multica": {"identifier": "MUL-45"}
}
```

`status` 缺省为 `todo`。Multica issue 描述会保留 Linear 来源（identifier 与 URL）；若指定了 `prompt_file`，还会记录使用的 prompt 文件路径。

## 约定与限制

- 写入类 tool（`sync_agent_prompt`、`delegate_to_multica`）要求显式给出 `agent_id`，不把 token 放进 tool 参数；凭据由本机 CLI/环境变量管理。
- `prompt_file` 读取的是运行 `workctl mcp serve` 的主机上的本地文件；每次委派都会读取最新内容，便于在本地版本化 prompt。
- 只有 `tools/list` 暴露稳定 schema；Multica 一侧的 managed MCP 工具 schema 变化后需要重新批准。
- 该桥接依赖 `multica` CLI 的当前 workspace 上下文（`--workspace-id` > `MULTICA_WORKSPACE_ID` > profile 默认值），`workctl` 的 `--workspace` 是 Linear workspace，不在此处生效。

相关背景见 [workctl → Multica 委派集成调研](research/workctl-multica-integration.md) 与 [Multica API 与 MCP 可行性](https://multica.ai/)。
