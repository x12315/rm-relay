# RM Relay 开发平台架构

本文是 RM Relay 的架构入口，面向第一次接触项目、准备参与实现或部署的工程师。建议先按
顺序读完本文，再根据问题进入专题页。读完后，你应该能沿一条开发闭环说明各组件为何存在、
数据由谁保管，以及当前实现与目标设计之间还有哪些距离。

> [!IMPORTANT]
> 本组文档记录已经确认的设计基线，不等于全部能力已经交付。当前仓库交付 STM32
> 嵌入式开发基线，以及环境身份核验、环境发布契约、本地与远程两种 BuildKit Builder；
> 真实 Registry push 和战队服务器证据尚未取得。Linux 环境、物理 Linux target 和虚拟 target 仍在建设。实际能力和
> 证据等级只以[支持矩阵](../user-guide/support-matrix.md)为准。

## 先建立一个完整模型

RM Relay 解决的是开发链路问题，不是机器人应用问题。用户项目决定写什么程序以及如何
启动它；RM Relay 让同一份项目声明能在固定环境中构建，并把可部署结果送到不同 target，
同时保留调试和数据返回路径。

一轮开发从开发机开始，也在开发机收口：

```text
开发机上的源码与项目声明
        │
        ▼
选择 development profile
        │
        ▼
local BuildKit 或 remote workspace builder
        │
        ▼
Build Output 返回开发机
        │
        ▼
target adapter ──→ MCU / 物理 Linux / 虚拟 Linux
        │                         │
        └──── shell、debugger、Managed Data ────→ 开发机
```

这条链路固定使用三个部署角色：

| 名称 | 准确定义 | 主要状态与入口 |
|---|---|---|
| **开发机** | 保存项目并发起开发操作的机器 | 源码、项目声明、Build Output、已取回数据，以及 `rm-relay`、mise、Mutagen、OpenOCD、GDB 等客户端入口 |
| **编译服务器** | 接收源码快照并执行远程构建的机器 | Workspace builder、BuildKit、ccache 和依赖 cache；结果必须返回开发机 |
| **目标机** | 最终执行用户程序或固件的设备 | MCU、物理 Linux target 或虚拟 Linux target，以及它们提供的运行、交互和调试能力 |

这些名称按职责划分，不表示必须准备三台物理机器。本地构建时，开发机同时承担编译；培训
服务也可以把编译服务器与虚拟目标机部署在同一台服务器上。物理位置可以合并，源码归属、
构建状态和 target 状态的责任边界不能合并。

这里有四个第一次阅读就要知道的术语：

- **Profile**：一组经过验证的环境、架构和 target 兼容要求。
- **Build Output**：可以交给烧录、传输或 debugger 的结果，例如 MCU 的 ELF/BIN/MAP、
  CMake Install Tree 或 ROS 2 Install Space。
- **Target**：接收 Build Output 并提供 flash、shell、transfer 或 debug 等开发能力的物理或
  虚拟设备。
- **Managed Data**：应用写入受管目录、需要从 target 取回开发机的日志、trace、bag 等开发
  数据。

这条链路最重要的约束是：源码、项目声明、Build Output 和已取回的数据以开发机为
真相源。远程服务和 target 可以保留 cache 或暂存数据，但不能成为唯一资产位置。

## 沿开发闭环理解四个责任面

### 1. 环境先固定“用什么构建”

Development profile 固定工具链、依赖、目标架构和兼容基线。项目需要额外依赖时，要从
官方环境构建派生镜像；不能在运行中的正式容器里临时安装，从而让开发机与编译服务器得到
不同环境。

当前仓库已经把 `base` 和 `mcu-dev` 作为可选择的环境 stage，并分别为 `linux/amd64`、
`linux/arm64` 定义了 Bake target。首个嵌入式 Profile 与 mise 能力层已经接入；算力侧
development/runtime 环境和项目 overlay 仍是后续实现，详见
[环境与 profile](environments-and-profiles.md)。

### 2. 构建只回答“如何得到可交付结果”

Local 与 remote Builder 消费相同的项目声明与 development profile。构建系统仍是
CMake、colcon、Ninja、CTest 等原生工具；`mise` 组织常用任务，`rm-relay` 只编排
跨容器、跨机器和 target 相关操作。

两种 backend 的共同出口都是开发机上的 Build Output。Remote workspace 是一次性工作区，
服务端 cache 可以删除；workspace builder 不直接把结果部署到 target。当前 MCU 模板已由
CMake install 将 ELF/BIN/MAP 导出到 `install/<profile>`，并生成内容校验 manifest。
统一 backend 已通过 Buildx local exporter 返回同一目录，并在发布新输出前使用受管临时目录；
真实服务器证据仍待补充。详见[构建与输出](builds-and-outputs.md)。

### 3. Target 接入回答“结果去哪里、如何调试”

Build Output 回到开发机后，target adapter 才接手。三类 target 共享能力名称，不共享内部
实现：

```text
rm-relay
└── target adapter（客户端能力接口）
    ├── MCU：直接调用宿主 flash/debug 工具
    ├── 物理 Linux：调用 rm-relay-node 承担的 provider 角色
    └── 虚拟 Linux：调用基于 K3s 的 virtual target provider
```

Adapter 选择实现并把开发机上的 Build Output、shell 或 debug 请求转换为对应操作；provider
管理 target 或 Target Environment 的生命周期与内部状态。MCU 没有独立 provider daemon。

