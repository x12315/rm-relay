# macOS：宿主 OpenOCD + 容器 GDB

通用的工具职责、状态和验收条件见
[OpenOCD/GDB 烧录与调试后端](backends/openocd-gdb.md)。本页只说明 macOS 的宿主边界。

## 边界

Docker Desktop 不直接透传 ST-Link。OpenOCD 在 macOS 宿主连接 USB，
`gdb-multiarch` 在 `mcu-dev` 容器通过 `host.docker.internal:3333` 连接。

## 接线与预检

给 RoboMaster C 正常供电，连接独立 ST-Link 的 SWDIO、SWCLK、GND 和目标参考电压。
不要用 ST-Link 的供电能力替代开发板规定的正常供电方式。

```bash
openocd --version
system_profiler SPUSBDataType | grep -Ei -A 8 'ST-LINK|STMicroelectronics'
test -s examples/deterministic-pi-control/build/stm32f407-robomaster-c/firmware/robomaster-c-pi-control-example.elf
```

若系统信息中没有 ST-Link，先检查数据线、USB 转接器和调试器本身，不要反复启动
OpenOCD。

## 启动宿主 OpenOCD

在仓库根目录运行：

```bash
openocd -f toolkit/openocd/boards/robomaster-c.cfg -c "bindto 0.0.0.0"
```

`0.0.0.0` 仅用于 Docker VM 到宿主的桥接。调试结束立即按 Ctrl-C 停止服务，
不要在不可信网络中长期暴露 3333、4444 或 6666 端口。

## 容器内烧录并停在观察点

另开终端，在仓库根目录运行：

```bash
docker run --rm -it -v "$PWD:/workspace" -w /workspace \
  mcu-dev/toolchain:local gdb-multiarch \
  examples/deterministic-pi-control/build/stm32f407-robomaster-c/firmware/robomaster-c-pi-control-example.elf \
  -ex "target extended-remote host.docker.internal:3333" \
  -ex "monitor reset halt" \
  -ex "load" \
  -ex "break pi_control_example_observation_ready" \
  -ex "continue" \
  -ex "print g_pi_control_observed_command" \
  -ex "print g_pi_control_observed_fault"
```

验收条件：

- `load` 没有写入或校验错误；
- 断点命中 `pi_control_example_observation_ready`；
- `g_pi_control_observed_command` 位于 `0.45149` 到 `0.45151`；
- `g_pi_control_observed_fault` 为 `0`。

四项全部满足后，才可记录为 `debug-tested`。
