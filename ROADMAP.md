# RM Relay 路线图

路线图记录公开的建设顺序，不承诺完成时间。具体任务、负责人和外部成果通过 GitHub Issue
追踪；只有取得真实证据的能力才会写入支持矩阵。

当前已建立 STM32 嵌入式开发基线，包括双架构开发镜像、跨平台 CMake 项目模板、PI
控制示例、STM32F407 交叉编译，以及 RoboMaster C 在 macOS 上的 DFU 烧录校验和
OpenOCD/GDB 源码调试闭环。

后续建设遵守[开发平台架构](docs/architecture/README.md)，目标是跑通这条基本链路：

```text
开发机源码
  ↓
rm-relay CLI
  ↓
在开发机本地构建或由编译服务器远程构建
  ↓
Build Output 返回开发机
  ↓
物理或虚拟 target
  ↓
交互式 CLI 与调试
  ↓
开发数据返回开发机
```

## 1. 扩展嵌入式实板调试支持

RoboMaster C 已完成首个 macOS 实板闭环。下一步根据 RM 队伍的真实设备补充 STM32
板卡与调试后端，并完善 Windows、Linux 的宿主接入。board profile 继续与项目结构分离。

## 2. 建立统一入口与 profile 模型

当前 CLI 已能组织本地与远程 BuildKit 构建，由固定 Workflow 执行 CMake，并以 Project、
Profile、Builder、Build Output 和 target adapter 串起 STM32F407 链路。远程链路目前有自动测试
和 mTLS Compose 契约，仍需取得真实战队服务器证据。下一步在这些契约上接入 Linux target。
CMake Presets、CMake、colcon 与原生调试后端仍是各自领域的事实源。

环境镜像通过 Dockerfile 能力层与 Bake 官方 profile 组合；每个正式 profile 提供独立
Dev Container Template，项目扩展以 mise overlay 构建成不可变的派生镜像。当前嵌入式
模板和示例迁移到这套入口后，再用相同方法扩展算力侧环境。

Project Template 继续作为核心资产。当前模板暂存在 monorepo；结构和声明稳定后，将迁入
可独立 clone 的模板仓库。`rm-relay init` 只为已有源码项目建立身份，不生成或下载项目。
Dev Container Template 同样属于核心 profile 契约，不作为 IDE 插件拆分。

### 环境定义仓库

当前环境定义继续保留在主仓库。Profile schema、环境与 image 的映射、独立验证和发布流程
稳定后，再建立 `rm-relay-environments`，分别组织官方默认与社区环境定义；构建后的
image blob 仍发布到 OCI Registry。仓库职责见
[仓库资产地图](docs/operator-guide/repository-assets.md#规划中的仓库边界)。

### 可选 IDE 与 Agent integration

可选 integration 不阻塞基本开发链路，也不在当前阶段创建仓库。满足以下条件后，再设计并
建立独立的 `rm-relay-integrations`：

- `rm-relay init`、Project identity 与机器可读结果已经稳定；
- profile 名称、项目声明 schema、Project Template 和 Build Output 边界已经稳定；
- CLI 提供适合 IDE 和 Agent 消费的机器可读结果；
- 核心仓库能用 contract test 验证外部 integration 没有复制构建、烧录或调试逻辑。

当前[VS Code 示例](docs/user-guide/vscode-example.md)只是可复制的参考片段，不是一键导入或
独立发布的 integration package。规划仓库的资产边界见
[仓库资产地图](docs/operator-guide/repository-assets.md#规划中的仓库边界)；具体目录、Skill
内容、发布命令、CI 和支持列表留到启动时设计。

## 3. 发布镜像并验证远程构建服务

将正式环境镜像发布到适合国内大流量访问的 OCI Registry，减少普通用户重复构建。Registry
采用托管服务还是战队自部署留待后续决议。官方自动生产与战队自行生产复用同一份
Dockerfile、Bake target、验证规则和 OCI 输出契约；Environment builder、Registry、Workspace
builder 保持原子解耦。

现有 BuildKit
workspace backend 已让 Build Output 直接返回开发机；下一步在真实服务器验证 mTLS、远端拉取、
cache 和多人使用边界。开发机 Docker 与远程 backend 继续使用同一项目声明和下游 target 链路。

编译服务器保留 BuildKit cache、依赖 cache 和可信范围内共享的 ccache。开发机上的 cache
独立保存，不与编译服务器同步。

首个远程实例由 [@x12315](https://github.com/x12315) 维护，只服务本战队和受邀友队。
公开注册、匿名代码和强隔离不属于这一阶段。

## 4. 建立算力侧环境与物理 Linux target

按照普通 Linux、原生 C++ 视觉、ROS 2 视觉、导航和厂商 runtime 的真实需求，逐步形成
有限的官方 profile。算力侧 development 与 runtime image 共享 target environment lineage；
用户程序以 CMake Install Tree 或 ROS 2 Install Space 进入目标环境，不制作日常应用镜像。

物理 target 首先支持 Debian/Ubuntu LTS 的 `amd64` 与 `arm64`：

- 以 `.deb` 安装 `rm-relay-node`，维护目标镜像、容器、挂载和环境版本；
- 通过 Mutagen 在开发机与目标宿主机之间同步 Build Output 和开发数据；
- 提供可重连的交互式 shell 和 debugger 直连；
- 验证有线直连、同一局域网 Wi-Fi 与 mDNS 发现。

首条算力侧黄金路径必须在真实 Linux target 上跑通远程构建、结果回传、增量传输、用户运行、
源码调试和数据取回。

## 5. 建立多用户虚拟 target

用 K3s 提供培训和快速体验所需的虚拟 target。每位用户或租户使用独立 namespace、credential、
quota 和 storage；`rm-relay` 提供与物理 target 相近的上层入口，不把 Kubernetes 暴露给
普通用户。

这一阶段只承诺可信战队和受邀队伍的应用 runtime 隔离，不承诺匿名公共 sandbox，也不把
Registry 和 workspace builder 迁入 K3s。

## 6. 提供战队部署说明

说明如何在一台或多台服务器上组合环境镜像构建器、Registry、workspace builder 和 K3s
virtual target，包括版本识别、cache、资源限制、更新与 smoke 验证。RM Relay 提供可复现
配置和检查方法，长期服务由使用它的战队自行维护。

## 7. 开展社区试用与协作

在首条完整开发链路稳定后，于 RM 论坛发布项目介绍并征集试用队伍。贡献重点包括板卡支持、
算力侧 target 兼容组合、跨平台验证、算力侧环境和文档。仓库外工作的记录方式见
[社区工作](docs/community/README.md)。

## 后续可选模块

以下能力不阻塞当前开发链路，出现真实需求和维护力量后再设计：

- 比赛用持久部署、`rm-relay-node` 开机自启扩展与运行模式切换；
- 多机器人批量部署；
- 通过局域网或专用网卡统一分发；
- 目标设备操作系统/ISO 镜像、初始化和高级运维；
- 待取回数据的状态查询、配额、过期回收和自动清理；
- 面向陌生用户的公开注册、多租户、配额和强隔离；
- 硬件 Farm 与集中式开发板管理；
- 原生 ARM64 builder、仅有 ARM 服务器的战队部署和更广泛的跨架构调度；
- 生产级 OTA、A/B 和强事务回滚；
- 大规模仿真、神经网络训练、数据集和实验管理。

这些模块可以复用环境、Build Output、target 和数据契约，但不会反向扩大当前实现的支持
声明。
