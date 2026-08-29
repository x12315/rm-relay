# 仓库资产地图

本页面帮助维护者判断新文件应放在哪里。RM Relay 将可交付的套件内容集中在
[`toolkit/`](../../toolkit/README.md)；
服务整个仓库的示例、验证和文档留在根目录。平台如何工作见
[开发平台架构](../architecture/README.md)，当前支持状态见
[支持矩阵](../user-guide/support-matrix.md)。

## RM Relay 的仓库边界

RM Relay 按能够独立演进的产品边界拆分仓库，而不是按可执行文件、部署位置或生成的制品
拆分。客户端、目标机 daemon、adapter、公共契约和核心 Template 即使生成不同制品，也
归属主仓库；相应功能交付后，必须随同一 RM Relay release 完成组合验证。

```text
rm-relay-integrations ──消费公开 CLI 与契约──→ rm-relay

rm-relay-environments ──发布环境镜像──→ OCI Registry
```

| 仓库 | 组织的资产 | 不组织的资产 | 当前状态 |
|---|---|---|---|
| `rm-relay` | `rm-relay` CLI、`rm-relay-node`、公共契约、Project Template、Dev Container Template，以及端到端验证和部署配置 | 可选 IDE/Agent integration；形成独立发布边界后的环境定义 | 当前主仓库 |
| `rm-relay-environments` | 官方默认与社区环境定义、共享构建能力、profile 与 image 的映射、构建和兼容性验证、发布配置 | OCI image blob、用户源码、核心 CLI 和 Template | 规划中，尚未创建 |
| `rm-relay-integrations` | 可一次性导入的 VS Code/VSCodium 配置，以及通过标准渠道安装的 Agent Skill | 构建、烧录、调试实现；Project Template 和 Dev Container Template | 规划中，尚未创建 |

`rm-relay-environments` 保存能够审查、构建和验证的环境定义；构建后的 image blob 发布到
OCI Registry。普通用户选择 profile，不直接管理这两个底层位置。当前
[`toolkit/container-images/`](../../toolkit/container-images/) 仍留在主仓库，等 profile 契约、
独立验证和发布流程稳定后再迁移。

`rm-relay-integrations` 只消费 `rm-relay` 的公开 CLI、schema、profile 名称和模板契约，核心
链路不能反向依赖它。当前只有[可复制的 VS Code 参考片段](../user-guide/vscode-example.md)，
尚未交付 integration package。两个规划仓库的建设条件见
[路线图](../../ROADMAP.md#2-建立统一入口与-profile-模型)。

## 套件实现与交付资产

| 内容 | 位置 | 当前职责 |
|---|---|---|
| 原生程序入口 | [`toolkit/cmd/`](../../toolkit/cmd/) | `rm-relay` 开发机 CLI 与 `rm-relay-node` 目标机 daemon；当前只有目录占位 |
| 内部 Go package | `toolkit/internal/` | 随真实功能实现增加，不预建 `builds`、`targets` 等概念目录 |
| 开发镜像 | [`toolkit/container-images/embedded-development/`](../../toolkit/container-images/embedded-development/README.md) | 可独立构建、验证和发布的固定工具链 |
| 项目模板 | [`toolkit/project-templates/cross-platform-cpp/`](../../toolkit/project-templates/cross-platform-cpp/README.md) | 供用户复制、改名，未来由 `rm-relay init` 生成的项目起点 |
| OpenOCD 配置 | [`toolkit/openocd/`](../../toolkit/openocd/) | 开发机通过 ST-Link 访问 MCU 时使用的板卡配置 |

Profile、protocol schema 和工具配置应跟随实际消费者保存。只有形成独立构建、测试或发布
边界后，内部 package 才升级为独立 module 或子仓库。

Project Template 与未来每个 profile 对应的 Dev Container Template 都属于套件交付资产。
可选 IDE 配置与 Agent Skill 不写入 Project Template；其边界见
[环境与 profile](../architecture/environments-and-profiles.md#两类核心-template-固定不同入口)。

## 仓库级支撑资产

| 内容 | 位置 | 当前职责 |
|---|---|---|
| 完整示例 | [`examples/deterministic-pi-control/`](../../examples/deterministic-pi-control/README.md) | 用明确行为、测试向量和目标产物展示完整开发路径 |
| 契约验证 | [`validation/`](../../validation/README.md) | 验证仓库拓扑、工具链策略和用户项目构建 |
| 项目文档 | [`docs/`](../) | 保存项目事实、使用入口、架构边界和运维方法 |

模板中的占位代码只证明项目结构可以工作；示例则必须具有完整行为和测试。两者都不是本仓库
向用户承诺兼容性的公共算法库。

## 新内容放在哪里

| 新内容 | 归属 |
|---|---|
| CLI、daemon 及其内部实现 | `toolkit/cmd/` 与 `toolkit/internal/` |
| 构建和发布固定开发工具链 | `toolkit/container-images/` |
| 用户复制或由 CLI 生成的项目起点 | `toolkit/project-templates/` |
| OpenOCD 板卡与调试器配置 | `toolkit/openocd/` |
| 展示完整、可测试行为 | `examples/` |
| 证明仓库契约仍成立 | `validation/` |
| 说明项目事实、架构或操作方式 | `docs/` |