| Target | 适合的实现 | 开发能力 |
|---|---|---|
| MCU | OpenOCD、DFU、serial、GDB 等宿主工具 | flash、reset、serial、debug |
| 物理 Linux | `rm-relay-node`、Docker、Mutagen | shell、transfer、debug、数据回收 |
| 虚拟 Linux | K3s namespace 与 virtual target provider | shell、transfer、应用 runtime、数据回收 |

MCU 不模拟 Linux 容器或 daemon；虚拟 target 也不复制物理宿主机的 kernel、驱动和设备。
Debugger 默认由开发机直连 target，不经过 workspace builder。各 target 的生命周期和
数据路径见[Target 接入与数据链路](targets-and-access.md)。

### 4. 服务部署回答“哪些角色放在哪些机器上”

环境镜像构建器、OCI Registry、workspace builder 和 K3s virtual target 是不同服务角色。
角色由输入、权限、持久状态和生命周期划分，不由物理机器划分：资源有限时可以部署在同一
台服务器，需要扩容时也可以分别迁移。

RM Relay 首版复用 BuildKit、Registry、Compose 和 K3s 已有控制面，不增加统一的
`rm-relay-server`。部署角色、访问者和状态边界见[服务拓扑](service-topology.md)。

## 谁拥有哪一部分

读完主线后，可以用这张表检查责任有没有串位：

| 对象 | 真相源或管理者 | 不应承担的责任 |
|---|---|---|
| 应用源码、算法、项目声明 | 开发机上的用户项目与 Git | 由远程 builder 或 target 长期托管 |
| Development/runtime 环境 | RM Relay profile 与镜像配置 | 在运行中的正式容器里临时改变 |
| Build Job | local/remote backend | 直接部署 target 或保存唯一源码 |
| Build Output | 开发机上的项目工作区 | 混入 build tree、cache 或服务端内部路径 |
| Target Environment | `rm-relay-node` 或 virtual target provider | 定义用户应用的启动与进程模型 |
| Managed Data | 取回后由开发机上的工作区保管 | 永久依赖 target 保存唯一副本 |

跨组件共用的名称、身份、目录和生命周期属于查阅信息，集中在
[开发契约参考](../reference/development-contracts.md)，不在每篇专题里重复定义。

## 贯穿所有组件的不变量

后续组件设计可以更换工具或协议，但不能破坏以下边界：

1. **开发机是真相源。** 远程构建只处理源码快照；Build Output 先回开发机，再进入 target。
2. **环境可复现。** 正式依赖在镜像构建时固化；Linux development/runtime 环境共享兼容
   lineage。
3. **Build Output 是交接面。** Target adapter 不读取带中间文件和绝对路径的 build tree；
   cache 只加速，不证明身份、权限或构建正确性。
4. **Target 可恢复。** Target Environment 可以由固定镜像重建；除尚未取回的数据外，target
   不保存唯一项目资产。
5. **调试不绕道 builder。** Workspace builder 只生成结果，shell、debugger 与数据连接面向
   target。
6. **平台不接管应用启动。** 普通程序、脚本、`ros2 run` 和 `ros2 launch` 由用户执行，
   RM Relay 不提供通用 `run/stop` 状态机。

## 主线运行组件为什么只有两个

RM Relay 只自研必须理解上述边界的薄层：

| 组件 | 负责 | 不负责 |
|---|---|---|
| `rm-relay` | 跨平台入口；选择 backend、profile 和 target adapter | 编译器、文件同步算法、应用进程管理 |
| `rm-relay-node` | 物理 Linux target 的环境、挂载、版本与基础配置 | 用户源码、应用启动逻辑、战队网络 |

`mise`、BuildKit、OCI Registry、K3s、Mutagen、CMake、colcon、OpenOCD 和 GDB 继续承担各自
已有的职责。只有现有工具无法表达平台契约时，薄组件才增加逻辑。

环境镜像构建服务是围绕 Docker Bake 的维护入口，不是常驻 daemon 或另一套普通用户 CLI；
Candidate 则只服务候选版本验收。它们不会扩大上述运行组件。`rm-relay` 与 `rm-relay-node` 当前
留在 monorepo；只有某个模块形成独立使用者、依赖、发布节奏和维护者后，才考虑拆入 RM Relay
umbrella project 下的子仓库。

## 不属于这条开发链路的能力

比赛用持久部署、开机自启、批量分发、目标系统镜像、生产级 OTA 和面向陌生用户的公共
sandbox 不阻塞当前开发闭环，见[路线图的后续可选模块](../../ROADMAP.md#后续可选模块)。
浏览器 IDE、通用任务队列、网络 overlay 和自研文件同步协议也不属于首版服务边界。

## 接下来按问题阅读

首次阅读到这里已经建立了完整模型。需要实现或审查某一段时，再进入对应文档：

| 你的问题 | 阅读入口 |
|---|---|
| Profile 如何组合，development/runtime 如何保持兼容 | [环境与 profile](environments-and-profiles.md) |
| Local/remote build 如何共享入口，哪些文件属于 Build Output | [构建与输出](builds-and-outputs.md) |
| 三类 target 如何接入，数据和调试如何返回开发机 | [Target 接入与数据链路](targets-and-access.md) |
| Registry、builder 和 K3s 应如何部署 | [服务拓扑](service-topology.md) |
| 某个术语、身份、目录或生命周期的准确契约 | [开发契约参考](../reference/development-contracts.md) |
| 现在究竟支持哪些平台和后端 | [支持矩阵](../user-guide/support-matrix.md) |
| 下一步按什么顺序建设 | [路线图](../../ROADMAP.md) |
| 源码、镜像、模板、示例、测试和静态检查如何分工 | [仓库资产地图](repository-assets.md) |
