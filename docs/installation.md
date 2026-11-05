# 安装与构建

仓库只提供一个入口 `workctl`，Linear 与 Plane 命令都从同一根命令树开始：

```bash
go build -o workctl ./cmd/workctl
go install github.com/kongken/workctl/cmd/workctl@latest
./workctl --help
./workctl issue --help
./workctl linear issue --help
./workctl plane --help
```

Windows 输出名可用 `workctl.exe`。Plane 预览需从包含本实现的 checkout 构建；尚未发布携带新 binary 的发行版。实际能力及验证状态见 [兼容矩阵](compatibility.md)。

CI 对现有代码执行 test/vet，构建 `cmd/workctl`，并交叉编译 Linux/macOS/Windows amd64；这不等于已验证这些平台的系统 keyring 运行行为。

原 `linear` binary 不再构建。`workctl linear` 提供与 `workctl plane` 对称的显式入口，同时保留直接的 `workctl issue` 等兼容命令。升级方式见 [升级与回退](migration.md)。
