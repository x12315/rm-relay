# 故障排查

先保留原始错误输出，再按症状检查。不要通过关闭校验、改用 `--privileged` 或长期
降低安全边界来掩盖根因。

| 症状 | 检查与处理 |
|---|---|
| Docker daemon 不可用 | 运行 `docker info`，确认 Docker Desktop 或 Linux daemon 已启动。 |
| 镜像架构不符 | 运行 `docker inspect --format '{{.Architecture}}' mcu-dev/toolchain:local`；按宿主重新指定 `linux/arm64` 或 `linux/amd64`。 |
| Ubuntu 包暂时无法下载 | 查看所选镜像站同步状态；重新执行完整构建，或由维护者覆盖 `UBUNTU_MIRROR`/`UBUNTU_PORTS_MIRROR`。不要关闭 APT 签名验证。 |
| CMake 误用宿主编译器 | 检查 preset、`CMAKE_TOOLCHAIN_FILE`，以及目标 build 目录 `CMakeCache.txt` 中的 `CMAKE_CXX_COMPILER`。必要时删除对应派生 build 目录后重新配置。 |
| Sanitizer 链接失败 | 运行镜像内 `/usr/local/lib/embedded-development/smoke/verify-base-tools.sh`；确认 `libclang-rt-18-dev` 在镜像包清单中。 |
| DFU 未枚举 | 确认板卡已进入 ROM DFU、使用数据线而非仅供电线，并检查宿主 USB 权限；macOS/Windows 默认在宿主运行 `dfu-util -l`。 |
| DFU 只出现错误存储区 | RoboMaster C 普通固件必须使用 `alt=0 @Internal Flash`；看到 Option Bytes、OTP 或 Device Feature 时不要写入。 |
| Linux 容器看不到 USB | 先在宿主确认设备和权限，再按实际 `/dev/bus/usb/BBB/DDD` 映射；不要用长期 `--privileged` 掩盖权限问题。 |
| ST-Link 未检测到 | 检查目标供电、USB 数据线、转接器、`lsusb` 或 macOS 系统信息，再检查 OpenOCD 版本。 |
| `target not halted` | 先检查 SWDIO、SWCLK、GND、参考电压和复位；再临时加 `-c "adapter speed 400"`。验证前不要修改默认 1800 kHz。 |
| GDB 在 macOS 无法连接 | 确认宿主 OpenOCD 正在监听 3333，且容器能解析 `host.docker.internal`。 |
| 程序在首个浮点运算处异常 | 反汇编 `Reset_Handler`，确认 CPACR 的 CP10/CP11 已启用且随后存在 `dsb`、`isb`。 |
| 出现不允许的 C++ runtime 符号 | 运行下面的禁止符号检查，定位引入源。 |

禁止符号检查：

```bash
arm-none-eabi-nm -C --undefined-only \
  examples/deterministic-pi-control/build/stm32f407-robomaster-c/firmware/robomaster-c-pi-control-example.elf \
  | grep -E 'operator new|operator delete|__cxa_throw|__gxx_personality_v0|__cxa_guard'
```

没有输出才符合当前受限 C++ 契约。`grep` 无匹配返回非零是正常结果。
