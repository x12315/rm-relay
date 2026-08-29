# 仓库资产地图

本页面向需要新增或审查仓库资产的维护者。它只说明文件应放在哪里，以及各类资产不能相互
替代什么；平台如何工作见[开发平台架构](../architecture/README.md)，当前支持状态见
[支持矩阵](../user-guide/support-matrix.md)，建设顺序见[路线图](../../ROADMAP.md)。

## 四类正式资产

| 资产 | 位置 | 交付内容 | 不承担的职责 |
|---|---|---|---|
| 镜像产品 | [`container-images/embedded-development/`](../../container-images/embedded-development/README.md) | 可独立构建、验证和发布的固定工具链 | 用户应用、IDE、个人配置 |
| 项目模板 | [`templates/cross-platform-cpp/`](../../templates/cross-platform-cpp/README.md) | 供用户复制、改名和继续开发的项目起点 | 完整示例行为、公共算法库 |
| 完整示例 | [`examples/deterministic-pi-control/`](../../examples/deterministic-pi-control/README.md) | 有明确行为、测试向量和目标产物的可执行示例 | 用户项目的固定目录模板、版本化算法 API |
| 契约验证 | [`validation/`](../../validation/README.md) | 从维护者视角验证仓库拓扑、工具链策略和消费者构建 | 用户项目功能、硬件证据的替代品 |

模板中的占位代码只证明结构可工作；PI 控制器展示一条完整行为路径。两者都不是本仓库向
应用承诺兼容性的控制算法库。

## 新内容放在哪里

| 新内容回答的问题 | 归属 |
|---|---|
| 如何构建并发布一套开发工具链？ | `container-images/` 下的独立镜像产品 |
| 用户复制后从哪里开始自己的项目？ | `templates/` 下的项目模板 |
| 某种项目结构如何实现可观察、可测试的完整行为？ | `examples/` 下的完整示例 |
| 如何证明仓库边界、工具链策略或消费者构建仍成立？ | `validation/` 下的契约验证 |

模板和示例内部的源码依赖方向由各自的就近 README 与构建配置说明。RoboMaster C、未来板卡、
算力侧环境和远程服务属于产品能力，不决定上述四类资产的仓库边界。
