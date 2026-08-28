# 环境与 profile

> [!IMPORTANT]
> 当前仓库只交付 `mcu-dev` 嵌入式开发镜像。本页其余内容是算力侧环境需要遵守的设计
> 基线，不是现有镜像清单。

RM Relay 不发布一个同时包含 MCU、ROS 2、视觉、导航和所有厂商 SDK 的全家桶镜像。
底层按可复用能力组织，最终只发布少量符合真实开发场景、经过验证的官方 profile。
一般开发者选择 profile；镜像维护者才接触能力片段和构建矩阵。

## 能力层与成品 profile

能力层回答“环境由哪些技术能力组成”：

```text
common
├── native-cpp
├── embedded
├── ros2
├── computer-vision
├── navigation
└── vendor-runtime
```

成品 profile 回答“哪类项目可以直接使用”，例如嵌入式开发、原生 C++ 自动瞄准、
ROS 2 自动瞄准或导航。只有进入 Bake、通过验证并写入支持矩阵的组合，才是正式产品。

能力层不按战队小组或某块开发板组织。RoboMaster C、特定相机 SDK 或 AX650N 属于 board、
vendor 或 target profile，不能反向决定仓库拓扑。

## 配置层次

```text
Dockerfile stages
固定操作系统、软件源、用户、目录、架构和系统包
        │
        ▼
mise 能力片段
组织开发工具、环境变量和可复用任务
        │
        ▼
docker-bake.hcl
选择经过验证的能力组合，发布官方 profile
        │
        ├──────── Dev Container Template
        │         描述 mount、USB/GPU 和 IDE 建议
        │
        └──────── 项目 mise overlay
                  声明项目特有环境扩展并生成派生镜像
```

各层只承担一种责任：

- Dockerfile 表达镜像的系统血统和 BuildKit stage。
- mise 片段减少工具、环境变量和任务定义的重复，不决定正式发布组合。
- Bake 是官方 profile 组合和多架构发布矩阵的唯一真相源。
- 每个官方 profile 使用独立的 Dev Container Template，避免堆积大量条件。
- 项目 overlay 只能在镜像构建阶段生效，不能修改已运行的官方环境。

普通用户通过 mise task 进入工作流。需要跨机器、target 状态或平台差异时，task 调用
`rm-relay`；纯构建和调试任务仍可直接调用 CMake、colcon、OpenOCD 等原生工具。mise
不是包管理器、构建系统或远程控制平面，`rm-relay` 也不复制这些工具的能力。

## 环境在镜像构建时固化

项目增加依赖时，结果必须成为新镜像：

```text
官方 image + 项目 mise overlay
            ↓
         派生 image
            ↓
        开发容器启动
```

运行中的容器不能再用 mise、APT 或 pip 改变环境。这样才能复用 Registry 和 Docker layer
cache，并让本地构建、远程构建和目标环境通过版本或内容摘要确认兼容关系。

普通成员不需要维护派生镜像。战队运维人员选择官方 Template，审查项目 overlay，再把固定
环境交给队员。社区以后可以分享“某个开源项目 + 某个官方 profile”的 overlay，但 RM Relay
不承诺验证任意 Dockerfile 和能力组合。

## 嵌入式与算力侧环境

MCU 固件在芯片上直接运行，只需要 development image：

```text
embedded development image
        ↓
ELF / BIN / MAP
        ↓
MCU
```

算力侧同时存在构建环境和目标环境：

```text
target environment lineage
├── runtime image
│   └── 运行项目所需的系统库、ROS 2、Python 或厂商 runtime
└── development image
    └── 编译器、headers、CMake、colcon、测试和调试工具
```

RM Relay 不把用户应用打进官方 runtime image。日常开发只把 CMake Install Tree 或 ROS 2
Install Space 送入固定目标环境。

同架构 native build 可以让 development image 复用 runtime image 的基础 layer。跨架构
构建时，x86 development image 无法继承 ARM64 runtime image；两者必须共享同一 target
package 基线和 ABI 契约，而不是强求 Docker layer 继承。具体关系见
[构建与输出](builds-and-outputs.md#跨架构构建)。

## 跨架构镜像生产

战队只有 x86 服务器时，ARM64 镜像允许由 BuildKit 通过 QEMU 构建。QEMU 只在镜像生产
期间执行目标架构的包安装与配置命令，产物仍是正常的 ARM64 文件系统和 ELF。

用户 workspace 不走这条转译路径，而使用 host 原生执行的 cross toolchain。原生 ARM64
builder 可以在后续用于提速和扩大兼容面，但不是首版成立的前提。

## 宿主边界

容器可以约束用户态依赖，不能包办目标宿主系统：

```text
RM Relay 环境负责
├── 用户态 library 和 runtime
├── ROS 2 / Python / vendor user space
└── 构建与调试工具

target host contract 负责
├── kernel 与驱动
├── Docker / container runtime
├── 设备节点与权限
├── 网络、时钟和实时调度
└── GPU / NPU 驱动兼容性
```

每个算力侧 target profile 都需要声明这层宿主要求。具体 schema 尚未确定；在 schema 和实机
验证完成前，不能把“容器能够构建”写成“目标硬件已经兼容”。
