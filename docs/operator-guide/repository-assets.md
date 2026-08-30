# 仓库资产地图

本页用于判断新文件应由哪个模块负责。RM Relay 不设置全局 `config/`、`assets/`
或 `profiles/` 杂项层；TOML、OpenOCD cfg 和其他声明式文件也是源码，跟随解释它们的
模块。

## 主仓库拓扑

```text
cmd/
├── rm-relay/           普通用户 CLI 的 composition root
├── rm-relay-maintainer/维护者候选制备与本地分发入口
└── rm-relay-node/      目标机组件的预留边界
internal/
├── cli/               公开命令树与 human/JSON 结果
├── project/           用户 `rm-relay.toml` 契约
├── profile/           受支持能力组合与 builtin Profile
├── build/             Plan、Workflow、backend 和 Build Output
├── target/            从已验证 Build Output 到 target 的 adapter
├── execution/         OS process、mise 和内嵌资源物化
└── maintainer/        仓库外候选环境与 CLI 本地分发

container-images/       可独立构建和发布的开发环境
project-templates/      用户项目起点；当前由 monorepo 提供
examples/               有完整行为与测试的示例
tests/
├── architecture/      可执行依赖方向
├── integration/       跨模块组合
├── distribution/      CLI archive 契约
├── e2e/               分发二进制驱动的自动真实链路
└── manual/            只需人判断的候选版本用户体验核验
scripts/verify/         仓库拓扑、版本和软件源等静态契约
docs/                   面向人的项目事实与入口
```

`internal/build/cmake/build.mise.toml` 归 CMake Workflow 所有；
`internal/target/openocd/board/` 归 OpenOCD adapter 所有。`rm-relay.toml` 中的 `system`
选择构建 Workflow，`preset` 是该 Workflow 的输入；Profile 通过 `adapter` 和
`board` 等语义 ID 选择 target 能力。两者都不保存跨模块文件路径。

Project 把输出角色映射到项目内的相对路径；Profile 声明 development image、必需输出角色，
并把 target 的 `artifact_role` 连接到 adapter。Build Plan 负责确认两边的角色契约相容。

用户项目中唯一的 RM Relay 配置入口是 `rm-relay.toml`。mise 分别承担容器内 Workflow 与
宿主工具 adapter 的受控执行，不要求用户项目提供 `mise.toml`。

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
| 候选环境与 CLI 本地分发实现 | `internal/maintainer/` |
| 开发镜像产品 | `container-images/<image>/` |
| 用户工程起点 | `project-templates/<template>/` |
| 完整可测行为 | `examples/<example>/` |
| 可执行依赖方向 | `tests/architecture/` |
| 跨模块组合行为 | `tests/integration/` |
| CLI archive 契约 | `tests/distribution/` |
| 分发二进制驱动的真实开发链路 | `tests/e2e/` |
| 候选版本的用户体验人工核验 | `tests/manual/user-experience/` |
| 静态仓库契约 | `scripts/verify/` |

模块逻辑的测试与 Go package 就近保存。`tests/` 只收模块外的可执行行为；静态文件与策略
检查放入 `scripts/verify/`；人工核验只保留自动测试不能代替的用户认知判断，不与自动 E2E 共用
实现或测试用品；镜像构建时的工具能力检查仍归各镜像的 `smoke/` 所有。
GoReleaser 定义 CLI 平台矩阵与 archive，Docker Bake 定义 image 构建矩阵，两者不在测试
代码中维护第二份目标列表。

## 规划中的仓库边界

RM Relay 按独立使用者、发布节奏和维护者拆分仓库，不按生成了几个可执行文件拆分。

| 仓库 | 负责范围 | 状态 |
|---|---|---|
| `rm-relay` | CLI、未来的 target daemon、公共契约和主线测试 | 当前主仓库 |
| `rm-relay-template-*` | 每个可独立 clone 的 Project Template | 规划中；当前模板仍在主仓库 |
| `rm-relay-environments` | 官方与社区环境定义、image/profile 映射、兼容性验证与发布 | 规划中 |
| `rm-relay-integrations` | 可选 VS Code/VSCodium 预设和 Agent Skill | 规划中 |

`container-images/` 在环境定义形成独立发布边界前继续留在主仓库。Integration 只消费
公开 CLI、schema 和 Profile ID，核心链路不反向依赖它。Project Template 迁出后仍是核心
入口；一仓库一模板保证用户可以直接 clone，不再复制 monorepo 子目录。
