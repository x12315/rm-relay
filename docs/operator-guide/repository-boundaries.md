# 开发环境架构边界

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
- `compute-dev`：未来普通 Linux、视觉、ROS 2、NVIDIA 或 AXERA 等算力侧环境的家族名。

当前只实现 `base` 和 `mcu-dev`。未来视觉算法可能拆成多个 compute profile，
不应因此把 ROS 2 当作整个算力侧的唯一边界。

## 用户项目的源码依赖方向

“共享核心”在这里是用户项目内部的架构角色，不是本仓库名为 `shared_core` 的产品模块。
该角色只包含领域数据、控制算法、状态和确定性规则。STM32 启动、HAL、RTOS、ROS 2、
网络、总线与硬件访问位于平台适配层，不能反向进入它。

RoboMaster C 是首个 board profile，不是项目核心。新增 STM32 或 STC 设备时增加
对应工具链与设备配置；不会用一块板的目录结构约束其他设备。

## 暂不交付的能力

- runtime 镜像只在出现具体应用运行依赖后引入；当前拆分价值不足。
- SSH/rsync 远程部署优先级较高，但不属于本分支。
- Windows 的宿主 OpenOCD 与 WSL2/USBIPD 路径需后续在真实机器验证。
- Apple `container` 可作为普通 OCI 运行时的实验项，不作为当前 USB/ST-Link 后端。
- Agent Skill 或 MCP server 只有在人工流程稳定、重复且确有收益后再实现；当前标准 CLI
  和文档是基本途径。

IDE 是用户负责的编辑和调试前端。仓库可以给出接入示例，但不打包 IDE、用户扩展或
个人配置。
