# 构建与输出

RM Relay 必须区分环境镜像生产和用户 workspace 构建。两者都可能使用 Docker 与
BuildKit，但输入、频率和责任完全不同。

```text
环境镜像生产
RM Relay 配置 → BuildKit → development/runtime image → Registry

用户 workspace 构建
固定 development image + 用户源码 → CMake/colcon → 本地构建结果
```

普通开发者不会为了每次修改源码而重新编写 Dockerfile，也不需要把应用制作成 OCI image。

## 构建入口

mise 是统一任务入口，不是构建系统。实际构建继续使用社区原生工具：

```text
mise
├── CMake Presets → CMake → Ninja
├── colcon → ament/CMake → Ninja
├── CTest / colcon test
└── ccache
```

CMake 是通用 C/C++ 项目的构建能力上限。ROS 2 workspace 在外层使用 colcon，不再引入
Meson、Xmake、Bazel、Nix 或另一套自定义构建描述作为官方基线。用户仍可直接调用原生
CMake、colcon 和 Ninja，mise 不能成为封闭入口。

## 本地与远程 backend

两种 backend 消费同一项目声明和同一 development profile：

```text
本地源码
    │
    ├── local backend
    │   └── 本地 Docker container
    │
    └── remote backend
        └── workspace 构建服务中的临时任务
                │
                ▼
          构建结果返回本地
```

远程 backend 使用 BuildKit 的 context 传输、cache 和输出导出能力。服务端运行由 RM Relay
维护的固定 workspace 构建定义，用户项目不因此增加一份应用 Dockerfile。构建服务不长期
托管源码 workspace，也不保存项目特有的构建知识。

远程构建完成后，输出必须回到开发者工作区，再由本地任务送往 target。构建服务器不会直接
部署物理设备或虚拟设备。这条边界保证本地 Docker 与远程服务可以共享下游流程，也允许各项
服务独立部署。

## 构建目录不是交付边界

build tree 包含 object、CMake cache、绝对路径和中间文件，不能交给 target adapter 猜测。
可部署输出使用已有构建系统的 install model：

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

CMake Install Tree、ROS 2 Install Space 和 MCU 固件文件是同一层级的 Build Output。烧录、
传输和 debugger 只消费 Build Output，不直接依赖 build tree。RM Relay 当前不定义新的
压缩包或应用包格式，也不要求每次构建用户应用镜像。

## 输出与 cache 的所有权

```text
项目工作区
├── build/<profile>/           本地构建中间目录
├── install/<profile>/         本地可见的 Build Output
└── .rm-relay/data/            target 返回的数据

backend 管理
├── BuildKit cache
├── ccache
├── 依赖下载 cache
└── 远程 job workspace
```

Build Output 是项目资产；cache 只是加速手段。切换 builder 或清空 cache 不得改变构建
语义。首版不在本地和远端之间同步 cache，也不把 cache 传到 target。

ccache 是显式依赖，不是 CMake 默认能力。CMake 通过 compiler launcher 接入它；cache
namespace 需要包含项目、profile 和工具链信息，具体布局由构建 backend 管理。

## 跨架构构建

首版面向“战队只有 x86 服务器、目标是 ARM64 Linux”的现实，采用两条固定路径：

```text
ARM64 环境镜像
x86 BuildKit → QEMU 执行目标架构安装过程 → ARM64 image

战队 workspace
x86 host → aarch64 cross toolchain → ARM64 Install Tree / Install Space
```

两条路径不会互相 fallback。QEMU 不参与大型用户 workspace 编译；cross compile 失败时
必须直接报告不兼容，不能静默改用转译构建。原生 ARM64 builder 是后续扩展，用于提速和
扩大兼容面。

### target sysroot

cross builder 使用隔离的 ARM64 sysroot：

```text
同一 target package 基线
├── runtime packages
│   └── ARM64 runtime image
└── runtime packages + development packages
    └── ARM64 target sysroot
            │
            ▼
      x86 cross builder
```

sysroot 通过 Ubuntu/Debian APT 构造，包含 headers、linker 文件、CMake Config 和
`pkg-config` metadata。它不能与 runtime 独立求解另一套依赖，也不能把所有 ARM64 包
直接混入 x86 builder 根目录。两边应记录实际包清单，并验证共同 runtime package 的版本
一致。

ROS 2、Nav2、OpenCV 等成熟依赖优先使用目标架构的 binary package。RM Relay 交叉编译
的是战队自己的 workspace，不承诺任意第三方源码 package 无适配完成 cross compile：

- 官方模板和已验证 profile 属于正式支持范围；
- 随 workspace 编译的第三方源码需要通过 cross-build contract test；
- 构建期执行 target 程序、混淆 host/target 生成器或依赖特殊架构工具的 package，首版不作
  兼容保证。

## 兼容性验证

跨架构构建至少分三层验证：

1. 检查 ELF architecture、dynamic loader、`DT_NEEDED`、RPATH 和 symbol version。
2. 在干净的匹配 runtime 环境中加载程序、项目 library、plugin 和 ROS resource index。
3. 在真实 target 上验证设备、驱动和内核相关行为。

通过前两层不能推导真实硬件已兼容。验证结论继续使用
[支持矩阵](../user-guide/support-matrix.md)记录的证据等级。

## 尚未定案

远程 workspace 构建定义的具体文件格式、Build Output manifest、失败后的断点恢复以及
runtime compatibility schema 尚未确定。实现这些内容时必须保持本页已经确认的边界：
项目构建定义随源码存在，服务端保持通用，构建结果先回本地，cache 不成为项目真相源。
