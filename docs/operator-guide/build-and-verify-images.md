# 镜像构建与验证

本页供镜像维护者和构建服务部署者使用。功能使用方式从
[使用指南](../user-guide/README.md)进入，不必了解镜像内部的安装过程。

## 唯一构建入口

从仓库根目录调用同一份 Bake 文件：

```bash
docker buildx bake \
  --file environments/embedded-development/docker-bake.hcl \
  mcu-dev-arm64 --load
```

在 x86_64 Linux 主机上将 target 改为 `mcu-dev-amd64`。`--load` 只支持将单一平台
结果加载到本地 Docker image store。

只验证构建而不加载镜像时：

```bash
docker buildx bake \
  --file environments/embedded-development/docker-bake.hcl \
  verify-arm64 --set '*.output=type=cacheonly'

docker buildx bake \
  --file environments/embedded-development/docker-bake.hcl \
  verify-amd64 --set '*.output=type=cacheonly'
```

配置了支持 multi-platform 的 `docker-container` 或远程 builder 后：

```bash
docker buildx bake \
  --file environments/embedded-development/docker-bake.hcl \
  multiarch --set '*.output=type=cacheonly'
```

本页的 Buildx builder 属于 environment image 生产，不是 `rm-relay build` 使用的 workspace
Builder。仓库不在这里自动创建、切换或删除持久 image-production Builder。

## 推送 OCI image

发布前由运维准备一个支持 `linux/amd64`、`linux/arm64` 的 Buildx Builder，并用 Docker
credential store 登录目标 Registry。RM Relay 不安装 Docker、不创建 Registry，也不接收登录
凭据。

三个输入都必须显式提供：

```bash
export RM_RELAY_IMAGE_BUILDER=image-factory
export RM_RELAY_ENVIRONMENT_TAG=registry.example.org/rm-relay/embedded-development:v0.1.0
export RM_RELAY_ENVIRONMENT_HANDOFF=/absolute/path/outside/repository/embedded-development-v0.1.0.toml

mise run environment:embedded:publish
```

handoff 路径必须位于仓库外、父目录已经存在且目标文件尚不存在。任务依次执行 Bake 静态检查、
`mcu-dev-multiarch` 构建、Dockerfile 内 smoke、Registry push 和远端 manifest 检查；最后确认
`linux/amd64` 与 `linux/arm64` 均存在，再原子写入：

```toml
schema_version = 1
environment_id = "embedded-development"
tag = "registry.example.org/rm-relay/embedded-development:v0.1.0"
digest = "sha256:<64位小写十六进制摘要>"
immutable_reference = "registry.example.org/rm-relay/embedded-development@sha256:<64位小写十六进制摘要>"
source_revision = "<Git commit>"
platforms = ["linux/amd64", "linux/arm64"]
```

普通开发者只消费 `immutable_reference`。官方 GitHub 自动化、战队自建 CI 或人工发布都调用
这项任务；触发条件、Builder 创建、cache、Registry 地址和凭据由各自部署注入，不复制
Dockerfile、Bake target 或 smoke。任务只接受 clean Git revision，并把 commit 写入 handoff；
Registry 与 CI 产品仍未选定。

## LTS 基线与软件源

版本基线见 [镜像版本基线](../../environments/embedded-development/locks/README.md)：

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
  --file environments/embedded-development/docker-bake.hcl \
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

这两项检查会打印版本，核对 Catch2 的固定 CMake package，并让 host GCC、host Clang 与
`arm-none-eabi-g++` 实际编译同一份 C++20 contract probe。镜像标签可用以下命令审查：

```bash
docker inspect mcu-dev/toolchain:local --format \
  'arch={{.Architecture}} ubuntu={{index .Config.Labels "dev.embedded.ubuntu.lts"}} gcc={{index .Config.Labels "dev.embedded.native-gcc.major"}} capabilities={{index .Config.Labels "dev.embedded.capabilities"}}'
```

## 消费者契约回归

镜像 smoke 通过后，还要验证仓库契约、C++ 消费者和公开 CLI 链路。先安装仓库固定的
维护工具并运行快速检查：

```bash
mise trust
mise install
mise run verify
mise run test:unit
mise run test:architecture
mise run test:integration
```

`verify` 只检查静态仓库契约，不启动 Docker。项目模板和 PI 示例各自用 CMake Workflow
作为构建事实源。候选回归必须使用干净的派生目录；先确认两处 `build/` 均被 Git 忽略且没有
待提交资产，再清除可能记录旧工作区路径的 CMake cache：

```bash
git check-ignore project-templates/cross-platform-cpp/build \
  examples/deterministic-pi-control/build
git status --short -- project-templates/cross-platform-cpp/build \
  examples/deterministic-pi-control/build
```

`git check-ignore` 必须列出两处目录，`git status` 必须没有输出；否则停止，不删除来源不明的
内容。确认后再单独清理：

