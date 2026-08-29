# 仓库资产地图

本页面帮助维护者判断新文件应放在哪里。RM Relay 将可交付的套件内容集中在
[`toolkit/`](../../toolkit/README.md)；
服务整个仓库的示例、验证和文档留在根目录。平台如何工作见
[开发平台架构](../architecture/README.md)，当前支持状态见
[支持矩阵](../user-guide/support-matrix.md)。

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
