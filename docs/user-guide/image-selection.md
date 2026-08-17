# 镜像选择与运行

## 能力边界

`base` 面向共享 C/C++ 代码的本机编译和质量检查，包含 CMake、Ninja、
Clang/GCC、GDB、CTest、Sanitizers、clang-format、clang-tidy、Python 和 uv。

`mcu-dev` 在 `base` 之上增加 GNU Arm Embedded、`gdb-multiarch`、OpenOCD 与
`dfu-util`，用于 Arm bare-metal 交叉编译、烧录和调试。当前只交付这两个
build target。

`compute-dev` 是未来承载普通 Linux、视觉、ROS 2、NVIDIA 或 AXERA 能力的家族名，
不是当前镜像，也不与 MCU 工具链耦合。

普通开发者通常直接使用已发布或由实验室维护者构建的 `mcu-dev/toolchain` 镜像。
如需在本机生成当前开发标签，再使用以下命令。

普通用户优先拉取实验室发布到国内 OCI 仓库的版本镜像。直接执行 Dockerfile 属于维护者
路径，仍可能访问 GitHub、GHCR 等少量上游来源。

## Apple Silicon 本机构建

```bash
docker buildx bake \
  --file container-images/embedded-development/docker-bake.hcl \
  mcu-dev-arm64 --load
```

## x86_64 Linux 本机构建

```bash
docker buildx bake \
  --file container-images/embedded-development/docker-bake.hcl \
  mcu-dev-amd64 --load
```

`--load` 只适合单一平台。默认的 `docker` driver 应分别验证两个架构：

```bash
docker buildx bake \
  --file container-images/embedded-development/docker-bake.hcl verify-arm64 \
  --set '*.output=type=cacheonly'
docker buildx bake \
  --file container-images/embedded-development/docker-bake.hcl verify-amd64 \
  --set '*.output=type=cacheonly'
```

配置了支持 multi-platform 的 `docker-container` 或远程 builder 后，可一次验证：

```bash
docker buildx bake \
  --file container-images/embedded-development/docker-bake.hcl multiarch \
  --set '*.output=type=cacheonly'
```

项目不自动创建或切换持久 Buildx builder。

当前基线是 Ubuntu 24.04 LTS、native GCC 14 和 Arm GNU 13.2.Rel1。产品级版本见
[`container-images/embedded-development/locks/versions.env`](../../container-images/embedded-development/locks/versions.env)。
镜像中的 `/opt/embedded-development/base-packages.txt` 与
`/opt/embedded-development/embedded-packages.txt` 记录该次构建实际安装的完整版本。

镜像维护、跨架构验收和未来发布方式见
[镜像构建与验证](../operator-guide/build-and-verify-images.md)。

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

烧录和调试使用工具原生命令；项目不增加统一 CLI。任何写入操作都应先根据设备文档
确认芯片、接口、alternate setting（DFU 备用接口）和目标地址。
