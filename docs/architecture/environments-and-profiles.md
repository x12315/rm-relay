# 环境与 profile

> [!IMPORTANT]
> 当前仓库只交付 `mcu-dev` 嵌入式开发镜像。本页其余内容是后续算力侧环境必须遵守的
> 设计基线，不是现有镜像清单。

RM Relay 不发布一个包含 MCU、ROS 2、视觉、导航和所有厂商 SDK 的全家桶镜像。镜像按
可复用能力分层，最终只发布少量符合真实开发场景、经过验证的官方 profile。普通成员选择
profile；维护者才接触底层能力片段。

## 两个组织维度

能力层回答“镜像由哪些技术能力组成”，例如：

```text
common
├── native-cpp
├── embedded
├── ros2
├── computer-vision
├── navigation
└── vendor-runtime
```

成品 profile 回答“哪类项目可以直接使用”，例如嵌入式开发、原生 C++ 自动瞄准、ROS 2
自动瞄准或导航。这里的名称只是场景说明；只有进入 Bake、通过验证并写入支持矩阵的 profile
才算正式产品。

能力层不按战队小组或具体板卡组织。RoboMaster C、某个相机 SDK 或 AX650N 只会成为
board、vendor 或 target profile，不会反向决定仓库拓扑。

## 配置层次

```text
Dockerfile
固定 Ubuntu、软件源、用户、目录、架构和镜像阶段
        │
        ▼
官方 mise 能力片段
声明开发包、工具、环境变量和基础任务
        │
        ▼
docker-bake.hcl
选择经过验证的能力组合，形成官方 profile
        │
        ▼
Dev Container Template
描述宿主接入、mount、USB/GPU 和 IDE 建议
        │
        ▼
项目 mise overlay
增加该项目特有的依赖、工具和任务
```

各层只有一个职责：

- Dockerfile 表达镜像的系统血统和 BuildKit stage。
- mise 能力片段减少重复的包清单和任务定义，但不决定发布哪些镜像。
- Bake 是官方 profile 组合的唯一真相源，不生成任意能力的笛卡尔积。
- 每个官方 profile 使用独立的 Dev Container Template，避免在一个模板中堆积大量条件。
- 用户项目可以增加 mise overlay，但不能改变官方 profile 的基础契约。

项目不再叠加 Task、just 或自制 CLI。mise 调用 CMake、colcon、OpenOCD、GDB、SSH 和
Docker 等原生工具；复杂的平台差异可以放在小型脚本中，但脚本不会形成第二套用户入口。

## 环境在镜像构建时固化

依赖只能在镜像构建阶段安装：

```text
官方 image + 项目 mise overlay
            ↓
       派生 image
            ↓
       开发容器启动
```

不允许容器启动后再运行 mise、APT 或 pip 修改环境。这样可以利用 Registry 和 Docker
layer cache，也能保证本地、远程构建服务和目标 runtime 使用可识别的版本。

普通成员不需要维护派生镜像。战队运维人员选择官方 Template，审查项目 overlay，并把
生成的固定环境交给队员使用。社区以后可以分享“某个开源项目 + 某个官方 profile”的
overlay，但不接受无法验证的任意 Dockerfile 组合。

## 嵌入式与算力侧的差异

MCU 固件在芯片上直接运行，因此嵌入式产品只需要 development image：

```text
embedded development image
        ↓
ELF / BIN / MAP
        ↓
MCU
```

算力侧需要 development 与 runtime 两种角色：

```text
target environment lineage
├── runtime image
│   └── 目标程序所需的系统库、ROS 2、Python 或厂商 runtime
└── development environment
    └── 编译器、headers、CMake、colcon、测试和调试工具
```

RM Relay 不把用户应用打进 runtime image。日常开发只把 CMake Install Tree 或 ROS 2
Install Space 送入固定 runtime 环境。

同架构 native build 可以让 development image 继承 runtime layer。跨架构构建时，
x86 development image 无法继承 ARM64 runtime image；两者必须共享同一 target package
基线和 ABI 契约，而不是强求 Docker layer 继承。具体关系见
[构建与输出](builds-and-outputs.md#跨架构构建)。

## 跨架构镜像生产

战队只有 x86 服务器时，ARM64 环境镜像允许由 BuildKit 通过 QEMU 构建。QEMU 在镜像
构建期间执行 ARM64 的包安装与配置命令，镜像中得到的仍是正常 ARM64 文件系统和 ELF。

这条路径只用于环境镜像生产，不用于编译大型用户 workspace。用户源码采用 host 原生运行
的 cross toolchain；拥有原生 ARM64 builder 的战队以后可以按架构调度 native build，
但它不是首版成立的前提。

## 主机兼容边界

development/runtime 环境可以约束用户态依赖，不能包办目标宿主机：

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

每个算力侧 target profile 都需要声明这层主机要求。具体 schema 尚未确定；在 schema
完成前，文档不能把“容器能够构建”写成“目标硬件已经兼容”。
