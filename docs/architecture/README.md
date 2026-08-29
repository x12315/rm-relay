# RM Relay 开发平台架构

> [!IMPORTANT]
> 本目录记录已经确认的设计基线，不表示对应能力已经实现。当前可用范围以根
> [README](../../README.md#当前能力) 和[支持矩阵](../user-guide/support-matrix.md)为准。

本组文档面向 RM Relay 的实现者和部署者，说明开发环境、构建服务与目标设备为何采用
当前边界。跨组件共用的名词、路径和不变量集中在
[开发契约参考](../reference/development-contracts.md)。

## 平台边界

RM Relay 为嵌入式、电控、视觉、导航和普通 Linux 应用提供同一条开发链路。各类项目可以
选择不同环境与设备后端，但都遵循下面的责任划分：

| 用户项目负责 | RM Relay 负责 |
|---|---|
| 应用源码、算法与项目声明 | 固定且可复现的开发环境 |
| 普通程序、脚本、`ros2 run` 或 `ros2 launch` 的启动方式 | 本地或远程构建编排 |
| 应用生成文件的完整性 | Build Output、target 接入、调试连接与开发数据回收 |

平台不解析应用进程结构，也不提供通用的应用 `run/stop` 状态机。比赛用持久部署、开机
自启、批量分发和目标系统镜像是独立问题，见
[路线图](../../ROADMAP.md#后续可选模块)。

## 总体链路

```text
维护者或战队运维
    │
    ├── 环境配置 ── BuildKit ── Registry
    │                              ├── development image
    │                              └── runtime image
    │
开发者电脑
    │
    ├── 本地源码 ── mise tasks / rm-relay CLI
    │                           ├── local build：本地 Docker
    │                           └── remote build：workspace builder
    │                                           │
    │                           Build Output ◀───┘
    │                           ├── CMake Install Tree
    │                           ├── ROS 2 Install Space
    │                           └── MCU ELF / BIN / MAP
    │
    └── target adapter
        ├── physical Linux：rm-relay-node + Docker + Mutagen
        ├── virtual Linux：K3s namespace
        └── MCU：OpenOCD / DFU / serial + GDB
                    │
                    └── shell、debugger 与开发数据返回开发者电脑
```

这条链路分为四个可以独立部署和替换的平面：

1. 环境供应固定工具、依赖、架构和 target environment lineage。
2. 构建在本地或 workspace builder 中执行用户项目，并把 Build Output 交回本地。
3. Target 接入把本地 Build Output 送入物理或虚拟设备，提供交互、调试和数据回收。
4. 服务部署按需组合 Registry、workspace builder 和虚拟 target 服务。

环境镜像生产与用户 workspace 构建都可以使用 BuildKit，但前者生产开发环境，后者消费
开发环境。workspace builder 不参与 target 的实时调试和数据回传。

## 关键设计取舍

### 本地是控制端

源码、项目声明、Build Output 和已经取回的数据保留在开发者电脑。远程服务处理源码快照和
可删除的 cache；Build Output 必须先返回本地，再由客户端选择 target。这样，local build、
remote build 和 target adapter 可以分别演进。

### 环境与应用分开交付

正式环境在镜像构建时固化。项目增加环境依赖时生成派生镜像，运行中的容器不能安装依赖
改变环境。日常开发只传输 Build Output，不反复构建包含用户应用的 OCI image。

### Build Output 是交付边界

Target adapter 消费 CMake Install Tree、ROS 2 Install Space 或 MCU 固件文件，不读取带有
中间文件和绝对路径的 build tree。Cache 只用于加速，不能成为项目资产或构建正确性的
前提。

### Target 共享能力语义，不共享实现

物理 Linux、虚拟 Linux 和 MCU 都可以声明 shell、transfer、flash 或 debug 等能力。MCU
不需要模拟 Linux 容器、目录或 daemon；虚拟 target 也不复制物理宿主机拓扑。

## 自研组件边界

RM Relay 只维护两个需要理解平台语义的薄组件：

| 组件 | 职责 | 边界 |
|---|---|---|
| `rm-relay` | 跨平台客户端入口；编排本地/远程构建与 target adapter | 不实现编译器、文件同步算法或应用进程管理 |
| `rm-relay-node` | 管理物理 Linux target 的受控环境、挂载、版本和基础配置 | 不持有用户源码，不定义应用启动逻辑或战队网络 |

`mise` 固定客户端工具版本、环境变量和项目任务，并调用 `rm-relay`、CMake、colcon 或
OpenOCD 等原生工具。BuildKit、OCI Registry、K3s 和 Compose 继续承担各自已有的基础设施
职责，首版不增加统一的 `rm-relay-server`。

这些组件目前适合留在 monorepo。只有模块形成独立使用者、依赖、发布节奏和维护者后，才
考虑拆入 RM Relay umbrella project 下的子仓库。

## 阅读路径

| 想了解的问题 | 文档 |
|---|---|
| 环境如何分层、组合和扩展 | [环境与 profile](environments-and-profiles.md) |
| 源码如何在本地或远端构建，输出与 cache 如何分离 | [构建与输出](builds-and-outputs.md) |
| 物理 Linux、虚拟 Linux 和 MCU 如何接入 | [Target 接入与数据链路](targets-and-access.md) |
| 战队服务器上的服务如何组合 | [服务拓扑](service-topology.md) |
| 跨模块术语、身份、路径和不变量 | [开发契约参考](../reference/development-contracts.md) |
