# CLI 发布脚本

本目录只保存 RM Relay CLI 的发布配置与薄调用脚本。正式发布、维护者本地 snapshot 和未来 CI
调用同一份 GoReleaser 配置；生成结果必须写入显式指定的仓库外目录。

```bash
scripts/release/cli.sh build /absolute/output/path
scripts/release/cli.sh snapshot /absolute/output/path
```

普通用户与候选体验不依赖本目录。候选体验由 `tests/support/candidate` 直接构建当前平台 CLI，
避免测试支持代码拥有正式发布流程。
