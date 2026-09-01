# 开发者人工核验

人工核验只判断自动测试难以可靠判断的用户体验：入口是否找得到，文档顺序能否完成任务，
命令和错误是否容易理解，输出是否说明下一步。Schema、archive、checksum、命令参数和模块契约
属于自动测试；发现缺口时补测试或 code review，不让核验者手工复算。

## 组合一套候选链路

人工场景使用真实模块，不再建立一套测试专用 mini 流程：

```text
已有 Builder + immutable environment reference
                         │
                         ▼
                 Candidate prepare
                         │
                         ▼
候选 CLI + 隔离配置 + template origin + 空 workspace
                         │
                         ▼
                 人工输入用户命令
```

按以下顺序执行：

1. 运行 `test:unit`、`test:architecture`、`test:integration`、`test:release` 和适用的 E2E；
2. 选择已经可用的 Builder 与 environment digest；没有正式来源时，先在 Linux 主机按
   [备用 how-to](../../docs/operator-guide/prepare-temporary-environment-source.md)取得 digest；
3. 按[Candidate 说明](../support/candidate/README.md)运行 `experience:prepare` 和
   `experience:enter`；
4. 在 Candidate shell 中选择下方场景，逐条输入普通用户命令并作认知判断；
5. 退出 shell，运行 `experience:clean`；使用临时来源时再由维护者回收其外部资源。

Candidate 制备本身不是人工测试结论。它不构建 image、不部署 Registry，也不拥有 Builder；
这些前置资源的正确性由各自模块验证。

## 当前场景

| 场景 | 当前已验证宿主 | backend / Profile | 最高证据 |
| --- | --- | --- | --- |
| [本地 MCU 开发体验](user-experience/local-mcu-development.md) | macOS arm64 | `local-buildkit` / `embedded-stm32f407-robomaster-c` | `cross-compiled`、OpenOCD `configured` |
| [远程 MCU 构建体验](user-experience/remote-mcu-development.md) | 尚未实机验证 | `remote-buildkit` / `embedded-stm32f407-robomaster-c` | 待记录 |

未连接硬件的场景不能产生 `detected`、`flashed`、`boot-observed` 或 `debug-tested` 证据。实板
状态以[支持矩阵](../../docs/user-guide/support-matrix.md)为准。
