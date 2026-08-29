# Linux：容器直连 ST-Link

本流程已形成标准命令，但仍需在实际 Linux 主机和目标板上验证。
通用的工具职责、状态和验收条件见
[OpenOCD/GDB 烧录与调试后端](backends/openocd-gdb.md)。

## 找到单个设备节点

```bash
lsusb | grep -i 'ST-LINK\|STMicroelectronics'
```

从输出中的 Bus/Device 数字形成 `/dev/bus/usb/BBB/DDD`。不要把个人设备路径或
序列号写入仓库。

## 启动容器

```bash
docker run --name mcu-shell --rm -dit \
  --device /dev/bus/usb/BBB/DDD \
  -v "$PWD:/workspace" -w /workspace/examples/deterministic-pi-control \
  mcu-dev/toolchain:local bash
```

`BBB/DDD` 必须替换为本次 `lsusb` 结果。项目不要求 `--privileged`。
设备权限不足时由宿主管理员审查 udev 规则；本项目不自动修改共享系统配置。

## OpenOCD 与 GDB

终端一：

```bash
docker exec -it -w /workspace mcu-shell \
  openocd -f toolkit/openocd/boards/robomaster-c.cfg
```

终端二：

```bash
docker exec -it mcu-shell gdb-multiarch \
  build/stm32f407-robomaster-c/firmware/robomaster-c-pi-control-example.elf \
  -ex "target extended-remote localhost:3333"
```

两个进程位于同一个 `mcu-shell` 容器，因此 GDB 使用 `localhost:3333`。
后续 `load`、观察点和验收变量见
[OpenOCD/GDB 后端](backends/openocd-gdb.md#实板调试验收)。调试结束后执行
`docker stop mcu-shell`；`--rm` 会移除已停止容器。
