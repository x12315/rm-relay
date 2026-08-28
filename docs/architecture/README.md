# RM Relay 开发平台架构

> [!IMPORTANT]
> 本目录记录已经确认的设计基线，不代表对应能力已经实现。当前可用范围仍以根
> [README](../../README.md#当前能力) 和[支持矩阵](../user-guide/support-matrix.md)为准。

RM Relay 面向完整的 RM 软件组：嵌入式、电控、视觉、导航和普通 Linux 应用可以使用
不同环境与设备后端，但共享一条从源码到目标设备的开发链路。

用户负责应用源码、算法和程序启动方式。RM Relay 管理这些内容之外的公共基础设施：
开发环境、本地或远程构建、构建结果、目标设备接入、调试连接和开发数据回收。平台不解析
ROS node、launch 文件或普通进程，也不提供通用应用 `run/stop` 状态机。

## 总体拓扑

```text
                         RM Relay 维护者 / 战队运维
                                    │
                       环境配置 ── BuildKit ── Registry
                                      │              │
                              development image  runtime image
                                      │              │
开发者电脑                            │              │
┌─────────────────────────────────────┴──────────────┴───────────────┐
│ 本地源码 → mise tasks → rm-relay CLI                              │
│                         │                                          │
│                         ├─ local build：本地 Docker                │
│                         └─ remote build：workspace builder         │
│                                      │                             │
│                         Build Output 返回本地                       │
│                         ├─ CMake Install Tree                      │
│                         ├─ ROS 2 Install Space                     │
│                         └─ MCU ELF / BIN / MAP                     │
└─────────────────────────┬──────────────────────────────────────────┘
                          │
                          ▼
                 target 接入与调试链路
        ┌─────────────────┼──────────────────────┐
        │                 │                      │
        ▼                 ▼                      ▼
 physical Linux      virtual Linux             MCU
 rm-relay-node       K3s namespace             OpenOCD / DFU / serial
 Docker + Mutagen    isolated runtime          GDB
        │                 │                      │
        └────── shell / debugger / 数据返回 ─────┘
                          │
                          ▼
                       开发者电脑
```

这张图由四个相互独立的平面组成：

1. **环境供应**固定工具、依赖、架构和目标环境血统。
2. **构建**在本地或服务器执行用户项目，并把 Build Output 交回本地。
3. **目标接入**把本地 Build Output 送入物理或虚拟设备，提供 shell、debugger 和数据回收。
4. **服务部署**让战队按需组合 Registry、workspace builder 和虚拟 target 服务。

环境镜像生产与用户项目构建虽然都可以使用 BuildKit，却是两条不同链路；远程构建服务器
也不进入目标设备的实时调试和数据回传路径。

## 最小自研边界

RM Relay 只自行维护两个需要理解项目语义的薄组件：

| 组件 | 职责 | 不负责 |
|---|---|---|
| `rm-relay` | 跨平台客户端入口；编排本地/远程构建和 target adapter | 编译、文件同步算法、应用进程管理 |
| `rm-relay-node` | 在物理 Linux target 上维护受控环境、挂载、版本和基础配置 | 用户源码、应用启动逻辑、战队网络 |

`mise` 负责固定客户端工具版本、环境变量和项目任务，并调用 `rm-relay` 或 CMake、colcon、
OpenOCD 等原生工具。它不承担跨机器状态管理。服务端没有首版自研的 `rm-relay-server`；
BuildKit、OCI Registry、K3s 和 Compose 各自处理已有的基础设施问题。

这个边界允许组件未来独立发布，但当前仍适合留在 monorepo 中。只有某个模块形成独立使用者、
依赖、发布节奏和维护者后，才考虑拆入 RM Relay umbrella project 下的子仓库。

## 跨模块不变量

- 本地源码和项目声明是真相源，远端不长期托管用户 workspace。
- 远程 Build Output 必须先回到本地，构建服务器不直接部署 target。
- 正式环境在镜像构建时固化；运行中的容器不能安装依赖改变环境。
- Linux target 中的容器可以长期存在，但可随时按已知版本重建，不能保存唯一项目资产。
- 用户直接管理应用如何启动和停止；平台只提供受控环境、交互入口和操作顺序。
- MCU 与 Linux target 共享上层能力语义，不共享容器、目录和 daemon 实现。
- 物理 target、虚拟 target 和服务器服务可以独立部署，普通用户仍通过同一客户端入口使用。

比赛用持久部署、开机自启、批量分发和目标系统镜像不是当前开发链路的一部分，见
[路线图](../../ROADMAP.md#后续可选模块)。

## 阅读路径

- [环境与 profile](environments-and-profiles.md)：环境镜像如何分层、组合和扩展。
- [构建与输出](builds-and-outputs.md)：源码如何在本地或远端构建，输出和 cache 如何分开。
- [Target 接入与数据链路](targets-and-access.md)：物理 Linux、虚拟 Linux 和 MCU 如何接入。
- [服务拓扑](service-topology.md)：战队服务器上的镜像、构建和虚拟 target 服务如何组合。
- [开发契约参考](../reference/development-contracts.md)：跨模块共用的术语、身份、路径和不变量。
