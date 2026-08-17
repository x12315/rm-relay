# 跨平台 C++ 项目模板

这是供用户复制并重命名的项目起点，不是示例应用或通用控制算法库。模板展示：

- 同一份 `portable-code` 被 host test 与 MCU firmware 复用；
- STM32F407 芯片目标与 RoboMaster C 板型分层；
- 标准 CMake configure、build、test 与 workflow presets；
- MCU 路径禁用异常、RTTI 和隐式动态运行时依赖。

## 开始使用

复制整个目录后，先修改顶层 `project()`、CMake target 名称和 C++ 命名空间，再用自己的
领域模型替换 `portable-code` 中的占位代码。不要让可移植代码依赖 ROS 2、HAL、RTOS、
总线帧、文件系统、线程或隐式时钟。

完整 native 验证：

```bash
cmake --workflow --preset native-clang
cmake --workflow --preset native-gcc
cmake --workflow --preset native-asan
```

交叉编译 RoboMaster C 的 STM32F407 固件：

```bash
cmake --workflow --preset stm32f407-robomaster-c
```

生成目录统一位于本模板的 `build/` 下。个人 IDE 或机器覆盖项应写入未提交的
`CMakeUserPresets.json`。

## 目录职责

- `portable-code/`：可同时编译到 host 和 MCU 的纯计算代码。
- `host-tests/`：在开发机上真实执行的测试。
- `firmware/`：目标启动、链接和最小 observation point（观察点）。
- `cmake/toolchains/`：编译器选择，不包含板卡事实。
- `cmake/target-profiles/`：芯片架构与 ABI 编译参数。
- `cmake/board-profiles/`：具体板型、MCU 容量和链接布局。
