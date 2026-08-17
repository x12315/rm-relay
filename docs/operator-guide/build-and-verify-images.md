# 镜像构建与验证

本页供镜像维护者和构建服务部署者使用。普通开发者从根目录的
[本地 Docker 快速开始](../../README.md#本地-docker-快速开始)进入即可，不必了解镜像
内部的安装过程。

## 唯一构建入口

从仓库根目录调用同一份 Bake 文件：

```bash
docker buildx bake \
  --file container-images/embedded-development/docker-bake.hcl \
  mcu-dev-arm64 --load
```

在 x86_64 Linux 主机上将 target 改为 `mcu-dev-amd64`。`--load` 只支持将单一平台
结果加载到本地 Docker image store。

只验证构建而不加载镜像时：

```bash
docker buildx bake \
  --file container-images/embedded-development/docker-bake.hcl \
  verify-arm64 --set '*.output=type=cacheonly'

docker buildx bake \
  --file container-images/embedded-development/docker-bake.hcl \
  verify-amd64 --set '*.output=type=cacheonly'
```

配置了支持 multi-platform 的 `docker-container` 或远程 builder 后：

```bash
docker buildx bake \
  --file container-images/embedded-development/docker-bake.hcl \
  multiarch --set '*.output=type=cacheonly'
```

仓库不自动创建、切换或删除持久 Buildx builder。未来 CI、registry 发布和云编译应
继续调用这些 target，通过 Bake override 设置 tag、cache 和 output；不要复制
Dockerfile 或维护第二份包清单。

## LTS 基线与软件源

版本基线见 [镜像版本基线](../../container-images/embedded-development/locks/README.md)：

- Ubuntu 固定在 24.04 LTS 系列，重新构建时接收同一 LTS 的安全更新；
- native 构建显式使用 GCC 14，STM32 构建使用 Arm GNU 13.2.Rel1；
- uv 与 CA 引导镜像继续使用不可变 digest；
- 发布环境由版本 tag、OCI digest 和镜像内包清单追踪。

默认构建使用 USTC：

```text
UBUNTU_MIRROR=https://mirrors.ustc.edu.cn/ubuntu
UBUNTU_PORTS_MIRROR=https://mirrors.ustc.edu.cn/ubuntu-ports
```

维护者可用 Bake override 切换一整组来源。例如在阿里云 ECS 中：

```bash
docker buildx bake \
  --file container-images/embedded-development/docker-bake.hcl \
  mcu-dev-amd64 --load \
  --set 'mcu-dev-amd64.args.UBUNTU_MIRROR=http://mirrors.cloud.aliyuncs.com/ubuntu'
```

ARM64 构建覆盖 `UBUNTU_PORTS_MIRROR`。切换来源后应重新构建，不在同一次 APT 求解中
混用多个同步时刻的镜像。APT 重试只处理短暂超时，不关闭 Ubuntu 签名验证。

镜像内以下文件保存本次实际解析结果：

```text
/opt/embedded-development/base-packages.txt
/opt/embedded-development/embedded-packages.txt
```

## 镜像能力检查

```bash
docker run --rm mcu-dev/toolchain:local sh -lc \
  '/usr/local/lib/embedded-development/smoke/verify-base-tools.sh &&
   /usr/local/lib/embedded-development/smoke/verify-embedded-tools.sh'
```

这两项检查会打印版本，并让 host GCC、host Clang 与 `arm-none-eabi-g++` 实际编译
同一份 C++20 contract probe。镜像标签可用以下命令审查：

```bash
docker inspect mcu-dev/toolchain:local --format \
  'arch={{.Architecture}} ubuntu={{index .Config.Labels "dev.embedded.ubuntu.lts"}} gcc={{index .Config.Labels "dev.embedded.native-gcc.major"}} capabilities={{index .Config.Labels "dev.embedded.capabilities"}}'
```

## 消费者契约回归

镜像 smoke 通过后，还要从仓库根目录验证真实消费者：

```bash
sh validation/project-contracts/verify-repository-layout.sh
sh validation/project-contracts/verify-toolchain-source-policy.sh
sh validation/project-contracts/verify-project-builds.sh
```

`verify-project-builds.sh` 会在项目模板和 PI 示例中分别运行 Clang、GCC、ASan/UBSan
与 F407 交叉编译 workflow。只有镜像能力与消费者构建都通过，才能认为镜像变更可交付。

## 发布与云构建扩展点

本仓库当前不实现自动发布和云编译服务。未来扩展应保持以下边界：

- 构建服务负责选择 builder、注入 registry tag、缓存和凭据；
- 工具版本、Dockerfile、Bake target 和 smoke contract 继续在本目录定义；
- 用户模板和示例作为镜像消费者，不被复制进镜像；
- CI 配置可以独立成子库，但不得重新定义镜像内容。
