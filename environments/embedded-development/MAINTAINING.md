# embedded-development 维护指南

本页供环境维护者修改、构建和检查当前嵌入式开发环境。普通开发者从
[使用指南](../../docs/user-guide/README.md)进入，不需要了解镜像的安装过程。

## 构建当前宿主架构

服务器或开发机需要已有 Docker 与 Buildx；本仓库不安装或修改 Docker。首次 checkout 先安装
仓库固定的维护工具，然后构建当前宿主对应的定义：

```bash
mise trust
mise install
mise run service:environment-image-builder:verify
```

`verify` 使用 `docker-bake.hcl` 中的 `verify-amd64` 或 `verify-arm64` group，不把结果写入宿主
image store。需要加载当前 `mcu-dev` 结果以便人工检查时运行：

```bash
mise run service:environment-image-builder:load
```

`--load` 只适用于单一平台。需要检查完整 multi-platform 定义时，使用已经支持
`linux/amd64` 与 `linux/arm64` 的 Buildx Builder：

```bash
docker buildx bake \
  --builder <image-production-builder> \
  --file environments/embedded-development/docker-bake.hcl \
  multiarch --set '*.output=type=cacheonly'
```

这里的 Builder 只用于环境生产，不是普通开发者执行 `rm-relay build` 的 workspace Builder。

## 版本与软件源

产品基线见[镜像版本基线](locks/README.md)：

- Ubuntu 固定在 24.04 LTS 系列，重新构建时接收同一 LTS 的安全更新；
- native 构建显式使用 GCC 14，STM32 构建使用 Arm GNU 13.2.Rel1；
- uv 与引导镜像使用固定版本或 OCI digest；
- 镜像内包清单记录每次实际解析结果。

默认 Ubuntu mirror 为 USTC。维护者可以通过 Bake override 整体替换来源，例如：

```bash
docker buildx bake \
  --file environments/embedded-development/docker-bake.hcl \
  mcu-dev-amd64 --load \
  --set 'mcu-dev-amd64.args.UBUNTU_MIRROR=http://mirrors.cloud.aliyuncs.com/ubuntu'
```

ARM64 构建改为覆盖 `UBUNTU_PORTS_MIRROR`。切换来源后应重新构建，不在一次 APT 求解中混用
多个同步时刻的镜像，也不关闭 Ubuntu 签名验证。实际包版本保存在：

```text
/opt/embedded-development/base-packages.txt
/opt/embedded-development/embedded-packages.txt
```

## 检查镜像能力

`service:environment-image-builder:load` 默认生成本地维护 tag `mcu-dev/toolchain:local`。检查其中
的工具链与 C++20 probe：

```bash
docker run --rm mcu-dev/toolchain:local sh -lc \
  '/usr/local/lib/embedded-development/smoke/verify-base-tools.sh &&
   /usr/local/lib/embedded-development/smoke/verify-embedded-tools.sh'

docker inspect mcu-dev/toolchain:local --format \
  'arch={{.Architecture}} ubuntu={{index .Config.Labels "dev.embedded.ubuntu.lts"}} gcc={{index .Config.Labels "dev.embedded.native-gcc.major"}} capabilities={{index .Config.Labels "dev.embedded.capabilities"}}'
```

再运行环境定义可能影响到的仓库契约：

```bash
mise run verify
mise run test:unit
mise run test:architecture
mise run test:integration
```

这些结果只能证明定义构建、镜像 smoke 与仓库契约成立。没有真实 OCI endpoint 和写入凭据时，
不能声称 Registry push 已验证；没有硬件时，也不能提升烧录或调试证据等级。

## 交给镜像生产

环境定义通过上述检查后，由[环境镜像构建服务](../../services/environment-image-builder/README.md)
负责 multi-platform push、manifest 核验和不可变 reference handoff。该服务消费本目录的文件，
但它的 Builder、Registry 凭据、cache 和发布记录不归本目录管理。
