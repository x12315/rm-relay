# embedded-development 镜像产品

这里维护 MCU 开发工具链镜像，供镜像维护者和构建服务部署者使用。普通开发者只需使用
构建好的 `mcu-dev/toolchain` 镜像，不需要修改 Dockerfile。

## 交付内容

- `Dockerfile`：`base` 与 `mcu-dev` 两个镜像 stage 的唯一构建定义。
- `docker-bake.hcl`：ARM64、AMD64 与 multi-platform 的标准 Buildx 入口。
- `identity.toml`：schema v1 固定声明 `id = "embedded-development"`，供 RM Relay CLI 核验。
- `locks/`：Ubuntu LTS、native GCC、Arm GNU 与 uv 的产品级版本基线。
- `smoke/`：镜像构建时和构建后都可运行的工具能力检查。
- `publish.sh`：官方自动化与战队自建共用的 build、verify、OCI push 入口。

`base` 提供 host C++20 构建、ccache 与质量工具，并内置 RM Relay 受控 CMake
Workflow；`mcu-dev` 在其上增加 GNU Arm Embedded、
OpenOCD、GDB 和 `dfu-util`。镜像只交付工具链，不包含用户应用、IDE、运行时服务或
个人设备配置。

发布入口不创建 Buildx Builder、不登录 Registry、不管理 cache，也不选择托管产品。调用者提供 image-production
Builder、带版本的 OCI tag 和仓库外 handoff 路径；发布要求 clean Git revision，成功后得到可交给
`rm-relay environment add` 的 immutable reference。该 Builder 与普通开发者执行
`rm-relay build` 使用的 workspace Builder 保持独立。

构建、验证和发布方式见
[镜像构建与验证](../../docs/operator-guide/build-and-verify-images.md)。
