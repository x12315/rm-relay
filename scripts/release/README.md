# CLI 发布脚本

本目录只保存 RM Relay CLI 的发布配置与薄调用脚本。正式发布、维护者本地 snapshot 和未来 CI
调用同一份 GoReleaser 配置；生成结果必须写入显式指定的仓库外目录。
它不负责 GitHub Release、长期制品存储或上传渠道。

通过 mise 运行时，先设置绝对输出路径：

```bash
export RM_RELAY_CLI_OUTPUT_DIR=/absolute/path/outside/repository/rm-relay-cli
mise run release:build
mise run release:snapshot
```

`release:build` 生成未归档的跨平台 binary；`release:snapshot` 生成 archive 和 checksum。
输出路径必须位于仓库外且事先不存在。两项任务只处理当前已提交 revision；工作区存在未提交
修改时会拒绝执行，避免得到无法追溯的制品。也可直接调用薄脚本：

```bash
scripts/release/cli.sh build /absolute/output/path
scripts/release/cli.sh snapshot /absolute/output/path
```

归档结构、LICENSE 与 checksum 契约由 `mise run test:release` 验证。`mise run test:e2e` 还会
解压当前平台 archive，并用其中的 CLI 驱动完整自动开发链路。两项测试都使用进程临时目录，
不会留下可供人工复用的制品。

普通用户与候选体验不依赖本目录。[候选体验环境](../../tests/support/candidate/README.md)直接构建
当前平台 CLI，避免测试支持代码拥有正式发布流程。
