# 仓库资产地图

本页用于判断新文件应由哪个模块负责。RM Relay 不设置全局 `config/`、`assets/`
或 `profiles/` 杂项层；TOML、OpenOCD cfg 和其他声明式文件也是源码，跟随解释它们的
模块。

## 主仓库拓扑

```text
cmd/                    可执行程序的 composition root
internal/
├── cli/               公开命令树与 human/JSON 结果
├── project/           用户 `rm-relay.toml` 契约
├── profile/           受支持能力组合与 builtin Profile
├── build/             Plan、Workflow、backend 和 Build Output
├── target/            从已验证 Build Output 到 target 的 adapter
└── execution/         OS process、mise 和内嵌资源物化

container-images/       可独立构建和发布的开发环境
project-templates/      用户复制后由 CLI 建立项目身份的起点
examples/               有完整行为与测试的示例
validation/             跨模块契约、平台编译与 acceptance
docs/                   面向人的项目事实与入口
```

`internal/build/cmake/build.mise.toml` 归 CMake Workflow 所有；
`internal/target/openocd/board/` 归 OpenOCD adapter 所有。`rm-relay.toml` 中的 `system`
选择构建 Workflow，`preset` 是该 Workflow 的输入；Profile 通过 `adapter` 和
`board` 等语义 ID 选择 target 能力。两者都不保存跨模块文件路径。

Project 把输出角色映射到项目内的相对路径；Profile 声明 development image、必需输出角色，
并把 target 的 `artifact_role` 连接到 adapter。Build Plan 负责确认两边的角色契约相容。

用户项目中唯一的 RM Relay 配置入口是 `rm-relay.toml`。mise 是 CLI 与受控环境之间的
执行层，不要求用户项目提供 `mise.toml`。

## 文件归属

| 新内容 | 位置 |
|---|---|
| CLI 命令与输出契约 | `internal/cli/` |
| Project schema 与初始化 | `internal/project/` |
| 正式 Profile 组合 | `internal/profile/builtin/<profile>/` |
| 构建系统解释与受控 task | `internal/build/<system>/` |
| local/remote build 执行实现 | `internal/build/backend/<backend>/` |
| target adapter 及其板卡/协议资产 | `internal/target/<adapter>/` |
| 通用进程与工具调用边界 | `internal/execution/` |
| 开发镜像产品 | `container-images/<image>/` |
| 用户工程起点 | `project-templates/<template>/` |
| 完整可测行为 | `examples/<example>/` |
| 模块外的组合证据 | `validation/` |

模块逻辑的测试与 Go package 就近保存；只有仓库拓扑、真实 development image 和
跨平台交付等跨模块证据放入 [`validation/`](../../validation/README.md)。

## 规划中的仓库边界

RM Relay 按独立使用者、发布节奏和维护者拆分仓库，不按生成了几个可执行文件拆分。

| 仓库 | 负责范围 | 状态 |
|---|---|---|
| `rm-relay` | CLI、未来的 target daemon、公共契约、Project Template 和组合验证 | 当前主仓库 |
| `rm-relay-environments` | 官方与社区环境定义、image/profile 映射、兼容性验证与发布 | 规划中 |
| `rm-relay-integrations` | 可选 VS Code/VSCodium 预设和 Agent Skill | 规划中 |

`container-images/` 在环境定义形成独立发布边界前继续留在主仓库。Integration 只消费
公开 CLI、schema 和 Profile ID，核心链路不反向依赖它。
