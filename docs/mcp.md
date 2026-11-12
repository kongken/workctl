# MCP 服务

`workctl mcp serve` 在 stdio 上启动一个 Model Context Protocol 服务，把 Linear 读取、Multica agent/issue 管理与委派桥接暴露为 MCP tools，便于其他 MCP client（例如读取 Linear issue 的 agent）发现并派发 Multica 开发任务。

服务只做同步桥接：读取 Linear issue、按需同步本地 prompt 文件、创建并分配 Multica issue，然后返回映射结果。它不会轮询 Multica run，也不会把状态回写 Linear。

## 启动

```bash
workctl mcp serve
workctl mcp serve --read-only   # 只暴露读取类 tool
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

加 `--read-only` 后只暴露读取类 tool（`get_linear_issue`、`list_multica_agents`、`get_multica_agent`、`get_multica_work_status`、`list_multica_issues`、`list_multica_issue_comments`），适合只想看状态的 client。

## Tools

Linear：

| Tool | 参数 | 说明 |
| --- | --- | --- |
| `get_linear_issue` | `issue_id`（必需） | 按 identifier（如 `ENG-123`）或 UUID 读取 Linear issue 的 JSON 字段。 |

Agent：

| Tool | 参数 | 说明 |
| --- | --- | --- |
| `list_multica_agents` | `include_archived`（可选） | 列出当前 Multica workspace 的 agent。 |
| `get_multica_agent` | `agent_id`（必需） | 读取单个 agent。 |
| `create_multica_agent` | `name`、`runtime_id`（必需）；`description`、`instructions`、`model`、`max_concurrent_tasks`（可选） | 在指定 runtime 上创建 agent。 |
| `update_multica_agent` | `agent_id`（必需）；`name`、`description`、`instructions`、`model`、`status`、`runtime_id`、`max_concurrent_tasks`（可选） | 只更新给出的字段；空字符串表示清空。 |
| `archive_multica_agent` | `agent_id`（必需） | 归档 agent（可恢复）。 |
| `restore_multica_agent` | `agent_id`（必需） | 恢复已归档 agent。 |
| `sync_agent_prompt` | `agent_id`、`prompt_file`（必需） | 读取本地 UTF-8 prompt 文件并更新指定 agent 的 instructions。 |

Issue：

| Tool | 参数 | 说明 |
| --- | --- | --- |
| `get_multica_work_status` | `issue_id`（必需） | 读取 Multica issue，用于检查委派任务状态。 |
| `list_multica_issues` | `assignee_id`、`project`、`status`、`limit`、`offset`（可选） | 按条件列出 issue。 |
| `create_multica_issue` | `title`（必需）；`description`、`assignee_id`、`project_id`、`status`、`priority`、`parent_id`（可选） | 创建 issue，可指定 assignee/项目/父 issue。 |
| `update_multica_issue` | `issue_id`（必需）；其余字段可选 | 只更新给出的字段；空字符串表示清空。 |
| `add_multica_issue_comment` | `issue_id`、`content`（必需）；`parent_id`（可选） | 发表评论，可回复到指定 thread。 |
| `list_multica_issue_comments` | `issue_id`（必需）；`roots_only`、`recent`（可选） | 列出 issue 评论。 |

委派：

| Tool | 参数 | 说明 |
| --- | --- | --- |
| `delegate_to_multica` | `issue`、`agent_id`（必需）；`project_id`、`status`、`prompt_file`（可选） | 读取 Linear issue，可选先同步 prompt，再创建并分配给 Multica agent 的 issue。 |

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

- 写入类 tool 在 tool 元数据里标记为 write，`--read-only` 会把它过滤掉；所有写入都要求显式给出 `agent_id`/`issue_id`，不把 token 放进 tool 参数，凭据由本机 CLI/环境变量管理。
- `prompt_file` 读取的是运行 `workctl mcp serve` 的主机上的本地文件；每次委派都会读取最新内容，便于在本地版本化 prompt。
- `update_multica_agent` / `update_multica_issue` 只发送调用方给出的字段：省略即保持不变，传空字符串则清空该字段。
- 只有 `tools/list` 暴露稳定 schema；Multica 一侧的 managed MCP 工具 schema 变化后需要重新批准。
- 该桥接依赖 `multica` CLI 的当前 workspace 上下文（`--workspace-id` > `MULTICA_WORKSPACE_ID` > profile 默认值），`workctl` 的 `--workspace` 是 Linear workspace，不在此处生效。

相关背景见 [workctl → Multica 委派集成调研](research/workctl-multica-integration.md) 与 [Multica API 与 MCP 可行性](https://multica.ai/)。
