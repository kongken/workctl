# 升级与回退

`linear` binary 已移除，`workctl` 是唯一入口。为与 `plane` 对称，新增 `workctl linear` 显式服务入口；原有直接命令继续兼容：

```bash
# 原脚本与兼容入口
linear --workspace acme issue view ENG-42 --json
workctl --workspace acme issue view ENG-42 --json

# 对称的显式服务入口
workctl --workspace acme linear issue view ENG-42 --json
```

迁移只改命令名：参数、别名、JSON 输出、`LINEAR_*` 环境变量、当前目录 `linear.toml` / `.linear.toml`、原 credentials/keyring 路径都不变。`LINEAR_API_KEY` 与 Linear `--workspace` 的原冲突规则仍保留。Plane 的 profile 只影响 `workctl plane`；没有默认 provider 或 `workctl issue` 自动路由。

Plane 使用 [独立配置](configuration.md)。不要把 Linear team/project UUID 直接复制为 Plane project/module，也不要将 `.linear.toml` 自动改成 Plane 配置。本版本不进行跨服务数据迁移。

Plane v1→v2：新建一个明确使用 v2 的 profile，先验证身份/目录，再在测试项目确认工作项与评论。旧 profile 可保留；不会自动转换原始 JSON 文件、重放失败写请求或降级到 v1。原生 JSON 格式随 API version 不同，跨版本脚本应显式处理。

回退时使用上一版本的 `workctl` binary，旧配置仍在原位置；仍保留的旧 `linear` binary 也可继续读取同一凭据。Plane 用户配置保留 schema_version=1；不认识该 schema 的版本应明确失败。远端已创建/修改的数据不会因 binary 回退而自动回滚；多步骤失败按错误中的资源 ID 显式恢复/清理。

没有发布稳定 Plane 支持。后续发布必须完成目标 Cloud/CE 实测；详见 [兼容与验证](compatibility.md)。
