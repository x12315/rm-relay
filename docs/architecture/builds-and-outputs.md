# 构建与输出

本文解释开发闭环的中段：固定环境如何消费用户源码，并以什么边界把结果交给 target。
它不说明烧录或部署步骤；这些操作从 Build Output 开始，见
[Target 接入与数据链路](targets-and-access.md)。

> [!IMPORTANT]
> 当前仓库已经实现嵌入式 local Docker 与 remote BuildKit backend，共用
> `install/<profile>` 输出边界。Remote backend 已由单测和 Compose 配置验证，尚未取得真实战队
> 服务器证据。使用入口见[Builder 配置](../user-guide/builders.md)。

## 先分开两条看似相同的构建链路

环境镜像生产和用户 workspace 构建都可能使用 Docker/BuildKit，但它们回答不同问题：

| 链路 | 输入 | 输出 | 何时运行 |
|---|---|---|---|
| 环境镜像生产 | RM Relay Dockerfile、mise 能力配置、Bake target | development/runtime image | 环境或 profile 变化时 |
| 用户 workspace 构建 | 固定 development image、项目声明、源码 | Build Output | 日常修改源码时 |

前者固定“用什么构建”，后者执行“构建什么”。日常 workspace 构建不要求用户维护应用
Dockerfile，也不把用户应用制作成 OCI image。

## 一份项目声明，两种 backend

Local 与 remote backend 的入口和出口保持相同：

```text
开发机源码 + 项目声明 + development profile
                  │
          ┌───────┴────────┐
          ▼                ▼
 local Docker        workspace builder
          │                │
          └───────┬────────┘
                  ▼
         Build Output 返回开发机
```

`rm-relay` 协调容器、远程 backend 与 target。Local backend 直接调用 Docker，
development image 内的 mise 再通过私有配置进入对应 Workflow；用户项目只声明 build
`system`、`preset` 和输出角色。编译和测试仍由 CMake、colcon、Ninja、CTest 等原生工具
执行，用户也可以绕过 `rm-relay` 直接调用这些工具。

通用 C/C++ 项目以 CMake 为官方基线，ROS 2 workspace 在外层使用 colcon。首版不再增加
Meson、Xmake、Bazel、Nix 或自定义构建描述；ccache 通过 CMake compiler launcher 或
colcon 的底层 CMake 参数接入，不改变构建图。

Remote backend 使用 BuildKit 的 context 传输、cache 和 local exporter。编译服务器运行 RM
Relay 维护的通用 workspace 构建定义，不读取用户自带的第二份应用 Dockerfile。一次 job
结束并把结果写回开发机后，源码快照和临时 workspace 可以删除；项目特有构建知识仍随源码
存在。

Workspace builder 不直接部署任何 target。这个中断点是有意设计的：local/remote build
可以复用同一条下游 target 链路，构建服务和设备接入也能独立部署或替换。

## Build Output 是唯一交接面

Build tree 包含 object、CMake cache、绝对路径和中间文件，既不可移植，也泄露 backend
内部布局。Target adapter 只消费构建系统现有的可部署输出：

| 项目类型 | 目标 Build Output | 不交付的内容 |
|---|---|---|
| 普通 CMake | CMake Install Tree | `CMakeFiles/`、object、CMake cache |
| ROS 2 workspace | colcon Install Space | build/log space 与中间 package 状态 |
| MCU | ELF、BIN、MAP 等固件文件 | object、临时链接文件 |

统一目录契约是：

```text
<project-root>/
├── build/<profile>/       backend 的可删除中间目录
├── install/<profile>/     开发机可见的 Build Output
└── .rm-relay/data/        从 target 取回的 Managed Data
```

RM Relay 当前不定义新的应用包格式或强制压缩包。CMake Install Tree、ROS 2 Install Space
和 MCU 固件文件已经能表达下游需要的内容。

当前 MCU 模板由 RM Relay 内部的 CMake Workflow 执行 configure、build 与 install，项目
不携带 mise task。两种 backend 成功后检查 Profile 要求的输出角色，并在
`install/<profile>/rm-relay-output.json` 的 schema v2 记录：

