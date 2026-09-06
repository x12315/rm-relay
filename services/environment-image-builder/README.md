# 环境镜像构建服务

本模块把一份经过检查的环境定义发布为 multi-platform OCI image，并生成交给消费者的不可变
reference。它面向镜像生产者和 CI 维护者，不处理普通用户每天修改的 workspace。

```text
环境定义 + image-production Builder + version tag
                       │
                       ▼
               build / smoke / push
                       │
                       ▼
          OCI Registry + 仓库外 handoff
```

## 输入边界

运行者需要提前准备：

- clean Git worktree 形式的环境源码；
- 一个能构建 `linux/amd64` 与 `linux/arm64` 的 Buildx Builder；
- 已完成登录、允许该 Builder push 的 OCI Registry；
- 带显式版本的 OCI tag；
- 位于环境源码外、父目录已存在且目标文件尚不存在的 handoff 路径。

本模块不创建或删除 Builder，不登录或部署 Registry，也不管理两者的凭据和 cache。环境源码
可以位于当前 monorepo，也可以是未来的独立环境仓库；Bake 和 identity 路径始终相对于该源码
根目录解析，不能越出根目录。

## 发布当前嵌入式环境

从仓库根目录显式提供生产资源：

```bash
export RM_RELAY_IMAGE_BUILDER=image-factory
export RM_RELAY_ENVIRONMENT_TAG=registry.example.org/rm-relay/embedded-development:v0.1.0
export RM_RELAY_ENVIRONMENT_HANDOFF=/absolute/path/outside/source/embedded-development-v0.1.0.toml

mise run service:environment-image-builder:publish
```

默认输入为当前仓库和 `environments/embedded-development/` 下的 Bake、identity 文件。消费独立
环境源码时，另外设置：

```bash
export RM_RELAY_ENVIRONMENT_SOURCE=/absolute/path/to/environment-source
export RM_RELAY_ENVIRONMENT_BAKE_FILE=docker-bake.hcl
export RM_RELAY_ENVIRONMENT_IDENTITY_FILE=identity.toml
```

任务先执行 Bake `--check`，再运行 `publish` group，启用 provenance 与 SBOM 并 push。完成后，
它从 Buildx metadata 取得顶层 digest，并从远端 manifest 确认 `linux/amd64`、`linux/arm64`
都存在。任何一步失败都不会发布 handoff。

## Handoff 契约

成功后原子创建 schema v1 TOML：

```toml
schema_version = 1
environment_id = "embedded-development"
tag = "registry.example.org/rm-relay/embedded-development:v0.1.0"
digest = "sha256:<64位小写十六进制摘要>"
immutable_reference = "registry.example.org/rm-relay/embedded-development@sha256:<64位小写十六进制摘要>"
source_revision = "<Git commit>"
platforms = ["linux/amd64", "linux/arm64"]
```

普通 Builder 和 Candidate 只消费 `immutable_reference`；version tag 只用于生产时命名。handoff
不是 Registry 凭据，也不应写回环境源码仓库。

## 当前证据

`publish_test.go` 用隔离的 fake Docker/Git 检查参数、路径边界、metadata digest、平台判断和
handoff 内容。真实 Registry push 尚未执行，因此当前只能证明发布控制流与契约已实现，不能
证明某个 Registry 或生产 Builder 已可用。

没有正式 Registry、只需在 Linux 主机完成候选验收时，可以使用
[临时环境来源](../../docs/operator-guide/prepare-temporary-environment-source.md)。那是一条明确的
备用路径，不是本服务部署出的长期 Registry。
