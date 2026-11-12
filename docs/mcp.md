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

加 `--read-only` 后只暴露 16 个读取类 tool（Linear 的 `get_linear_issue`；agent 的 `list_multica_agents`、`get_multica_agent`、`get_multica_agent_env`、`list_multica_agent_tasks`；issue 的 `get_multica_work_status`、`list_multica_issues`、`list_multica_issue_comments`、`list_multica_issue_children`、`list_multica_issue_pull_requests`、`list_multica_issue_run_messages`、`list_multica_issue_runs`、`search_multica_issues`、`get_multica_issue_timeline`、`get_multica_issue_usage`、`get_multica_issue_wakeups`），适合只想看状态的 client。

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
| `set_multica_agent_avatar` | `agent_id`、`file`（必需） | 上传 agent 头像图片。 |
| `copy_multica_agent` | `source_agent_id`（必需）；`name`、`runtime_id`、`description`、`instructions`、`model`、`max_concurrent_tasks`、`no_skills`（可选） | 复制 agent（可放到别的 runtime）。 |
| `get_multica_agent_env` | `agent_id`（必需） | 读取 agent 的 custom env（有审计）。 |
| `set_multica_agent_env` | `agent_id`、`env`（必需） | 替换 custom env，传入 JSON 对象字符串；值等于 `****` 时保留原值。 |
| `manage_multica_agent_mcp` | `agent_id`、`action`（`list`/`add`/`remove`/`enable`/`disable`）；`server_id`（除 list 外必需） | 管理 agent 绑定的 workspace MCP server。 |
| `manage_multica_agent_skills` | `agent_id`、`action`（`list`/`add`/`set`）；`skill_ids`（add/set 必需） | 查看或调整 agent 的 skills。 |
| `list_multica_agent_tasks` | `agent_id`（必需）；`limit`、`before`（可选） | 列出 agent 的 run 记录。 |

Issue：

| Tool | 参数 | 说明 |
| --- | --- | --- |
| `get_multica_work_status` | `issue_id`（必需） | 读取 Multica issue，用于检查委派任务状态。 |
| `list_multica_issues` | `assignee_id`、`project`、`status`、`limit`、`offset`（可选） | 按条件列出 issue。 |
| `create_multica_issue` | `title`（必需）；`description`、`assignee_id`、`project_id`、`status`、`priority`、`parent_id`（可选） | 创建 issue，可指定 assignee/项目/父 issue。 |
| `update_multica_issue` | `issue_id`（必需）；其余字段可选 | 只更新给出的字段；空字符串表示清空。 |
| `add_multica_issue_comment` | `issue_id`、`content`（必需）；`parent_id`（可选） | 发表评论，可回复到指定 thread。 |
| `list_multica_issue_comments` | `issue_id`（必需）；`roots_only`、`recent`（可选） | 列出 issue 评论。 |
| `assign_multica_issue` | `issue_id`（必需）；`to`、`to_id`、`unassign`、`start`（可选） | 分配/取消分配 issue 给 member、agent 或 squad。 |
| `set_multica_issue_status` | `issue_id`、`status`（必需）；`duplicate_of`、`start`（可选） | 修改状态；`duplicate_of` 需配合 `cancelled`。 |
| `cancel_multica_issue_task` | `run_id`（必需）；`issue`（可选） | 取消进行中或排队的 run。 |
| `list_multica_issue_children` | `issue_id`（必需） | 按 stage 列出子 issue。 |
| `manage_multica_issue_comment` | `comment_id`、`action`（`update`/`resolve`/`unresolve`/`delete`）（必需）；`content`、`expected_revision`（update 必需） | 更新、resolve、unresolve 或删除评论。 |
| `manage_multica_issue_labels` | `issue_id`、`action`（`list`/`add`/`remove`）（必需）；`label_id`（add/remove 必需） | 查看或增删 issue label。 |
| `manage_multica_issue_metadata` | `issue_id`、`action`（`list`/`get`/`set`/`delete`）（必需）；`key`、`value`、`value_type`（按 action） | 读写 issue 级 KV metadata。 |
| `manage_multica_issue_property` | `issue_id`、`action`（`list`/`set`/`unset`）（必需）；`name`、`value`（按 action） | 读写自定义属性值。 |
| `manage_multica_issue_subscribers` | `issue_id`、`action`（`list`/`add`/`remove`）（必需）；`user`、`user_id`（可选） | 管理订阅者。 |
| `list_multica_issue_pull_requests` | `issue_id`（必需） | 列出 issue 关联的 PR。 |
| `reorder_multica_issue` | `issue_id`（必需）；`before`、`after`、`top`、`bottom`（可选） | 在状态列内调整 issue 顺序。 |
| `rerun_multica_issue` | `issue_id`（必需） | 重新触发当前 assignee 的 run。 |
| `list_multica_issue_run_messages` | `run_id`（必需）；`issue`、`since`（可选） | 列出某次 run 的消息。 |
| `list_multica_issue_runs` | `issue_id`（必需）；`active`、`siblings`（可选） | 列出 issue 的执行历史。 |
| `search_multica_issues` | `query`（必需）；`limit`、`include_closed`（可选） | 按标题/描述/评论搜索 issue。 |
| `get_multica_issue_timeline` | `issue_id`（必需）；`activity_only`、`actions`、`since`（可选） | 读取 issue 时间线。 |
| `get_multica_issue_usage` | `issue_id`（必需） | 汇总 issue 的 token 用量。 |
| `get_multica_issue_wakeups` | `action`（`list`/`get`/`runs`/`events`）（必需）；`issue_id`（events 外必需）、`wakeup_id`（get/runs 必需） | 读取 issue 的 wakeup 规则与事件类型。 |
| `manage_multica_issue_wakeup` | `issue_id`、`action`（`create`/`update`/`disable`/`delete`/`trigger`/`checkin`）（必需）；`wakeup_id`（create 外必需）；其余为 wakeup 条件/调度字段 | 创建、更新、启停或触发 wakeup。 |

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
