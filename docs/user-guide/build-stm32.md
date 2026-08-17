# STM32F407 固件构建

STM32 preset 同样属于独立项目。以下命令在所选项目目录内运行：

```bash
cmake --workflow --preset stm32f407-robomaster-c
```

模板产物位于：

```text
templates/cross-platform-cpp/build/stm32f407-robomaster-c/firmware/
├── robomaster-c-starter.elf
├── robomaster-c-starter.bin
└── robomaster-c-starter.map
```

PI 示例产物位于：

```text
examples/deterministic-pi-control/build/stm32f407-robomaster-c/firmware/
├── robomaster-c-pi-control-example.elf
├── robomaster-c-pi-control-example.bin
└── robomaster-c-pi-control-example.map
```

例如，在 PI 示例目录检查目标架构和大小：

```bash
arm-none-eabi-readelf -h \
  build/stm32f407-robomaster-c/firmware/robomaster-c-pi-control-example.elf
arm-none-eabi-size \
  build/stm32f407-robomaster-c/firmware/robomaster-c-pi-control-example.elf
```

`.elf` 包含段信息和调试符号，供 GDB/OpenOCD 使用；`.bin` 是从 Flash 起始地址写入
的裸二进制；`.map` 用于审查链接段、符号来源和内存占用。

首个 board profile 使用 STM32F407IGH6、Cortex-M4F hard-float、1 MiB Flash 和
128 KiB 连续 SRAM；64 KiB CCM 尚未纳入链接布局。启动代码负责初始化
`.data`/`.bss` 并在执行浮点代码前启用 FPU。

## 验证状态

构建状态：

- `built`：某个构建命令成功。
- `host-tested`：native 测试在宿主架构容器中通过。
- `cross-compiled`：目标 ELF/BIN/MAP 成功生成并通过静态检查。

硬件后端状态：

- `configured`：后端配置和命令入口已建立。
- `detected`：工具已只读枚举目标设备。
- `flashed`：固件已成功写入目标芯片。
- `flash-verified`：写入内容已由工具校验或回读比对。
- `boot-observed`：复位后已观察到约定的启动行为。
- `debug-tested`：调试器已连接、命中观察点并核对约定状态。

两组状态不可相互替代。当前 F407 固件为 `cross-compiled`，macOS DFU 为
`detected`，OpenOCD/GDB 配置为 `configured`；其余四个硬件状态尚未完成验证。

## 配置分层与新增设备

每个项目内的 `cmake/toolchains/` 只选择编译器，`cmake/target-profiles/` 描述
STM32F407/Cortex-M4F 架构与 ABI，`cmake/board-profiles/` 描述 RoboMaster C 的器件
容量和链接布局。

新增 STM32 型号或板卡时增加独立 target/board profile、链接脚本和必要的平台启动
代码，不要把芯片或 HAL 依赖引入 `portable-code` 或 `portable-controller`。STC 需要
独立工具链与设备配置，不属于当前已支持范围。
