# embedded-development 镜像产品

这里维护 MCU 开发工具链镜像，供镜像维护者和构建服务部署者使用。普通开发者只需使用
构建好的 `mcu-dev/toolchain` 镜像，不需要修改 Dockerfile。

## 交付内容

- `Dockerfile`：`base` 与 `mcu-dev` 两个镜像 stage 的唯一构建定义。
- `docker-bake.hcl`：ARM64、AMD64 与 multi-platform 的标准 Buildx 入口。
- `locks/`：Ubuntu LTS、native GCC、Arm GNU 与 uv 的产品级版本基线。
- `smoke/`：镜像构建时和构建后都可运行的工具能力检查。

`base` 提供 host C++20 构建、ccache 与质量工具，并内置 RM Relay 受控 CMake
Workflow；`mcu-dev` 在其上增加 GNU Arm Embedded、
OpenOCD、GDB 和 `dfu-util`。镜像只交付工具链，不包含用户应用、IDE、运行时服务或
个人设备配置。

构建、验证和未来发布方式见
[镜像构建与验证](../../docs/operator-guide/build-and-verify-images.md)。
