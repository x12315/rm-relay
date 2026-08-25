# 仓库资产与产品边界

本页说明仓库中的资产如何分工。项目整体技术拓扑见
[开发平台架构](../architecture/README.md)，稳定术语与生命周期见
[开发契约参考](../reference/development-contracts.md)。

## 资产类型与受众

仓库同时服务两类人员，但不混淆他们维护的资产：

| 资产 | 位置 | 主要用途 | 主要受众 |
|---|---|---|---|
| 镜像产品 | `container-images/embedded-development/` | 构建和发布固定工具链 | 镜像维护者、构建服务部署者 |
| 项目模板 | `templates/cross-platform-cpp/` | 复制、改名并发展为用户项目 | 一般开发者 |
| 完整示例 | `examples/deterministic-pi-control/` | 功能体验、教学和回归验收 | 一般开发者、审查者 |
| 契约验证 | `validation/project-contracts/` | 检查仓库边界与消费者构建 | 两类人员 |

模板中的占位代码只用来证明结构可工作；PI 控制器则是有明确行为和测试向量的示例。
两者都不是仓库向应用承诺版本兼容性的控制算法库。

## 镜像家族

仓库按能力而不是团队名称组织环境：

- `mcu-dev`：裸机/RTOS 目标的交叉编译、烧录与调试。
- 算力侧 profile：未来分别覆盖普通 Linux、原生 C++ 视觉、ROS 2 视觉、导航和厂商
  runtime，不用一个 `compute-dev` 全家桶承载所有能力。

当前只实现 `base` 和 `mcu-dev`。嵌入式目标不需要 Linux runtime；未来算力侧每个正式
profile 则要提供匹配的 development/runtime 环境。runtime 只包含通用运行依赖，不打包
用户应用。

底层能力由 Dockerfile 与 mise 配置分段，最终成品组合只在 Bake 中定义。普通成员选择
官方 profile；项目特有扩展通过 mise overlay 构建成派生镜像，不能在运行中的容器内临时
修改环境。

## 用户项目的源码依赖方向

“共享核心”在这里是用户项目内部的架构角色，不是本仓库名为 `shared_core` 的产品模块。
该角色只包含领域数据、控制算法、状态和确定性规则。STM32 启动、HAL、RTOS、ROS 2、
网络、总线与硬件访问位于平台适配层，不能反向进入它。

RoboMaster C 是首个 board profile，不是项目核心。新增 STM32 或 STC 设备时增加
对应工具链与设备配置；不会用一块板的目录结构约束其他设备。

## 已设计但尚未实现

- mise 统一任务入口与官方 profile/Template 模型；
- 国内 OCI Registry、远程 workspace 构建和邀请制体验实例；
- 算力侧 development/runtime 环境与跨架构 sysroot；
- 物理/虚拟 Linux target、Development Session 与数据取回；
- Windows 的宿主 OpenOCD 与 WSL2/USBIPD 路径。

这些能力属于基本开发链路，不能因为尚未实现而写成项目外能力。比赛用持久部署、开机
自启、批量分发、目标系统镜像和公网多租户则位于
[后续可选模块](../../ROADMAP.md#后续可选模块)。

IDE 只负责编辑和调试体验。项目可以提供 VS Code/VSCodium、Dev Container 的工作区、
任务和调试预设，但这些配置必须调用 mise、CMake、OpenOCD、GDB 和 SSH 等已有入口，
不能另建构建逻辑。任何镜像都不打包 IDE、用户扩展或个人配置。

Apple `container` 可以作为普通 OCI runtime 的实验项，不作为当前 USB/ST-Link 后端。
Agent Skill 或 MCP server 只封装已经稳定、重复的操作；架构仍以标准配置、CLI、文档和
可执行验证为准。
