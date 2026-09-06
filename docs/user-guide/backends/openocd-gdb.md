# OpenOCD/GDB 烧录与调试后端

OpenOCD 连接 ST-Link 等 debug adapter（调试适配器）并暴露 GDB server；GDB 加载
带符号的 ELF、控制 CPU、设置断点和读取变量。两者组合既可写入，也可执行源码级
调试，不能用“配置能解析”代替实板连接成功。

## 当前配置

RoboMaster C 当前使用 `internal/target/openocd/board/robomaster-c.cfg`：

- ST-Link interface；
- SWD transport；
- 1800 kHz adapter speed；
- STM32F4 target；
- 不假设板上可控 reset 线。

板卡接线和连接器信息见 [RoboMaster C 设备说明](../boards/robomaster-c.md)。

## 主机边界

- macOS：宿主 OpenOCD 访问 ST-Link，容器内 `gdb-multiarch` 连接
  `host.docker.internal:3333`。
- Linux：OpenOCD 与 GDB 可位于同一容器，前提是明确映射 USB 设备并解决权限。
- Windows：后续在真实主机验证宿主 OpenOCD + 容器/WSL GDB；当前为 `planned`。

平台命令分别见 [macOS 流程](../flash-debug-macos.md)和
[Linux 流程](../flash-debug-linux.md)。

## 配置解析

没有 ST-Link 时仍可检查配置文件能否被 OpenOCD 解析：

```bash
openocd -f internal/target/openocd/board/robomaster-c.cfg -c "init; shutdown"
```

允许在尝试打开 adapter 时失败；Tcl 语法错误、找不到 interface/target 配置或无法读取
项目配置不允许通过。解析成功只记为 `configured`。

## 实板调试验收

建立可靠的 SWD 连接后，OpenOCD 应识别 ST-Link 与 STM32F407。GDB 使用
PI 示例的 `robomaster-c-pi-control-example.elf` 执行 `load`，在
`pi_control_example_observation_ready` 设置断点并读取：

```text
g_pi_control_observed_command: 0.45149 .. 0.45151
g_pi_control_observed_fault: 0
```

状态必须分别记录：

- `load` 成功：`flashed`；
- OpenOCD/GDB 写入校验成功：`flash-verified`；
- 断点命中且两个变量满足范围：`debug-tested`。

若只看到 LED 或串口输出，可以另记 `boot-observed`，不能据此声称断点和变量观察链路
已经通过。

RoboMaster C 已在 macOS 使用 ST-Link V2、OpenOCD 0.12.0 和容器内 GDB 15.1 完成
该验收：目标电压稳定在 3.3 V，STM32F407 被识别，断点命中，观测命令为
`0.451500028`，故障标志为 `0`。

OpenOCD 的 adapter 与 GDB server 参数以
[OpenOCD 官方文档](https://openocd.org/doc/html/)为准。
