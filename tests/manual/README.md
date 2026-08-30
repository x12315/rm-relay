# 开发者人工核验

本目录保存候选版本必须由开发者逐条执行的人工测试。它验证公开入口是否能被人按照文档完成，
以及输出、提示和证据边界是否符合预期；不是普通用户教程，也不代替自动测试。

## 与其他验证的边界

| 入口 | 负责内容 |
|---|---|
| package 旁的 `*_test.go` | 单个模块的确定行为 |
| `tests/architecture/` | 模块依赖方向 |
| `tests/integration/` | 不依赖真实外部进程的模块组合 |
| `tests/distribution/` | CLI archive 与 checksum 契约 |
| `tests/e2e/` | 分发 CLI、Git 与 Docker 的自动端到端链路 |
| `tests/manual/` | 开发者逐条操作、观察并记录的候选版本核验 |
| `scripts/verify/` | 仓库拓扑、版本和软件源等静态契约 |

人工核验产生什么证据，由场景自身限定。`configured` 或 `cross-compiled` 不能因为人工执行而
升级为 `detected`、`flashed`、`boot-observed` 或 `debug-tested`；实板状态仍以
[支持矩阵](../../docs/user-guide/support-matrix.md)为准。

## 场景契约

每个场景文件使用 `<workflow>-<host-os>-<host-arch>.md` 命名，并依次包含：

1. 目的与证据；
2. 适用组合；
3. 前置条件；
4. 准备候选产物；
5. 执行步骤；
6. 结果记录；
7. 清理；
8. 未覆盖。

执行步骤分别写明操作、预期结果和失败说明。命令必须由核验者逐条输入，不能用新脚本或自动
E2E 代替。场景应消费正式模块边界和公开 CLI，不调用 Go package 内部接口。

宿主命令、前置条件或证据范围出现真实分叉时，新增场景文件；不要在既有场景中不断叠加平台
条件。新增文件可以复用相同章节，但必须独立给出可执行命令和预期结果。

## 当前场景

| 场景 | 宿主 | backend / Profile | 最高证据 |
|---|---|---|---|
| [本地 MCU 开发链路](local-mcu-development-cycle-darwin-arm64.md) | macOS arm64 | `local-container` / `embedded-stm32f407-robomaster-c` | `cross-compiled`、OpenOCD `configured` |
