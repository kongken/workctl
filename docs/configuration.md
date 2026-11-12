# 配置

Linear 与 Plane 共用一个用户配置文件：Unix 为 `$XDG_CONFIG_HOME/workctl.toml`，默认 `~/.config/workctl.toml`；Windows 为 `%APPDATA%/workctl.toml`。配置按服务分段，避免一个服务的字段改变另一个服务的解析结果。

```toml
schema_version = 1

[linear]
team_id = "ENG"
workspace = "acme"
# api_key = "..." # 建议使用环境变量或 credentials/keyring

[plane]
default_profile = "cloud"

[plane.profiles.cloud]
base_url = "https://api.plane.so"
workspace = "acme"
api_version = "v2"
```

Linear 选项的优先级为 `LINEAR_<KEY>` 环境变量 > 当前目录 `linear.toml` / `.linear.toml` > 共享用户文件的 `[linear]` 段。Plane 的 profile 选择为命令行参数 > `PLANE_PROFILE` > 当前目录 `workctl.toml` / `.workctl.toml` > 用户文件的 `plane.default_profile`；Plane 的环境变量和项目字段遵循各字段的命令约定。用户文件不存在时忽略，旧版显式 Linear 配置格式仍可读取。

Plane 严格校验自己的 `[plane]` schema 和 `schema_version`，未知 Plane 字段或坏 TOML 会报错；`[linear]` 中的字段由 Linear 读取，Plane 会忽略。Linear 命令不会读取 Plane profile，Plane 命令也不会初始化 Linear 凭据。共享文件若包含 `api_key`，必须限制为当前用户可读（Unix `0600`）；更安全的选择是环境变量、系统 keyring 或各自的 credentials 存储。错误信息只包含来源路径和解析位置，不包含配置值。

项目配置仅允许 Plane 的 profile、workspace、project_id（以及 Linear 原有项目字段）；可省略 `schema_version`，提供时必须为 1。凭据不写入项目文件。
