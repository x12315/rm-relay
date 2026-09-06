# embedded-development 环境定义

本目录定义 RM Relay 当前的 MCU 开发环境。它面向环境维护者；普通开发者只消费已经发布并由
Builder 核验的不可变 OCI reference。

## 目录负责什么

- `Dockerfile` 定义 `base` 与 `mcu-dev` 两个可交付 stage；
- `docker-bake.hcl` 定义 `linux/amd64`、`linux/arm64` 和 multi-platform 构建矩阵；
- `identity.toml` 固定声明 `id = "embedded-development"`；
- `locks/` 记录 Ubuntu、编译器与工具的版本基线；
- `smoke/` 在镜像内检查工具版本，并让 GCC、Clang 和 Arm GNU 实际编译 C++20 probe。

`base` 提供 host C++20 构建、ccache 和质量工具；`mcu-dev` 增加 GNU Arm Embedded、
OpenOCD、GDB 与 `dfu-util`。两者都只包含开发工具，不包含 IDE、用户应用、目标机 runtime
或个人设备配置。

本目录不创建 Buildx Builder、不登录或部署 Registry，也不拥有 OCI push 和发布 handoff。
镜像生产由独立的[环境镜像构建服务](../../services/environment-image-builder/README.md)完成，
OCI 存储仍是另一项服务。构建和检查环境定义见[维护指南](MAINTAINING.md)。
