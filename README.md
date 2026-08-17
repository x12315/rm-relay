# RM Relay

面向 RoboMaster 队伍的全开源、新手友好开发套件，覆盖嵌入式与 Linux 应用开发，逐步
适配 RM 常见开发板、PC 和边缘计算板卡。

> [!IMPORTANT]
> 项目仍在建设。目前可用的是 STM32 嵌入式开发基线；快速体验服务器、战队部署方案和
> Linux 应用开发环境尚未交付。路线图表达建设方向，不代表已经支持。

RM 队伍的成员和工程经验随赛季快速流动。一套环境如果只能由少数人安装、升级和排错，
很容易在交接后失效。RM Relay 将重复出现的工具链配置、项目入口和验证方法整理成可复现
的公共基础设施，让新人能开始使用，也让队伍有能力继续维护。

## 项目原则

- **全开源：** 使用开放的命令行工具链，优先组合成熟的上游项目，不用专有 IDE 或工程
  格式控制构建过程。
- **跨平台、跨架构：** 面向 Windows、Linux、macOS，以及 x86_64、ARM64、MCU 和边缘
  计算目标。Docker 统一适合容器化的工具环境；USB 设备访问、烧录、调试、串口和 IDE
  接入按宿主平台补齐。项目同时保留本地 Docker 与远程 Docker 服务两种使用拓扑。
- **降低使用门槛：** 将稳定而重复的流程封装为少量入口。用户可以选择未来的公共快速
  体验服务、当前可用的本地 Docker，或由战队长期维护的内部服务。
- **控制维护成本：** 尽量复用上游并减少自研代码。工具版本按 LTS 基线固定，大体积镜像
  和构建流量优先放在国内，少量源码仍连接原始上游。

这些原则为何适合 RM 场景，见
[为什么 RM 队伍需要共同维护开发基础设施](docs/community/why-rm-relay.md)。

## 当前能力

当前仓库提供 C++20/STM32 开发镜像、可复制的跨平台 CMake 项目模板，以及一份在 host
测试和 MCU 固件中复用相同控制逻辑的 PI 示例。镜像覆盖 `linux/amd64` 与
`linux/arm64`，真实主机验证目前以 Apple Silicon macOS 为主。

STM32F407 和 RoboMaster C 已能完成交叉编译；RoboMaster C 的 ROM DFU 已在 macOS
完成只读枚举，OpenOCD/GDB 配置尚待使用 SWD 实板完成烧录和源码调试闭环。支持状态以
[支持矩阵](docs/user-guide/support-matrix.md)为准，不能从工具存在推导硬件已经验证。

RoboMaster C 是首个支持的 board profile，不是项目结构中心。开发镜像只提供工具链，
不包含机器人应用 runtime 或 IDE；编辑器可以消费 CMake Presets 和调试配置，但不是
构建的唯一真相源。

## 使用方式

### 快速体验（规划中）

面向没有本地开发环境的初学者。公共服务器将允许用户通过网络体验编译和基础验证，计划
由 [@x12315](https://github.com/x12315) 部署与维护，不承诺生产级可用性。完成试用后，
推荐联系战队维护者部署长期使用的内部服务。

### 本地 Docker（当前可用）

适合已有基本 Docker 使用经验的开发者。工具链在本机容器中运行，源码和构建产物保留在
工作区；USB、烧录与调试按宿主平台接入。

### 战队部署（规划中）

面向战队运维人员，在战队服务器上托管镜像与远程编译能力，为成员提供稳定入口。部署
文档尚未交付；当前维护者可以先参考
[镜像构建与验证](docs/operator-guide/build-and-verify-images.md)。

## 本地 Docker 快速开始

以下命令需要 Docker 与 Buildx，从仓库根目录执行。它们会构建本机架构镜像，在本地
`build/` 目录运行测试并生成固件，不会写入开发板 Flash。

```bash
docker version
docker buildx version
docker buildx bake \
  --file container-images/embedded-development/docker-bake.hcl \
  mcu-dev --load
sh validation/project-contracts/verify-repository-layout.sh
sh validation/project-contracts/verify-project-builds.sh
```

验证包含模板和 PI 示例的 native Clang、native GCC、ASan/UBSan 测试，以及
STM32F407/RoboMaster C 交叉编译。进一步操作见：

- [镜像选择与运行](docs/user-guide/image-selection.md)
- [native 构建与测试](docs/user-guide/build-native.md)
- [STM32 固件构建](docs/user-guide/build-stm32.md)
- [烧录、调试与平台支持](docs/user-guide/support-matrix.md)

维护项目或参与建设时，阅读[仓库资产边界](docs/operator-guide/repository-boundaries.md)、
[项目路线](ROADMAP.md)、[社区工作](docs/community/README.md)与[贡献指南](CONTRIBUTING.md)。

## 发起与许可证

RM Relay 由首都师范大学 PIE 战队发起，是非官方开源社区项目，与 DJI 或 RoboMaster
官方不存在隶属或背书关系。

项目采用 [Apache License 2.0](LICENSE)。首个开发基线稳定后，我们将在 RoboMaster
社区征集试用队伍和贡献者。
