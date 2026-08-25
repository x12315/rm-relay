# RoboMaster C 型开发板

RoboMaster C 是仓库首个 board profile，不是项目结构的中心。通用控制代码不得依赖
本页的芯片、BOOT 或接口细节。

## 稳定设备事实

| 项目 | 值 |
|---|---|
| MCU | STM32F407IGH6 |
| CPU/FPU | Cortex-M4F，`fpv4-sp-d16`，hard-float |
| 构建 preset | `stm32f407-robomaster-c` |
| 用户 Flash 起始地址 | `0x08000000` |
| ROM USB DFU 标识 | `0483:df11` |
| DFU 用户 Flash 接口 | `alt=0 @Internal Flash` |
| SWD 板端插座 | Molex PicoBlade 1.25 mm，`53261-0471` |

构建配置使用 1 MiB Flash 和连续的 128 KiB SRAM1+SRAM2。64 KiB CCM 当前不在链接
布局中。链接脚本属于每个可独立复制的项目，例如模板中的
`templates/cross-platform-cpp/cmake/board-profiles/stm32f407ighx-flash.ld`；
板型事实没有存放在仓库根目录的全局 CMake 配置中。

## 进入 ROM USB DFU

按板卡手册使用 `BOOT0=1`、`BOOT1=0`，并将 J31 的 pin 1 与 pin 3 短接后复位。
通过板载 Micro-USB 数据口连接宿主，再执行只读枚举：

```bash
dfu-util -l
```

只有输出同时包含 `0483:df11`、`alt=0`、`@Internal Flash` 和 `0x08000000`，才能把
该连接记为 `detected`。`alt=1` Option Bytes、`alt=2` OTP Memory 与 `alt=3`
Device Feature 不属于普通固件流程。

退出 DFU 时恢复板卡正常 BOOT 配置并复位。具体开关、跳线和接口位置以 DJI 官方
RoboMaster Development Board Type C User Manual 为准。

## SWD 与独立 ST-Link

SWD 需要可靠连接 `SWDIO`、`SWCLK`、`GND` 和 `VTref 3.3V`。板端是 Molex
PicoBlade 1.25 mm 插座，不能因为间距相同而购买 JST-GH 线束。该接口已在 macOS
使用独立 ST-Link、宿主 OpenOCD 和容器 GDB 完成 `debug-tested`。

## 当前验证边界

```text
固件构建：cross-compiled
macOS + dfu-util：flash-verified
macOS + OpenOCD/GDB：debug-tested
Linux / Windows 实板链路：未验证
```

烧录会覆盖用户 Flash。只读枚举与实际写入的边界见
[dfu-util 后端](../backends/dfu-util.md)；源码调试流程见
[OpenOCD/GDB 后端](../backends/openocd-gdb.md)。

## 参考资料

- [DJI RoboMaster Development Board Type C 产品页](https://www.robomaster.com/en-US/products/components/general/development-board-type-c)
- [DJI RoboMaster Development Board Type C User Manual](https://rm-static.djicdn.com/tem/35228/RoboMaster%20Development%20Board%20Type%20C%20User%20Manual.pdf)
