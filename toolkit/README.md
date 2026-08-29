# RM Relay Toolkit

这里保存 RM Relay 自身的实现与交付资产。仓库级示例、验证和文档分别位于根目录的
`examples/`、`validation/` 和 `docs/`。

- `cmd/rm-relay/`：开发机 CLI 入口，负责 Project/Profile 解析、构建与 target 操作。
- `cmd/rm-relay-node/`：Linux 目标机 daemon 入口，当前仅预留目录。
- `internal/`：Project、Profile、Execution Plan、Build Output、backend 和 target adapter。
- `container-images/`：本地与远程构建使用的固定开发环境。
- `mise/`：CLI 传入 builder 或宿主工具的受控 mise 基线。
- `profiles/`：RM Relay 维护的环境、输出角色和 target capability 组合。
- `project-templates/`：用户复制或未来由 CLI 生成的项目起点。
- `openocd/`：开发机访问 MCU 时使用的 OpenOCD 配置。

不要按架构讨论中的领域名词预建 package。新目录只有在形成明确实现、依赖和测试边界后才加入。
