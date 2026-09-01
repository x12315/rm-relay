# RM Relay

面向 RoboMaster 队伍的全开源开发基础设施，当前从 STM32 嵌入式基线起步，逐步扩展到
Linux 应用开发和 RM 常见开发板、PC、边缘计算板卡。

> [!IMPORTANT]
> 项目仍在建设。目前已跑通 STM32 项目初始化、本地 BuildKit 构建、Build Output 校验与 OpenOCD
> 命令解析。不可变环境的身份核验、发布契约、远程 BuildKit backend 与 mTLS Compose 部署已进入代码基线，
> 但尚未取得真实 Registry push 和战队服务器证据；快速体验服务器、IDE 一键配置、Linux 应用环境、target 数据回收和
> target 受控运行尚未交付。面向普通用户的原子化 Quick Start 也尚未形成；路线图和架构
> 文档表达建设方向，不代表已经支持。

RM 队伍的成员和工程经验随赛季快速流动。一套环境如果只能由少数人安装、升级和排错，
很容易在交接后失效。RM Relay 将重复出现的工具链配置、项目入口和验证方法整理成可复现
的公共基础设施。项目的完整边界从开发环境延伸到目标设备上的受控开发环境；用户负责应用
源码和启动方式，平台负责构建、传输、调试接入与开发数据回收。

## 项目原则

- **全开源与现代生态：** 以现代软件工程方法组织机器人开发领域成熟的开源工具，摆脱对
  封闭 IDE 和专有开发套件的依赖，为 RM 开发者提供开箱即用、同时留有探索空间的开发套件。
- **广泛的兼容性与支持面：** 逐步覆盖 Windows、macOS 和 Linux，支持 x86、Arm 等架构。
  通过本地容器、远程构建和宿主工具等方式，开发 PC 应用，以及编译、烧录和调试嵌入式
  设备。
- **低使用门槛：** 逐步为 VS Code 等 IDE 提供一键配置，并支持云端编译。熟悉 Docker 的
  用户也可以在本地获得同一套构建环境。
- **最小自研与长期维护：** 项目自身同样优先复用成熟的开源组件，以少量代码完成必要
  整合，降低社区贡献和长期维护成本。同时控制自部署的复杂度，让缺少专职运维人员的战队
  也能在队内完成服务的搭建和维护。

这些原则为何适合 RM 场景，见
[为什么 RM 队伍需要共同维护开发基础设施](docs/community/why-rm-relay.md)。

## 当前能力

当前仓库提供 C++20/STM32 开发镜像、`rm-relay` CLI、跨平台 CMake Project Template，
以及一份在 host 测试和 MCU 固件中复用相同控制逻辑的 PI 示例。CLI 管理本地或远程 Buildx
resource，经指定 Builder 核验并登记不可变的 environment image，再由镜像内 mise 执行固定
CMake Workflow，将 ELF、BIN、MAP 和校验 manifest 导出到开发机。
镜像覆盖 `linux/amd64` 与 `linux/arm64`；GoReleaser 配置可以生成 Darwin、Linux、Windows
的 amd64/arm64 CLI snapshot，真实主机验证目前以 Apple Silicon macOS 为主。

STM32F407 和 RoboMaster C 已能完成交叉编译；RoboMaster C 已在 macOS 通过 ROM DFU
完成写入与回读校验，并通过 ST-Link、OpenOCD 和 GDB 完成固件加载、断点与变量检查。
支持状态以[支持矩阵](docs/user-guide/support-matrix.md)为准，不能从一个平台的结果推导
其他平台已经验证。

RoboMaster C 是首个支持的 board profile，不是项目结构中心。当前嵌入式镜像只提供
工具链，不包含 IDE。未来算力侧会分别提供 development 与匹配的 runtime 环境，但仍不
打包用户应用；编辑器可以消费 CMake Presets 和调试配置，不能成为构建的唯一真相源。

## 使用方式

### 远程快速体验（规划中）

面向没有本地开发环境的初学者。首个实例计划由
[@x12315](https://github.com/x12315) 部署，只向本战队与受邀友队开放，用于验证远程编译
和虚拟 target 链路。它与本地、战队服务器使用同一 CLI，不建设浏览器 IDE，也不承诺
生产级可用性。公开注册和面向陌生用户的强隔离服务属于后续方向。

### 本地 BuildKit（代码链路已完成）

RM Relay 在现有 Docker/Buildx 上管理独立的本地 Builder，源码和构建产物保留在开发机；
USB、烧录与调试按宿主平台接入。CLI 已支持拉取、核验并登记正式 OCI image digest；
官方 Registry 与独立 Project Template 的普通用户分发入口尚未交付，因此当前仍是维护者验证路径。

### 战队远程构建（已配置，待实机验证）

仓库提供 rootless BuildKit 的 mTLS Compose 配置，以及开发机侧 Builder 登记、真实 solve 检查和
远程 workspace backend。Registry 采用托管还是自部署尚未决定；网络接入和证书签发仍由战队负责。参见
[Builder 配置](docs/user-guide/builders.md)与[部署 mTLS BuildKit 服务](docs/operator-guide/deploy-buildkit-service.md)。

想先理解项目将如何工作，阅读[开发平台架构](docs/architecture/README.md)和
[开发契约参考](docs/reference/development-contracts.md)。维护项目或参与建设时，再阅读
[仓库资产地图](docs/architecture/repository-assets.md)、[项目路线](ROADMAP.md)、
[开发者人工核验](tests/manual/README.md)、[社区工作](docs/community/README.md)与
[贡献指南](CONTRIBUTING.md)。当前功能的使用方式从[使用指南](docs/user-guide/README.md)
进入，CLI 制品维护见[发布脚本](scripts/release/README.md)，环境定义维护见
[`embedded-development` 维护指南](environments/embedded-development/MAINTAINING.md)，镜像生产见
[环境镜像构建服务](services/environment-image-builder/README.md)，服务部署与备用维护路径从
[运维指南](docs/operator-guide/README.md)进入。

## 发起与许可证

RM Relay 由首都师范大学 PIE 战队发起，是非官方开源社区项目，与 DJI 或 RoboMaster
官方不存在隶属或背书关系。

项目采用 [Apache License 2.0](LICENSE)。首个开发基线稳定后，我们将在 RoboMaster
社区征集试用队伍和贡献者。
