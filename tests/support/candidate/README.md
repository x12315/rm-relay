# 候选体验环境

本模块供维护者在正式 CLI Release、OCI image 和独立 Project Template 尚未同时发布时，组合一套
接近普通用户入口的候选环境。它只制备当前 CLI、隔离配置、模板 Git origin 和空 workspace；
Builder 与 environment image 必须已经存在，并由 Candidate 借用。

这不是 staging，也不是基础设施部署器。Candidate 不构建或替换 image，不创建、删除 Buildx
resource，也不管理 Registry 和 build cache。

## 外置目录

Go `os.UserCacheDir()` 决定平台惯用的 cache base：

| 系统 | base |
| --- | --- |
| macOS | `~/Library/Caches/` |
| Linux | `${XDG_CACHE_HOME:-~/.cache}/` |
| Windows | `%LocalAppData%\` |

每个 repository clone 或 worktree 对应一个独立目录：

```text
<user-cache>/rm-relay/experience/<repository-key>/
├── state.json
├── bin/
├── config/
├── template.git/
├── workspace/
└── logs/
```

这些内容不进入源码仓库，也不保存不可替代资产。系统不会可靠地自动回收它们，维护者需要在
核验结束后显式执行 `experience:clean`。

## 准备已有资源

开始前需要：

1. clean 的 RM Relay worktree；
2. 普通开发机 catalog 中已经可解析的逻辑 Builder，默认是 `local`；
3. 该 Builder 能拉取的 `image@sha256:<digest>`。

正式或战队 Registry 已经提供 image 时直接使用。尚无可用 Registry、需要在 Linux 主机完成一次
候选验收时，先按[临时环境来源](../../../docs/operator-guide/prepare-temporary-environment-source.md)
取得 handoff 中的 `immutable_reference`。

## 制备

设置不可变 reference；选择远程 Builder 时再设置 Builder ID：

```bash
export RM_RELAY_CANDIDATE_ENVIRONMENT='registry.example.org/rm-relay/embedded-development@sha256:<64位小写十六进制摘要>'
export RM_RELAY_CANDIDATE_BUILDER=local

mise run experience:prepare
```

`RM_RELAY_CANDIDATE_ENVIRONMENT_ID` 默认是 `embedded-development`，只有核验其他环境定义时才需
覆盖。`prepare` 会：

- 在仓库外构建当前宿主平台的候选 CLI；
- 从已提交的 Project Template 文件创建本地 Git origin；
- 将选中的 Builder 及给定 environment mapping 复制到隔离 catalog；
- 记录 repository、CLI、Builder、environment 与 template identity；
- 创建空 workspace。

它不会验证 image 可拉取；该行为留给候选 CLI 的 `environment check`，从而真实覆盖普通用户
链路。源码 dirty 或同一 worktree 的 Candidate 已存在时，`prepare` 会拒绝覆盖。

## 进入

```bash
mise run experience:enter
```

进入前会重新核对 repository revision、CLI digest/version、Builder ID/kind/Buildx name、
environment digest 和 template revision。校验通过后，Candidate 在外置 `workspace/` 打开宿主
shell，将候选 CLI 放到 `PATH` 首位，并设置：

```text
RM_RELAY_CONFIG_DIR    隔离的 Builder catalog
RM_RELAY_TEMPLATE_URL  本地候选模板 origin
```

Candidate 不自动 clone、build 或运行用户程序。人工判断步骤见
[开发者人工核验](../../manual/README.md)。

## 清理

退出 Candidate shell 后，从源码仓库运行：

```bash
mise run experience:clean
```

清理先校验 state schema、管理标记和 repository identity，再删除当前 Candidate 的外置目录。
它不会删除借用的 Buildx resource、image、Registry 或 cache。使用临时环境来源时，Candidate
清理完成后再返回对应运维 how-to，单独回收那些明确创建的资源。

状态损坏、路径不是普通目录或 ownership 不匹配时，命令停止，不扩大删除范围。状态 schema v3
不再接受 `image_id`、`image_reference` 或 `previous_image_id` 等旧 ownership 字段。