```bash
cmake -E remove_directory project-templates/cross-platform-cpp/build
cmake -E remove_directory examples/deterministic-pi-control/build
```

从仓库根目录运行候选 development image，先验证 Project Template。`bash -e` 保证任一 workflow
失败时容器立即返回非零状态；该命令全部通过后才能验证 PI 示例。

```bash
docker run --rm -t \
  -v "$PWD:/workspace" \
  -w /workspace/project-templates/cross-platform-cpp \
  mcu-dev/toolchain:local bash -euc '
    cmake --workflow --preset native-clang
    cmake --workflow --preset native-gcc
    cmake --workflow --preset native-asan
    cmake --workflow --preset stm32f407-robomaster-c
  '
```

再验证完整 PI 示例：

```bash
docker run --rm -t \
  -v "$PWD:/workspace" \
  -w /workspace/examples/deterministic-pi-control \
  mcu-dev/toolchain:local bash -euc '
    cmake --workflow --preset native-clang
    cmake --workflow --preset native-gcc
    cmake --workflow --preset native-asan
    cmake --workflow --preset stm32f407-robomaster-c
  '
```

这些命令应在候选 development image 内执行。前三个 workflow 由 CTest 发现并运行 Catch2
测试，最后一个生成 F407 固件，但不访问硬件。

最后验证 CLI archive。完整本地链路还需要一个已推送的 immutable environment：

```bash
mise run test:distribution
export RM_RELAY_E2E_LOCAL_ENVIRONMENT='registry.example.org/rm-relay/embedded-development@sha256:<64位小写十六进制摘要>'
mise run test:e2e
```

`test:distribution` 检查 Darwin、Linux、Windows 的 amd64/arm64 archive 和 SHA-256；它不在
当前主机运行其他平台的二进制。`test:e2e` 解压当前平台 archive，真实执行 Git clone、
`rm-relay init`、受管 Buildx 构建、Build Output 校验和 OpenOCD dry-run。两项任务都在临时目录生成
并消费自己的候选制品，不向仓库写入 `dist/`，结束后也不保留可供人工核验的环境。缺少 Git、
Docker 或可用的 Docker daemon 时，实际执行会失败；未设置 environment digest 时，本地 E2E
明确跳过。dry-run 只解析 adapter 配置并生成宿主 mise/OpenOCD 命令，不执行 mise 或 OpenOCD。

这组结果最多证明 `host-tested`、`cross-compiled` 和 OpenOCD `configured`。真实烧录、启动
和源码调试仍以[支持矩阵](../user-guide/support-matrix.md)的硬件证据为准。

## 开发者人工核验

自动 E2E 通过后，先按[候选体验环境](candidate-experience-environment.md)制备仓库外的候选 CLI、
development image 和 Project Template origin，再从[人工测试索引](../../tests/manual/README.md)
选择与宿主和链路匹配的场景。核验者只逐条输入普通用户会执行的公开命令，判断文档顺序、可见
输出和理解成本；确定性的 archive、checksum、manifest 与错误码继续由自动测试负责。

人工结果同样遵守证据等级：没有连接硬件的场景只能记录 `cross-compiled` 与 adapter
`configured`，不能升级为 `detected`、`flashed`、`boot-observed` 或 `debug-tested`。

## CLI 本地候选制品

GoReleaser 是 CLI 支持矩阵、版本注入、archive 命名和 checksum 的事实源：

```bash
export RM_RELAY_CLI_OUTPUT_DIR=/absolute/path/to/rm-relay-cli
mise run distribution:cli:build

export RM_RELAY_CLI_OUTPUT_DIR=/absolute/path/to/rm-relay-cli-snapshot
mise run distribution:cli:snapshot
```

输出目录必须是仓库外尚不存在的绝对路径；两项任务只接受 clean revision。完整约束和 Windows
示例见 [CLI 本地分发制品](cli-distribution.md)。当前命令不发布 GitHub Release。CLI archive
只包含 `rm-relay[.exe]` 与 `LICENSE`；mise、development image 和 Project Template 分别通过
自身渠道交付。

## 自动化扩展点

本仓库已提供可独立调用的 environment publish task，以及单节点 mTLS workspace builder 部署；
尚未实现官方 CI workflow、OCI Registry 部署和公共云服务。Workspace builder 部署入口见
[部署 mTLS BuildKit 服务](deploy-buildkit-service.md)。后续 adapter 应保持以下边界：

- 构建服务负责选择 builder、注入 registry tag、缓存和凭据；
- 工具版本、Dockerfile、Bake target 和 smoke contract 继续在本目录定义；
- 用户模板和示例作为镜像消费者，不被复制进镜像；
- CI 配置可以独立成子库，但只能注入现有 publish task 的输入。

当前 integration test 使用 fake Docker CLI 核对参数、命令顺序、manifest 判断和仓库外 handoff；
没有 Registry 写入凭据时，不能声称真实 push 已验证。
