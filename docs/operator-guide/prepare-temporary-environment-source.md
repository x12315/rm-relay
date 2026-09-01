# 准备临时环境来源

本页供维护者在没有正式 OCI Registry 时，为一次候选验收准备可由本机 BuildKit 拉取的
environment digest。它以 Linux Docker Engine 与 Buildx 为保证基线；macOS Docker Desktop
可能兼容，但 host network 与跨架构行为未纳入本路径的保证范围。

这是一条备用维护路径。Registry 只绑定 loopback、没有认证，所有资源都应在验收结束后移除；
不要将它改成局域网或公网服务。长期 Registry 需要 TLS、访问控制、持久存储和独立运维决议。

## 1. 确认资源名没有被占用

本流程将创建三个明确命名的临时资源：

```text
rm-relay-temporary-registry
rm-relay-environment-image-builder
rm-relay-local
```

先分别检查：

```bash
docker container inspect rm-relay-temporary-registry
docker buildx inspect rm-relay-environment-image-builder
docker buildx inspect rm-relay-local
```

新环境中三条命令都应报告对象不存在。任一对象已经存在时停止，不删除、不替换，也不尝试把
未知资源改造成当前流程需要的配置；先确认它的所有者和用途。

## 2. 启动 loopback Registry

使用固定到 OCI digest 的 CNCF Distribution 3.1.1：

```bash
docker run --detach \
  --name rm-relay-temporary-registry \
  --publish 127.0.0.1:5000:5000 \
  --env OTEL_TRACES_EXPORTER=none \
  registry:3.1.1@sha256:1be55279f18a2fe1a74edf2664cac61c1bea305b7b4642dab412e7affdcb3e33

curl --fail http://127.0.0.1:5000/v2/
```

空 JSON response 表示 Registry V2 endpoint 可达。镜像数据只保存在该临时 container 中，移除
container 后一并消失。

## 3. 创建两类 Buildx Builder

Registry 地址写作 `localhost:5000`。BuildKit 运行在 container 中，因此两个 Builder 都显式使用
Linux host network。两条命令的参数相同，但资源职责不同：

| Buildx resource | 使用者 | 职责 |
| --- | --- | --- |
| `rm-relay-environment-image-builder` | 环境镜像构建服务 | 从 Dockerfile 构建开发环境，并 push 到临时 Registry |
| `rm-relay-local` | RM Relay 的逻辑 Builder `local` | 从 Registry 拉取开发环境，在其中编译用户 workspace |

因此二者必须保持独立的 cache 和生命周期：前者属于镜像生产，后者属于日常项目构建。这里的
`rm-relay-local` 是 Docker Buildx resource 名称；普通用户选择的仍是逻辑 ID `local`。

```bash
docker buildx create \
  --name rm-relay-environment-image-builder \
  --driver docker-container \
  --driver-opt image=moby/buildkit:v0.32.2@sha256:28a898719c18a33f4e8000685287fa36fd0dd9560c6440227d3a732d79bb41d8 \
  --driver-opt network=host

docker buildx create \
  --name rm-relay-local \
  --driver docker-container \
  --driver-opt image=moby/buildkit:v0.32.2@sha256:28a898719c18a33f4e8000685287fa36fd0dd9560c6440227d3a732d79bb41d8 \
  --driver-opt network=host

docker buildx inspect --bootstrap rm-relay-environment-image-builder
docker buildx inspect --bootstrap rm-relay-local
```

第一个 Builder 的 Platforms 必须同时包含 `linux/amd64` 与 `linux/arm64`。Docker Engine 的官方
BuildKit 通常可以使用自带 QEMU；若平台缺失，先按
[Docker multi-platform 文档](https://docs.docker.com/build/building/multi-platform/)补齐主机能力，
不要把发布矩阵降为单架构。`rm-relay-local` 是 Candidate 后续借用的 workspace Builder。

## 4. 生成不可变 reference

仓库必须 clean。先选择仓库外目录保存一次性 handoff，并把 `<version>` 换成能对应当前候选
revision 的版本字符串：

```bash
export RM_RELAY_MAINTENANCE_ROOT=/absolute/path/outside/repository
mkdir -p "$RM_RELAY_MAINTENANCE_ROOT"

export RM_RELAY_IMAGE_BUILDER=rm-relay-environment-image-builder
export RM_RELAY_ENVIRONMENT_TAG=localhost:5000/rm-relay/embedded-development:<version>
export RM_RELAY_ENVIRONMENT_HANDOFF="$RM_RELAY_MAINTENANCE_ROOT/embedded-development-<version>.toml"

mise run service:environment-image-builder:publish
```

任务会 push `linux/amd64`、`linux/arm64`，核验远端 manifest，再写入 handoff。打开该文件并复制
`immutable_reference`；Candidate 的准备步骤只需要这个 reference 和已存在的 `local` Builder。

## 5. 验收结束后清理

先退出并清理 Candidate。确认没有其他构建仍在使用这些资源后，再按创建顺序的反向移除：

```bash
docker buildx rm rm-relay-local
docker buildx rm rm-relay-environment-image-builder
docker container rm --force rm-relay-temporary-registry
```

这会同时清除两个临时 Builder 的 cache 和 Registry 中的候选 image。handoff 文件由维护者在不再
需要追查此次候选版本后单独删除；Candidate 不替运维回收这些外部资源。

Registry 的本地运行方式来自
[CNCF Distribution 部署说明](https://distribution.github.io/distribution/about/deploying/)，Builder
的 host network 用法来自
[Docker Buildx 本地 Registry 示例](https://docs.docker.com/build/ci/github-actions/local-registry/)。
