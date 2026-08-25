# 支持矩阵

本表记录仓库当前实际交付程度，不表示某工具理论上能运行的平台。状态含义见
[STM32 固件构建](build-stm32.md#验证状态)。

| 设备/后端 | macOS | Linux | Windows |
|---|---|---|---|
| RoboMaster C 构建 | `cross-compiled` | `configured` | `planned` |
| RoboMaster C + dfu-util | `flash-verified` | `configured` | `planned` |
| RoboMaster C + OpenOCD/GDB | `debug-tested` | `configured` | `planned` |
| STM32 ROM UART | `planned` | `planned` | `planned` |
| pyOCD/CMSIS-DAP | `planned` | `planned` | `planned` |
| J-Link GDB Server | `optional` | `optional` | `optional` |
| STC ISP/stcgal | `planned` | `planned` | `planned` |

- `planned` 表示已识别到未来需求，但仓库还没有对应配置和真实验证。
- `optional` 表示可按需接入的兼容路线，不进入默认开源镜像依赖。
- 两者都不是构建或硬件验证状态，不能解释成“已经支持”。
- macOS 已通过 DFU 完成用户 Flash 写入和按产物大小回读比对，并通过 ST-Link、OpenOCD
  与容器内 GDB 命中 PI 示例观察点、核对约定变量。
- Linux 的工具和命令已配置，并通过 Linux 容器 smoke；USB 直连仍需真实 Linux
  主机验证。Windows 留在后续计划中。

当前已建立的入口：

- [RoboMaster C 设备说明](boards/robomaster-c.md)
- [dfu-util 后端](backends/dfu-util.md)
- [OpenOCD/GDB 后端](backends/openocd-gdb.md)

新增板型或后端时，先增加真实配置与可执行验证，再更新本表。不要先创建空配置或
从其他操作系统的结果推导支持状态。
