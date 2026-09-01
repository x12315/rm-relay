# 开发者人工核验

人工核验用于判断自动测试难以可靠判断的用户体验：文档顺序是否足以完成任务，命令和错误提示
是否易懂，输出能否让人判断下一步，以及证据等级是否表达准确。

archive 内容、checksum、manifest、错误码和模块契约属于自动测试。发现这些方面缺少覆盖时，应补
测试或进行 code review，不让核验者手工复算。

## 执行顺序

1. 先通过 `test:unit`、`test:architecture`、`test:integration`、`test:distribution` 和适用的 E2E；
2. 按[候选体验环境](../../docs/operator-guide/candidate-experience-environment.md)运行
   `experience:prepare` 与 `experience:enter`；
3. 在候选 shell 中选择下方场景，逐条输入普通用户命令；
4. 退出 shell 后运行 `experience:clean`。

候选环境制备不是人工测试结论。它只在仓库外提供候选 CLI、development image、Git template
origin 和空工作区。

## 当前场景

| 场景 | 当前已验证宿主 | backend / Profile | 最高证据 |
| --- | --- | --- | --- |
| [本地 MCU 开发体验](user-experience/local-mcu-development.md) | macOS arm64 | `local-buildkit` / `embedded-stm32f407-robomaster-c` | `cross-compiled`、OpenOCD `configured` |
| [远程 MCU 构建体验](user-experience/remote-mcu-development.md) | 尚未实机验证 | `remote-buildkit` / `embedded-stm32f407-robomaster-c` | 待记录 |

未连接硬件的场景不能产生 `detected`、`flashed`、`boot-observed` 或 `debug-tested` 证据。实板
状态以[支持矩阵](../../docs/user-guide/support-matrix.md)为准。