- `schema_version`、`project_id`、`profile_id`、`profile_digest` 和 `producer_version`；
- Builder 的逻辑 `id` 与实现 `kind`；
- environment 的 `id`、实际 `reference` 与 `digest`；
- 每个 artifact 的语义 `role`、相对 `path`、`size` 与 `sha256`。

Remote backend 先将 local exporter 写入
受管临时目录，确认声明产物存在后再原子发布整个 install tree；失败构建不会留下可供 target
消费的新 manifest。Target adapter 只接收重新校验过的 Build Output；PI 示例仍保留直接 CMake
构建，用于验证构建系统本身。

## Cache 只改变速度

构建链路中存在四类状态：

| 状态 | 所在位置 | 是否项目资产 |
|---|---|---|
| 开发机 build tree | `build/<profile>/` | 否，可重新生成 |
| Build Output | `install/<profile>/`（目标契约） | 是，开发机可见 |
| Managed Data | `.rm-relay/data/` | 是，取回后由开发机保管 |
| BuildKit、ccache、依赖 cache、remote workspace | backend 管理 | 否，可删除 |

切换 builder 或清空 cache 不得改变构建语义。首版不在开发机与编译服务器之间复制 cache，
也不把 cache 传到 target。

ccache 是显式依赖，不是 CMake 默认能力。可信战队或邀请制 backend 可以让相同环境与
工具链的用户共享 ccache，但 job workspace 和 build tree 仍彼此隔离。若以后提供不受信任
的公共服务，可以改为公共只读种子 cache 与用户私有写 cache。无论采用哪种布局，cache
命中都不能证明身份、权限、完整性或构建正确性。

## 跨架构构建

首条算力侧链路假定 x86 编译服务器与 ARM64 Linux target。这里仍要分开镜像生产和源码
编译：

```text
环境镜像生产
x86 BuildKit ── QEMU 执行 ARM64 包安装 ──→ ARM64 image

用户 workspace
x86 host ── aarch64 cross toolchain ──→ ARM64 Install Tree / Install Space
```

QEMU 不参与大型用户 workspace 编译。Cross compile 失败时应直接报告不兼容，不能悄悄
退回 QEMU；原生 ARM64 builder 是后续的性能与兼容性扩展。

### Sysroot 与 runtime 必须来自同一基线

Cross builder 使用隔离的 ARM64 sysroot：

```text
同一 target package 基线
├── runtime packages ───────────────→ ARM64 runtime image
└── runtime + development packages ─→ ARM64 sysroot ─→ x86 cross builder
```

Sysroot 包含 headers、linker 文件、CMake Config 和 `pkg-config` metadata。它不能独立求解
依赖，也不能把所有 ARM64 package 装进 x86 builder 根目录。Runtime image 与 sysroot 都
记录实际 package 清单，并检查共同 runtime package 的版本。

ROS 2、Nav2、OpenCV 等成熟依赖优先使用目标架构 binary package。正式支持范围包括官方
模板和已验证 profile；随 workspace 构建的第三方源码要通过 cross-build contract test。
构建期必须执行 target 程序、混用 host/target 生成器或依赖特殊架构工具的 package，首版
不保证兼容。

## 构建证据不能替实机证据

跨架构输出按顺序检查：

1. ELF architecture、dynamic loader、`DT_NEEDED`、RPATH 和 symbol version；
2. Runtime 中的程序、project library、plugin 与 ROS resource index；
3. 真实 target 上的 kernel、驱动、设备和调试行为。

前两层只证明 Build Output 与用户态 runtime 匹配，不能推出真实硬件已经兼容。支持结论必须
沿用[支持矩阵](../user-guide/support-matrix.md)中的证据等级。

## 仍待组件设计确定的内容

Remote workspace 构建定义已经由固定 frontend、不可变 environment 与 local exporter 落实。
远程失败后的断点恢复和 runtime compatibility schema 尚未确定。后续设计必须沿用现有 Project、
Profile、Execution Plan 与 Build Output 边界：项目声明随
源码存在，服务端保持通用，Build Output 先回开发机，cache 不成为项目真相源。
