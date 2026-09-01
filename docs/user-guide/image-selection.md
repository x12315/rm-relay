# 镜像选择与运行

## 能力边界

`base` 面向共享 C/C++ 代码的本机编译和质量检查，包含 CMake、Ninja、
Clang/GCC、GDB、CTest、Sanitizers、clang-format、clang-tidy、Python 和 uv。

`mcu-dev` 在 `base` 之上增加 GNU Arm Embedded、`gdb-multiarch`、OpenOCD 与
`dfu-util`，用于 Arm bare-metal 交叉编译、烧录和调试。当前只交付这两个
build target。

算力侧环境未来会按普通 Linux、原生 C++ 视觉、ROS 2、导航和厂商 runtime 形成有限的
官方 profile，不使用一个全家桶镜像承载所有能力。这些 profile 尚未交付，也不与 MCU
工具链耦合。设计边界见[环境与 profile](../architecture/environments-and-profiles.md)。

公开镜像尚未发布，因此普通用户目前没有稳定的 image 获取入口。本页只说明已获得 image 后的
使用边界；候选 image 的构建、双架构检查与发布准备属于
[镜像构建与验证](../operator-guide/build-and-verify-images.md)，不作为用户上手步骤。

当前基线是 Ubuntu 24.04 LTS、native GCC 14 和 Arm GNU 13.2.Rel1。产品级版本见
[`environments/embedded-development/locks/versions.env`](../../environments/embedded-development/locks/versions.env)。
镜像中的 `/opt/embedded-development/base-packages.txt` 与
`/opt/embedded-development/embedded-packages.txt` 记录该次构建实际安装的完整版本。

## 运行工作区

```bash
docker run --rm -it \
  -v "$PWD:/workspace" -w /workspace \
  mcu-dev/toolchain:local bash
```

容器只提供工具链。源码、构建目录和编辑器配置由工作区或用户环境管理。接下来从
[native 构建与测试](build-native.md)选择模板或示例项目。

## USB backend 的运行边界

- Linux 可在明确映射 USB 设备并配置权限后，从容器运行 OpenOCD 或
  `dfu-util`；也可选择在宿主运行。
- macOS 与 Windows 默认在宿主运行 USB backend，容器负责构建固件，并在需要时
  用 GDB 连接宿主调试服务。镜像中包含这些工具不代表 Docker Desktop 能直接透传
  USB 设备。
- Windows 当前仅为规划平台，需在真实 Windows 主机验证后才能更新支持状态。

当前 `rm-relay` 通过受管 Buildx resource 完成 local build，镜像内 mise 执行固定 CMake Workflow；
OpenOCD adapter 才通过宿主 mise 调用 OpenOCD。GDB 和 DFU 仍使用原生工具，不另建构建或
调试协议。任何写入操作都应先根据设备文档确认芯片、接口、alternate setting（DFU 备用
接口）和目标地址。
