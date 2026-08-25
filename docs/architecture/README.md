# RM Relay 开发平台架构

> [!IMPORTANT]
> 本目录记录已经确认的设计基线，不代表对应能力已经实现。当前可用范围仍以根
> [README](../../README.md#当前能力) 和[支持矩阵](../user-guide/support-matrix.md)为准。

RM Relay 面向完整的 RM 软件组。嵌入式、电控、视觉、导航和普通 Linux 应用可以使用
不同环境与设备后端，但共享一条开发链路：源码在本地或远端完成构建，构建结果回到开发者
工作区，再进入物理或虚拟 target，最终把开发数据带回本地并清理目标环境。

用户负责应用源码、算法、启动方式和少量项目声明。RM Relay 负责应用源码以下的公共
基础设施，包括环境、构建后端、构建结果、target 接入、调试连接和 Development Session。
平台不会解释 ROS node、launch 文件或普通进程的内部组织，也不替用户设计应用的 `run`
命令。

当前基本链路止于 Development Session。比赛用持久部署、开机自启、批量分发和目标系统
镜像属于后续可选模块，见[路线图](../../ROADMAP.md#后续可选模块)。

## 两条不能混淆的链路

### 环境生产链路

这条链路由 RM Relay 维护者和战队运维人员使用。它生产固定版本的开发环境，不编译用户
每天修改的应用源码。

```text
能力配置
Dockerfile stages + mise fragments
            │
            ▼
    Docker Buildx Bake
    定义有限的官方 profile
            │
            ▼
         BuildKit
  跨架构镜像可使用 QEMU
            │
            ▼
       OCI Registry
        │         │
        │         └──────── runtime image
        │                      │
        ▼                      ▼
development image        Linux target
        │
        ▼
本地或远程 workspace 构建
```

Dockerfile 固定系统血统和镜像阶段，mise 片段组织各项开发能力，Bake 只发布经过验证的
场景组合。项目需要额外依赖时，以官方 profile 为基础构建派生镜像；运行中的容器不得
临时安装依赖来改变环境。

### 日常开发链路

```text
本地项目
├── 应用源码
├── project_id
├── profile 声明
├── CMake Presets / colcon 配置
└── mise tasks
        │
        ▼
统一开发入口
        │
        ├──────── local backend
        │         本地 Docker
        │
        └──────── remote backend
                  workspace 构建服务
        │
        ▼
development image
CMake / colcon / Ninja / ccache
        │
        ▼
构建结果返回本地
├── CMake Install Tree
├── ROS 2 Install Space
└── MCU ELF / BIN / MAP
        │
        ▼
target adapter
        │
        ├──────── MCU target
        │         OpenOCD / DFU / serial
        │         GDB 调试
        │
        └──────── Linux target
                  ├── physical target
                  └── virtual target
                          │
                          ▼
                  Development Session
                  ├── 排他占用 target
                  ├── 部署与增量传输
                  ├── 完整交互式 CLI
                  ├── 用户自行运行程序
                  ├── debugger 直连 target
                  └── /workspace/data
                          │
                          ▼
                  数据返回本地
                  .rm-relay/data/
                          │
                          ▼
                  清理 Session
```

无论使用本地还是远端构建，构建结果都先进入本地工作区。开发者电脑继续充当控制端，负责
选择 target、发起传输和连接 debugger。构建服务器不转发实时调试数据，也不直接接管
目标设备。

## 组件职责

| 组件 | 在 RM Relay 中的职责 | 不负责 |
|---|---|---|
| mise | 固定工具版本、组织环境变量和用户任务入口 | 构建系统、远程控制平面、Session 状态 |
| CMake Presets / CMake | 普通 C/C++ 项目的配置、构建、测试和安装规则 | ROS workspace 编排、远程执行 |
| colcon | 编排 ROS 2 workspace 中的 package | target 部署和设备管理 |
| Ninja | 执行 CMake 或 ament 生成的构建图 | 环境、依赖和部署语义 |
| Dockerfile / Bake | 定义镜像阶段、官方 profile、架构和发布矩阵 | 用户应用的日常构建入口 |
| BuildKit | 镜像构建、远程执行、缓存和构建结果导出 | target、交互终端和调试会话 |
| ccache | 缓存 C/C++ 编译单元 | Docker layer 和项目构建目录 |
| OCI Registry | 保存和分发固定版本的环境镜像 | 用户源码与 Session 数据 |
| SSH | 连接远程服务和 Linux target | ROS 应用协议和文件生命周期 |
| OpenOCD / GDB | MCU 烧录、调试以及 Linux 源码调试 | 构建与应用进程编排 |
| Docker Compose | 管理 Linux Session 的容器、网络和 volume | 用户身份、构建任务和源码同步 |

RM Relay 维护这些组件之间的配置、契约和验证，不重新实现编译器、同步算法、IDE、通用
任务队列或远程开发平台。

## 阅读路径

- [环境与 profile](environments-and-profiles.md)：环境镜像如何分层、组合和扩展。
- [构建与输出](builds-and-outputs.md)：源码如何在本地或远端构建，输出和 cache 如何分开。
- [Target 与 Session](targets-and-sessions.md)：构建结果如何进入设备，调试和数据如何返回。
- [服务拓扑](service-topology.md)：镜像构建、Registry、workspace 构建和虚拟执行服务如何组合。
- [开发契约参考](../reference/development-contracts.md)：术语、身份、路径、生命周期和不变量。

仓库当前仍是 monorepo。某个模块只有在形成独立使用者、依赖、发布节奏和维护者后，才考虑
拆成 RM Relay umbrella project 下的子仓库。模块是否独立维护，不改变它在基本链路中的
产品责任。
