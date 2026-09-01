# 候选体验环境

候选体验环境供维护者在正式 CLI Release、OCI image 和独立 Project Template 尚未发布时检查完整
用户入口。它把测试用品放在仓库外，核验者进入后仍按普通用户方式 clone、初始化和构建项目。

这不是远程 staging。staging 是将来部署在独立服务器上的预生产服务实例，不由本地体验环境
模拟。

## 存储位置

RM Relay 通过 Go `os.UserCacheDir()` 选择平台惯用目录：

| 系统 | `os.UserCacheDir()` base |
| --- | --- |
| macOS | `~/Library/Caches/` |
| Linux | `${XDG_CACHE_HOME:-~/.cache}/` |
| Windows | `%LocalAppData%\` |

每个 repository clone 或 worktree 对应：

```text
<user-cache>/rm-relay/experience/<repository-key>/
├── state.json
├── bin/
├── config/
├── template.git/
├── workspace/
└── logs/
```

这里的内容都可以重新生成。系统不会可靠地自动清理它们，当前使用显式 `experience:clean`。
`ccache` 和 Docker/BuildKit cache 属于 build backend，不在该目录中，也不随体验环境删除。

## 制备

候选环境需要一个可由 BuildKit 拉取的 development image digest。当前 Registry 产品和正式
发布入口尚未确定，因此先由维护者显式提供：

```bash
export RM_RELAY_CANDIDATE_ENVIRONMENT='registry.example.org/rm-relay/embedded-development@sha256:<64位十六进制摘要>'
mise run experience:prepare
```

该任务完成以下工作：

- 在仓库外构建当前宿主平台的候选 CLI；
- 从已提交的 Project Template 文件创建本地 Git origin；
- 构建并加载 `mcu-dev/toolchain:local`；
- 在隔离的 Builder catalog 中把 `embedded-development` 映射到给定 digest；
- 记录 Git revision、CLI digest、image ID 和 template revision；
- 创建空的模拟用户 workspace。

源码仓库 dirty 或已有同仓库体验环境时，任务会拒绝覆盖。它不会创建临时 branch，也不会调用
`git subtree split`。

## 进入

```bash
mise run experience:enter
```

进入前会重新核对 repository revision、CLI digest/version、environment digest、development
image ID 与 template revision。任一身份变化都会中止，避免核验者测试到混合版本。

校验通过后，RM Relay 在外置 `workspace/` 打开宿主 shell，将候选 CLI 临时放到 `PATH` 首位，通过
`RM_RELAY_TEMPLATE_URL` 提供本地 Git origin，并把候选 CLI 的用户配置隔离到 `config/`。它不会
自动 clone、build 或运行用户程序。
人工步骤见[本地 MCU 开发体验](../../tests/manual/user-experience/local-mcu-development.md)。

## 清理

退出候选 shell 后执行：

```bash
mise run experience:clean
```

清理命令先校验 `state.json` 的 schema、管理标记和 repository identity，再逐项删除隔离 catalog
中登记的远程逻辑 Builder 及其宿主 Docker Buildx 实例，最后恢复 `experience:prepare` 前的
`mcu-dev/toolchain:local` tag；原来没有该 tag 时，移除候选 tag。全部 Docker 操作成功后才删除
对应 `experience/<repository-key>/`。

内建 `local` Builder 的映射随隔离目录删除；`rm-relay-local` Buildx resource 及其 cache 可以
继续供后续开发使用，不属于候选目录的清理对象。

状态文件损坏、路径是 symlink、Buildx 资源无法删除或 image tag 无法恢复时，命令会停止并保留
目录，避免扩大删除范围。
它不会删除其他仓库的体验环境、共享 build cache 或正式分发制品。
