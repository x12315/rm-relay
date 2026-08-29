# 环境与 profile

> [!IMPORTANT]
> 当前仓库只交付 `mcu-dev` 嵌入式开发镜像。本页其余内容是算力侧环境需要遵守的设计
> 基线，不是现有镜像清单。

RM Relay 按开发场景发布少量经过验证的官方 profile，不提供同时包含 MCU、ROS 2、视觉、
导航和所有厂商 SDK 的单一镜像。一般开发者选择 profile；镜像维护者负责能力层、构建矩阵
和验证。

## 能力层与成品 profile

能力层描述可复用的技术组成：

```text
common
├── native-cpp
├── embedded
├── ros2
├── computer-vision
├── navigation
└── vendor-runtime
```

成品 profile 描述可以直接使用的开发场景，例如嵌入式开发、原生 C++ 自动瞄准、ROS 2
自动瞄准或导航。一个组合只有同时满足以下条件，才属于正式产品：

- 进入 Bake 发布矩阵；
- 通过对应架构和目标环境的验证；
- 在支持矩阵中记录证据等级。

能力层不按战队小组或某块开发板组织。RoboMaster C、特定相机 SDK 和 AX650N 属于 board、
vendor 或 target profile，不能反向决定仓库拓扑。

## 配置职责

```text
Dockerfile stages
        │
        ▼
mise 能力片段
        │
        ▼
docker-bake.hcl
        ├── 官方 development/runtime image
        └── 对应的 Dev Container Template

官方 image + 项目 mise overlay
        │
        └── 项目派生镜像
```

| 配置层 | 负责的内容 | 不负责的内容 |
|---|---|---|
| Dockerfile stage | 操作系统、软件源、系统包、用户、目录和架构 | 正式发布组合 |
| mise 能力片段 | 开发工具版本、环境变量和可复用任务 | 跨机器状态和 target 生命周期 |
| `docker-bake.hcl` | 官方 profile 组合与多架构发布矩阵 | 用户项目特有依赖 |
| Dev Container Template | 每个官方 profile 独立描述 mount、USB/GPU 接入和 IDE 建议 | 构建逻辑的唯一入口 |
| 项目 overlay | 项目特有的环境扩展 | 修改运行中的官方环境 |

普通用户通过 mise task 进入工作流。涉及跨机器、target 状态或平台差异时，task 调用
`rm-relay`；纯构建和调试任务可以直接调用 CMake、colcon、OpenOCD 等原生工具。mise
不替代包管理器或构建系统，`rm-relay` 也不复制这些工具的能力。

## 环境在镜像构建时固化

项目增加依赖时，必须通过 overlay 生成新的派生镜像；在战队中由运维人员审查 overlay：

```text
官方 image + 项目 mise overlay
            ↓
         派生镜像
            ↓
        开发容器启动
```

运行中的容器不得通过 mise、APT 或 pip 改变正式环境。这一限制让 Registry 和 Docker layer
cache 可以复用，也让本地构建、远程构建与 target environment 通过版本或内容摘要确认
兼容关系。

普通成员不需要维护派生镜像。社区可以分享“开源项目 + 官方 profile”的 overlay，但
RM Relay 只验证进入正式矩阵的组合，不承诺任意 Dockerfile 或能力组合可用。

## Development 与 runtime 环境

MCU 固件直接在芯片上运行，只需要 development image：

```text
embedded development image
        ↓
ELF / BIN / MAP
        ↓
MCU
```

算力侧 profile 同时包含开发环境和目标环境：

```text
target environment lineage
├── runtime image
│   └── 系统库、ROS 2、Python 或厂商 runtime
└── development image
    └── 编译器、headers、CMake、colcon、测试和调试工具
```

官方 runtime image 不包含用户应用。日常开发把 CMake Install Tree 或 ROS 2 Install Space
送入固定的 Target Environment。

同架构 native build 可以让 development image 复用 runtime image 的基础 layer。跨架构
构建时，x86 development image 无法继承 ARM64 runtime image；两者必须共享同一 target
package 基线和 ABI 契约。构建侧的具体关系见
[构建与输出](builds-and-outputs.md#跨架构构建)。

## 跨架构镜像生产

当战队只有 x86 服务器时，BuildKit 可以通过 QEMU 生产 ARM64 镜像。QEMU 只在镜像构建
期间执行目标架构的包安装与配置命令；产物仍是普通的 ARM64 文件系统和 ELF。

用户 workspace 使用 host 原生执行的 cross toolchain，不走 QEMU 转译。原生 ARM64
builder 可以在后续用于提速和扩大兼容面，但不是首条开发链路的前提。

## Target host contract

容器固定用户态环境，目标宿主机仍需满足 kernel、驱动和设备接入要求：

| RM Relay 环境负责 | Target host contract 负责 |
|---|---|
| 用户态 library 和 runtime | kernel 与驱动 |
| ROS 2、Python 和 vendor user space | Docker 或其他 container runtime |
| 构建、测试与调试工具 | 设备节点、权限和网络 |
| 经过验证的 target package 基线 | 时钟、实时调度及 GPU/NPU 驱动兼容性 |

每个算力侧 target profile 都必须声明宿主要求。具体 schema 尚未确定；在 schema 与实机验证
完成前，“容器能够构建”只证明环境配置成立，不能作为目标硬件兼容结论。
