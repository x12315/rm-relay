# 支持矩阵

本页记录仓库当前实际交付程度，不表示某工具理论上能运行的平台。构建与硬件后端属于两个
证据域，状态定义见[开发契约参考](../reference/development-contracts.md#证据等级)。平台列表示
完整用户路径实际运行的宿主系统；只在 Linux 容器中完成工具 smoke，不能证明 Linux 宿主
路径已经通过。

## 构建证据

| 项目/目标 | macOS | Linux | Windows |
|---|---|---|---|
| STM32F407 / RoboMaster C 固件 | `cross-compiled` | — | — |

`—` 表示仓库没有记录该宿主系统上的实际构建证据，不是验证状态。

## 硬件后端证据

| 设备/后端 | macOS | Linux | Windows |
|---|---|---|---|
| RoboMaster C + dfu-util | `flash-verified` | `configured` | — |
| RoboMaster C + OpenOCD/GDB | `debug-tested` | `configured` | — |

- macOS 已通过 DFU 完成用户 Flash 写入和按产物大小回读比对，并通过 ST-Link、OpenOCD
  与容器内 GDB 命中 PI 示例观察点、核对约定变量。
- Linux 的工具和命令已配置，并通过 Linux 容器 smoke，因此硬件 backend 只能记为
  `configured`；构建和 USB 直连仍需在真实 Linux 主机上分别验证。Windows 留在后续计划中。

## 尚未交付的路线

| 路线 | 分类 |
|---|---|
| Windows 上的构建、DFU 和 OpenOCD/GDB 路径 | `planned` |
| STM32 ROM UART | `planned` |
| pyOCD/CMSIS-DAP | `planned` |
| J-Link GDB Server | `optional` |
| STC ISP/stcgal | `planned` |

`planned` 表示已识别到未来需求，但还没有对应的完整用户路径和真实验证。`optional` 表示可
按需接入的兼容路线，不进入默认开源镜像依赖。两者都不是证据状态，不能解释成“已经支持”。

当前已建立的入口：

- [STM32F407 固件构建](build-stm32.md)
- [RoboMaster C 设备说明](boards/robomaster-c.md)
- [dfu-util 后端](backends/dfu-util.md)
- [OpenOCD/GDB 后端](backends/openocd-gdb.md)

新增板型或后端时，先增加真实配置与可执行验证，再更新本表。不要先创建空配置或
从其他操作系统的结果推导支持状态。
