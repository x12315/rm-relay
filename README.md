# RM Relay

面向 RoboMaster 队伍的全开源开发基础设施，当前从 STM32 嵌入式基线起步，逐步扩展到
Linux 应用开发和 RM 常见开发板、PC、边缘计算板卡。

> [!IMPORTANT]
> 项目仍在建设。目前已跑通 STM32 项目初始化、本地容器构建、Build Output 校验与 OpenOCD
> 命令解析；快速体验服务器、IDE 一键配置、战队部署方案、Linux 应用环境、远程构建和
> target 数据回收尚未交付。路线图和架构文档表达建设方向，不代表已经支持。

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
以及一份在 host 测试和 MCU 固件中复用相同控制逻辑的 PI 示例。CLI 直接调用 Docker，
再由镜像内 mise 执行固定 CMake Workflow，将 ELF、BIN、MAP 和校验 manifest 导出到开发机。
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

### 本地 Docker（当前可用）

适合已有基本 Docker 使用经验的开发者。工具链在本机容器中运行，源码和构建产物保留在
工作区；USB、烧录与调试按宿主平台接入。

### 战队部署（规划中）

面向战队运维人员，在战队服务器上组合 Registry、远程 workspace 构建与 K3s 虚拟 target，
为成员提供稳定入口。部署文档尚未交付；当前维护者可以先参考
[镜像构建与验证](docs/operator-guide/build-and-verify-images.md)。

## 本地 Docker 快速开始

当前还没有公开 CLI Release 和 OCI image。以下路径用于体验正在开发的基线，需要包含
`subtree` 命令的 Git、Docker/Buildx 和宿主 mise；不会写入开发板 Flash。

```bash
git clone https://github.com/x12315/rm-relay.git
cd rm-relay
mise trust
mise install

docker buildx bake \
  --file container-images/embedded-development/docker-bake.hcl \
  mcu-dev --load
mise run distribution:snapshot
```

这里的 `mcu-dev` 让 Buildx 选择本机架构；维护者验证指定架构时使用
`mcu-dev-arm64`、`mcu-dev-amd64` 或对应的 `verify-*` target。

以下 POSIX shell 步骤已在当前 macOS 基线上执行；Linux 使用相同 archive 结构，但完整宿主
链路尚未验证。命令会解压与本机匹配的 archive，并把 CLI 加入本次终端的 `PATH`：

```bash
platform="$(mise exec -- go env GOOS)_$(mise exec -- go env GOARCH)"
archive="$(find dist -maxdepth 1 -name "rm-relay_*_${platform}.tar.gz" -print -quit)"
test -n "$archive"
mkdir -p dist/local-bin
tar -xzf "$archive" -C dist/local-bin rm-relay
export PATH="$PWD/dist/local-bin:$PATH"
rm-relay --version
```

GoReleaser 同时生成 Windows archive，但 Windows 上的完整构建、Docker 与设备链路尚未形成
验证证据，见[支持矩阵](docs/user-guide/support-matrix.md)。

Project Template 当前仍位于 monorepo。下面先把该子目录拆成一个本地 Git 分支，再 clone
为自己的项目；未来会由可独立 clone 的模板仓库替代这段临时步骤。

```bash
git subtree split \
  --prefix=project-templates/cross-platform-cpp \
  --branch local/cross-platform-cpp-template
git clone \
  --branch local/cross-platform-cpp-template \
  --single-branch \
  . ../rm-relay-starter
cd ../rm-relay-starter
```

在新项目中逐步运行公开入口：

```bash
rm-relay init
rm-relay build
sed -n '1,220p' \
  install/embedded-stm32f407-robomaster-c/rm-relay-output.json
rm-relay flash --target openocd-stlink --dry-run
```

构建结果和 manifest 位于
`install/embedded-stm32f407-robomaster-c/`。`dry-run` 只解析并显示 OpenOCD 命令；它不
证明开发板已经连接或写入。镜像选择、native/STM32 构建、实板接入、IDE 示例和故障排查
统一从[使用指南](docs/user-guide/README.md)进入。维护者的自动测试与镜像验证入口见
[镜像构建与验证](docs/operator-guide/build-and-verify-images.md)。

想先理解项目将如何工作，阅读[开发平台架构](docs/architecture/README.md)和
[开发契约参考](docs/reference/development-contracts.md)。维护项目或参与建设时，再阅读
[仓库资产地图](docs/operator-guide/repository-assets.md)、[项目路线](ROADMAP.md)、
[社区工作](docs/community/README.md)与[贡献指南](CONTRIBUTING.md)。

## 发起与许可证

RM Relay 由首都师范大学 PIE 战队发起，是非官方开源社区项目，与 DJI 或 RoboMaster
官方不存在隶属或背书关系。

项目采用 [Apache License 2.0](LICENSE)。首个开发基线稳定后，我们将在 RoboMaster
社区征集试用队伍和贡献者。
