# 使用指南

本目录说明当前 STM32 开发基线已经实现的功能。它假设 development image、CLI 和项目已经
可用；三者面向普通用户的稳定获取入口尚未交付。候选版本的资产准备与人工验收属于
[开发者人工核验](../../tests/manual/README.md)，不包装成本手册的首次使用流程。仓库尚未
交付的能力不在这里预演，实际支持程度以[支持矩阵](support-matrix.md)为准。

## 功能入口

1. 先阅读[选择本地或远程 Builder](builders.md)，准备本机 Docker 或登记战队 BuildKit 服务。
2. 已获得 development image 后，阅读[镜像选择与运行](image-selection.md)，确认能力与宿主
   边界并进入 `mcu-dev` 镜像。
3. 用[native 构建与测试](build-native.md)确认 host 路径，或直接按
   [STM32F407 固件构建](build-stm32.md)生成 RoboMaster C 固件。
4. 需要连接实板时，先查看[RoboMaster C 板卡说明](boards/robomaster-c.md)，再选择烧录或
   调试 backend。

## 按任务查找

| 目标 | 阅读入口 |
|---|---|
| 选择本地 Docker 或登记远程 BuildKit | [Builder 配置](builders.md) |
| 选择镜像、运行容器和处理宿主设备边界 | [镜像选择与运行](image-selection.md) |
| 运行 native Clang、GCC 和 sanitizer 测试 | [native 构建与测试](build-native.md) |
| 生成并检查 STM32F407 ELF/BIN/MAP | [STM32F407 固件构建](build-stm32.md) |
| 使用 ROM DFU 烧录 | [dfu-util backend](backends/dfu-util.md) |
| 使用 ST-Link、OpenOCD 和 GDB | [OpenOCD/GDB backend](backends/openocd-gdb.md) |
| 按宿主系统接入 OpenOCD/GDB | [macOS 流程](flash-debug-macos.md) · [Linux 流程](flash-debug-linux.md) |
| 查询首个支持板卡的容量、接线和证据 | [RoboMaster C 板卡说明](boards/robomaster-c.md) |
| 接入 VS Code/Cortex-Debug | [VS Code 示例](vscode-example.md) |
| 按错误症状定位问题 | [故障排查](troubleshooting.md) |
| 判断某个平台或 backend 是否经过真实验证 | [支持矩阵](support-matrix.md) |

镜像维护和发布不属于用户操作，见[镜像构建与验证](../operator-guide/build-and-verify-images.md)。
架构角色、目标能力和未来设计见[开发平台架构](../architecture/README.md)。
