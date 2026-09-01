# CLI 本地分发制品

GoReleaser 继续定义 `rm-relay` 的平台矩阵、版本注入、archive 命名与 checksum。薄发布脚本在
仓库外的临时 clone 中运行 GoReleaser，再把结果原子写入指定目录；仓库根不生成 `dist/`。

## 输出目录

两个任务都要求 `RM_RELAY_CLI_OUTPUT_DIR`：

- 必须是绝对路径；
- 必须位于仓库外；
- 目标路径不能已经存在。

维护者在 macOS 或 Linux 上可以直接调用脚本：

```bash
scripts/release/cli.sh snapshot /absolute/path/to/rm-relay-cli
```

需要通过 mise 组合维护任务时，设置 `RM_RELAY_CLI_OUTPUT_DIR` 后运行同名 task。脚本和 task 都
不会自行选择长期制品目录；调用者负责确定保留位置和后续上传方式。

## 两种输出

```bash
mise run distribution:cli:build
mise run distribution:cli:snapshot
```

`distribution:cli:build` 生成未归档的跨平台 binary，适合检查 build matrix。`distribution:cli:snapshot` 生成
Darwin、Linux、Windows 的 amd64/arm64 archive 和 SHA-256 checksum，适合检查本地候选分发。

两者只使用当前已提交 revision；工作区有未提交改动时会拒绝运行，避免生成无法追溯的候选制品。
当前任务不发布 GitHub Release，也不生成包管理器 manifest。

## 自动测试

```bash
mise run test:distribution
mise run test:e2e
```

两项测试各自在进程临时目录中调用同一个 distribution command。`test:distribution` 检查所有 archive、
LICENSE 和 checksum；`test:e2e` 解压当前平台 CLI，再执行 Git clone、项目初始化、Docker 构建、
Build Output 校验与 OpenOCD dry-run。测试结束后不会留下可供人工复用的 archive；人工核验应使用
[候选体验环境](candidate-experience-environment.md)。
