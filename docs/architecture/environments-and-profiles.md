# 环境与 profile

本文解释开发闭环的第一步：RM Relay 如何决定“在哪个环境里、为哪类 target 构建”。如果
你还不知道 Profile、Build Output 和 Target 的关系，请先读
[架构入口](README.md#先建立一个完整模型)。

> [!IMPORTANT]
> 当前仓库只交付 `mcu-dev` 嵌入式开发镜像。本文其余算力侧内容是后续实现必须遵守的
> 设计基线，不是已经发布的镜像清单。

## Profile 是经过验证的使用组合

Profile 不是“把若干工具装在同一个镜像里”的别名。它把开发场景所需的环境、架构和
target 兼容要求绑定为一个可验证组合。一个组合只有同时满足以下条件，才能成为正式
profile：

- 进入 Bake 发布矩阵；
- 通过对应架构和目标环境的验证；
- 在支持矩阵中记录实际证据等级。

普通开发者选择成品 profile，例如嵌入式开发、原生 C++ 视觉或 ROS 2 导航。镜像维护者才
需要处理组成这些 profile 的能力层：

```text
可复用能力层
common
├── native-cpp
├── embedded
├── ros2
├── computer-vision
├── navigation
└── vendor-runtime
        │
        ▼
经过 Bake 组合与验证的成品 profile
```

能力层按技术组成划分，不按战队小组或某块开发板划分。RoboMaster C、特定相机 SDK 和
AX650N 分别形成 board、vendor 和 target 兼容维度，不能反向决定仓库拓扑：board 约束硬件
与烧录/调试后端，vendor 约束厂商 SDK/runtime，target 约束目标宿主与运行环境；具体 schema
尚未确定。

算力侧按普通 Linux、原生 C++ 视觉、ROS 2 视觉、导航和 vendor runtime 等真实场景形成
有限的独立 profile，不发布单一 `compute-dev` 全家桶，也不把 ROS 2 当作全部 PC 或边缘
计算场景的边界。

## 从配置到可用环境

环境配置按责任逐层收敛：

```text
Dockerfile stage       固定 OS、系统包、用户和架构
        │
mise 能力片段           固定开发工具、环境变量和常用任务
        │
docker-bake.hcl         选择并发布官方 profile 组合
        │
Dev Container Template 描述 mount、设备接入和 IDE 建议
        │
development image      供 local/remote backend 消费
```

| 配置层 | 负责 | 不负责 |
|---|---|---|
| Dockerfile stage | OS、软件源、系统包、用户、目录、架构 | 决定哪些组合正式发布 |
| mise 能力片段 | 工具版本、环境变量、可复用任务 | 跨机器状态和 target 生命周期 |
| `docker-bake.hcl` | 官方 profile 与多架构发布矩阵 | 用户项目特有依赖 |
| Dev Container Template | profile 对应的 mount、USB/GPU 和 IDE 建议 | 成为唯一构建入口 |
| 项目 overlay | 项目特有的环境扩展 | 修改运行中的官方环境 |

目前仓库已经实现 Dockerfile/Bake 这部分：作为环境输出的 `base` 与 `mcu-dev` 都有
`linux/amd64`、`linux/arm64` target。Dockerfile 中的其他 helper stage 只为这些输出准备
文件。当前的 CMake Workflow 已由 development image 内的受控 mise task 执行；更多
能力组合、正式 Dev Container Template 和派生环境流程仍待实现。

环境镜像不打包 IDE、用户扩展或个人配置。Dev Container Template 可以给 VS Code 等编辑器
提供 mount、设备和任务建议，但必须继续调用 mise、CMake、OpenOCD 等已有入口，不能建立
另一套构建真相源。

## 两类核心 Template 固定不同入口

Project Template 与 Dev Container Template 都属于 RM Relay 的核心资产，但不解决同一个
问题：

| 核心资产 | 固定的入口 | 消费方式 | 当前状态 |
|---|---|---|---|
| Project Template | 用户项目的源码、CMake、测试和目标配置结构 | 当前由用户复制并改名；未来也可由 `rm-relay init` 交互式生成 | `project-templates/cross-platform-cpp/` 已实现 |
| Dev Container Template | profile 对应的 environment、mount、设备接入和 IDE 建议 | 用户按 profile 创建 development container | 尚未实现 |

Project Template 与项目声明、profile 和 Build Output 契约共同演进，不能作为可选插件拆出核心
仓库。Dev Container Template 也属于 profile 的环境交付，不等同于某个 IDE 的专用配置。

环境定义与可选 integration 以后可以形成独立仓库，但不改变上述两类核心 Template 的归属。
三个仓库分别组织什么、当前资产为何仍留在 monorepo，集中见
[仓库资产地图](../operator-guide/repository-assets.md#规划中的仓库边界)。

## 项目依赖通过派生镜像进入

官方 profile 只覆盖可共同验证的依赖。项目增加依赖时，运维人员审查项目 overlay，并在
开发开始前生成派生镜像：

```text
官方 image + 项目 mise overlay
            │
            ▼
          派生镜像
            │
            ▼
     local / remote build
```

运行中的正式容器不得通过 mise、APT 或 pip 改变环境。否则同一个 profile 名称可能在不同
机器上代表不同依赖，Registry layer cache、远程构建复现和 target 兼容判断都会失去依据。

普通成员只选择已经准备好的 image，不需要维护派生镜像。社区可以分享“开源项目 + 官方
profile”的 overlay，但 RM Relay 只承诺正式矩阵内经过验证的组合。

## MCU 与 Linux 环境为什么不同

MCU 固件在芯片上直接运行，因此只有 development image：

```text
embedded development image
        │
        ▼
   ELF / BIN / MAP
        │
        ▼
       MCU
```

Linux 应用同时依赖构建工具和目标用户态，所以 profile 要维护一条 Target Environment
lineage：

```text
同一兼容 lineage
├── runtime image
│   └── 系统库、ROS 2、Python 或厂商 runtime
└── development image
    └── 编译器、headers、CMake、colcon、测试与调试工具
```

官方 runtime image 不包含用户应用。日常开发把 CMake Install Tree 或 ROS 2 Install Space
送入固定的 Target Environment，而不是反复制作应用 OCI image。

同架构 native build 可以复用 runtime image 的基础 layer。x86 为 ARM64 交叉编译时，
development image 不能继承 ARM64 runtime 文件系统；两者必须通过 target package 基线和
ABI 契约保持一致。具体构建关系见[跨架构构建](builds-and-outputs.md#跨架构构建)。

## 镜像跨架构生产不等于项目交叉编译

战队只有 x86 服务器时，BuildKit 可以借助 QEMU 执行 ARM64 镜像中的包安装和配置命令，
最终产物仍是普通 ARM64 文件系统和 ELF。

用户 workspace 使用 host 原生执行的 cross toolchain，不走 QEMU。这样把低频环境生产和
高频源码编译分开；原生 ARM64 builder 以后可以用于提速，但不是首条链路的前提。

## 容器之外还有 host contract

容器只能固定用户态，不能替 target 宿主机提供 kernel、驱动和物理设备：

| Profile 环境负责 | Target host contract 负责 |
|---|---|
| 用户态 library 与 runtime | kernel 与驱动 |
| ROS 2、Python、vendor user space | Docker 或其他 container runtime |
| 构建、测试、调试工具 | 设备节点、权限、网络 |
| target package 基线 | 时钟、实时调度、GPU/NPU 驱动兼容性 |

每个面向算力侧 target 的 RM Relay Profile 都必须声明这些宿主要求。Host contract 的 schema
尚未确定；在 schema 和实机验证完成前，“镜像能构建”只能证明环境被配置，不能证明目标
硬件兼容。

## 本页保留的设计边界

后续实现可以调整配置文件格式，但必须保持：正式组合经过发布矩阵和验证；依赖在镜像构建
阶段固化；MCU 不需要 runtime image；Linux development/runtime 共享兼容 lineage；镜像
生产中的 QEMU 不成为用户 workspace 编译的 fallback。
