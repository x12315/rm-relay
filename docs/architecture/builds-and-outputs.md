# 构建与输出

> [!IMPORTANT]
> 本页同时描述当前嵌入式构建和规划中的统一构建链路。已经实现的 backend、profile 与
> target 组合以[支持矩阵](../user-guide/support-matrix.md)为准。

RM Relay 必须将环境镜像生产与用户 workspace 构建分成两条链路。两者都可以使用 Docker 和
BuildKit，但输入、频率和产物不同：

| 链路 | 输入 | 产物 | 使用频率 |
|---|---|---|---|
| 环境镜像生产 | RM Relay 的 Dockerfile、mise 能力配置和 Bake target | development/runtime image | 环境依赖或 profile 变化时 |
| 用户 workspace 构建 | 固定 development image、项目声明和源码 | Build Output | 日常源码变化时 |

日常 workspace 构建不要求用户维护应用 Dockerfile，也不把应用制作成 OCI image。

## 构建入口

mise 组织项目任务，`rm-relay` 协调跨容器、跨机器和 target 相关操作。编译与测试仍由各
生态的原生工具完成：

```text
mise
├── 原生任务
│   ├── CMake Presets → CMake → Ninja
│   ├── colcon → ament/CMake → Ninja
│   └── CTest / colcon test
└── rm-relay
    ├── local / remote build backend
    └── target adapter
```

CMake 是官方通用 C/C++ 项目的构建基线。ROS 2 workspace 在外层使用 colcon；官方基线
不再增加 Meson、Xmake、Bazel、Nix 或自定义构建描述。用户可以绕过 mise 与 `rm-relay`，
直接调用 CMake、colcon 和 Ninja。ccache 通过 CMake compiler launcher 或 colcon 的底层
CMake 参数接入，不改变构建图。

## 本地与远程 backend

两种 backend 消费同一份项目声明和 development profile：

```text
本地源码
    │
    ├── local backend ── 本地 Docker container ──┐
    │                                             │
    └── remote backend ─ workspace builder ───────┤
                                                  ▼
                                      Build Output 返回本地
```

远程 backend 使用 BuildKit 的 context 传输、cache 和输出导出能力。服务器运行 RM Relay
维护的固定 workspace 构建定义，不从用户项目读取另一份应用 Dockerfile。BuildKit local
exporter 把结果写回客户端指定目录，RM Relay 不在这段链路上增加自定义传输包或同步协议。

服务器只在一次 job 内持有源码快照和临时 workspace。项目特有的构建知识留在源码中的
项目声明；服务端保存的 BuildKit cache、ccache 和依赖 cache 都可以删除。

Build Output 必须返回开发者工作区，再由客户端送往 target。workspace builder 不直接部署
物理或虚拟设备。这一边界让 local build 与 remote build 复用同一条 target 链路，也允许
构建服务和 target 服务独立部署。

## Build Output 边界

build tree 包含 object、CMake cache、绝对路径和中间文件，不能作为 target adapter 的输入。
可部署输出使用构建系统现有的 install model：

```text
普通 CMake
build/<profile>/
    ↓ cmake --install
install/<profile>/
    ├── bin/
    ├── lib/
    └── share/

ROS 2
colcon build
    ↓
Install Space

MCU
install/<profile>/
    ├── firmware.elf
    ├── firmware.bin
    └── firmware.map
```

CMake Install Tree、ROS 2 Install Space 和 MCU 固件文件都是 Build Output。烧录、传输和
debugger 只消费这层输出。RM Relay 当前不定义新的应用包格式或强制压缩包。

## 资产与 cache

```text
项目工作区
├── build/<profile>/           本地构建中间目录
├── install/<profile>/         本地可见的 Build Output
└── .rm-relay/data/            target 返回的 Managed Data

backend 管理
├── BuildKit cache
├── ccache
├── 依赖下载 cache
└── 远程 job workspace
```

Build Output 是项目资产，cache 只用于加速。切换 builder 或清空 cache 不得改变构建语义。
首版不在本地与远端之间同步 cache，也不把 cache 传到 target。

ccache 是显式依赖，不是 CMake 默认能力。可信的战队或邀请制远程 backend 可以让使用
同一环境与工具链的用户共享 ccache；ccache 按编译器、参数和输入内容判断命中，各 job 的
workspace 和 build tree 仍相互隔离。本地 ccache 只保存在开发者电脑。

若以后开放不受信任的公共构建服务，可以采用公共只读种子 cache 和用户私有写 cache。
无论如何布局，cache 都不能参与身份或权限判断，也不能成为构建完整性的证据。

## 跨架构构建

首条算力侧链路面向“x86 构建服务器、ARM64 Linux target”，镜像和 workspace 使用不同
路径：

```text
ARM64 环境镜像
x86 BuildKit → QEMU 执行目标架构安装过程 → ARM64 image

用户 workspace
x86 host → aarch64 cross toolchain → ARM64 Install Tree / Install Space
```

两条路径不得互相 fallback。QEMU 不参与大型用户 workspace 编译；cross compile 失败时
必须直接报告不兼容。原生 ARM64 builder 是用于提速和扩大兼容面的后续扩展。

### Target sysroot

Cross builder 使用隔离的 ARM64 sysroot：

```text
同一 target package 基线
├── runtime packages
│   └── ARM64 runtime image
└── runtime packages + development packages
    └── ARM64 target sysroot
            │
            └── x86 cross builder
```

Sysroot 通过 Ubuntu/Debian APT 构造，包含 headers、linker 文件、CMake Config 和
`pkg-config` metadata。它与 runtime image 必须使用同一组 runtime package 版本，不得独立
求解依赖，也不能把所有 ARM64 package 直接安装到 x86 builder 根目录。两边都记录实际
package 清单，并检查共同 runtime package 的版本。

ROS 2、Nav2、OpenCV 等成熟依赖优先使用目标架构的 binary package。RM Relay 只承诺对
进入支持范围的 workspace 完成交叉编译：

- 官方模板和已验证 profile 属于正式支持范围；
- 随 workspace 构建的第三方源码需要通过 cross-build contract test；
- 构建期执行 target 程序、混用 host/target 生成器或依赖特殊架构工具的 package，首版不作
  兼容保证。

## 兼容性证据

跨架构构建依次检查三个层级：

1. ELF architecture、dynamic loader、`DT_NEEDED`、RPATH 和 symbol version；
2. 匹配 runtime 环境中的程序、项目 library、plugin 与 ROS resource index；
3. 真实 target 上的设备、驱动和 kernel 相关行为。

前两层通过只证明产物与用户态 runtime 匹配，不能推导真实硬件已经兼容。验证结论继续使用
[支持矩阵](../user-guide/support-matrix.md)中的证据等级。

## 后续组件设计

远程 workspace 构建定义的文件格式、Build Output manifest、失败后的断点恢复和 runtime
compatibility schema 尚未确定。后续设计必须保持四项边界：项目构建声明随源码存在，服务端
保持通用，Build Output 先返回本地，cache 不成为项目真相源。
