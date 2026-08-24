# dfu-util 烧录后端

`dfu-util` 对应 STM32 ROM USB DFU 的烧录后端，可枚举、写入和回读 Flash；它不能
暂停 CPU、设置断点或提供源码级调试，因此不能产生 `debug-tested` 状态。

## 运行位置

- macOS、Windows：默认在宿主运行，Docker Desktop 不作为原始 USB 透传承诺。
- Linux：可以在宿主运行，也可在显式映射 USB 设备并设置权限后从容器运行。

项目镜像包含 `dfu-util`，但宿主运行路径仍需宿主单独安装该工具。多设备场景使用
命令行参数选择设备，不把 USB 序列号写入项目配置。

## 安全枚举

默认只执行：

```bash
dfu-util -l
```

对 RoboMaster C，确认输出包含：

```text
0483:df11
alt=0, name="@Internal Flash  /0x08000000/..."
```

枚举成功只记为 `detected`。不要对 `alt=1` Option Bytes、`alt=2` OTP Memory 或
`alt=3` Device Feature 执行普通固件命令。

## 写入用户 Flash

> 警告：以下命令会覆盖目标板用户 Flash 中的现有固件。确认板型、DFU 设备、
> `alt=0`、起始地址和固件产物无误，并明确接受丢失原厂固件后才能执行。

```bash
dfu-util -d 0483:df11 -a 0 -s 0x08000000:leave \
  -D examples/deterministic-pi-control/build/stm32f407-robomaster-c/firmware/robomaster-c-pi-control-example.bin
```

命令成功只能记为 `flashed`。若板上同时存在多个相同 VID:PID 的设备，必须使用本次
运行参数精确选择目标，不要把个人设备标识提交到仓库。

## 按产物大小回读比对

回读不会恢复写入前的原厂固件，只用于校验刚写入的范围。执行写入后，可按实际产物
大小回读并比较：

```bash
firmware_path=examples/deterministic-pi-control/build/stm32f407-robomaster-c/firmware/robomaster-c-pi-control-example.bin
firmware_size=$(wc -c < "$firmware_path" | tr -d ' ')
dfu-util -d 0483:df11 -a 0 \
  -s "0x08000000:$firmware_size" -U /tmp/robomaster-c-readback.bin
cmp "$firmware_path" /tmp/robomaster-c-readback.bin
```

只有回读命令与 `cmp` 都成功，才可记为 `flash-verified`。该结果仍不证明应用已启动；
启动证据应由 UART、LED、USB CDC 或调试器等 observation backend 单独产生。

普通流程禁止 mass erase、unprotect、Option Bytes 和 OTP 操作。出现设备或存储区不符
时立即停止，先检查 BOOT 配置、数据线、USB 权限和目标选择。

命令参数含义以 [dfu-util 官方手册](https://dfu-util.sourceforge.net/dfu-util.1.html)
为准；设备特有地址和 BOOT 条件仍以对应 board 文档为准。
