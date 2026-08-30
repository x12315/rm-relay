# 跨平台 C++ 项目模板

以这个模板创建独立 Git 项目，可以得到一个同时面向 host 和 STM32 的 C++ 项目起点。
模板只提供工程结构和占位代码，不是示例应用或通用控制算法库。它已经接好：

- 同一份 `portable-code` 被 host test 与 MCU firmware 复用；
- STM32F407 芯片目标与 RoboMaster C 板型分层；
- 标准 CMake configure、build、test 与 workflow presets；
- MCU 路径禁用异常、RTTI 和隐式动态运行时依赖。

模板尚未拆出独立仓库，因此普通用户的稳定 clone 入口尚未交付。候选版本如何验证当前 Git
边界，见[开发者人工核验](../../tests/manual/README.md)。

## 开始使用

创建项目后，先修改顶层 `project()`、CMake target 名称和 C++ 命名空间，再用自己的
领域模型替换 `portable-code` 中的占位代码。不要让可移植代码依赖 ROS 2、HAL、RTOS、
总线帧、文件系统、线程或隐式时钟。

使用 RM Relay 入口时，先为项目生成唯一标识，再由固定开发镜像构建：

```bash
rm-relay init
rm-relay build
```

固件与 `rm-relay-output.json` 位于
`install/embedded-stm32f407-robomaster-c/`。用户只编辑 `rm-relay.toml`，其中声明
Project/Profile、CMake preset 与输出角色；镜像内 mise task 由 RM Relay 维护。

也可以绕过 RM Relay，直接使用 CMake 完成 native 验证：

```bash
cmake --workflow --preset native-clang
cmake --workflow --preset native-gcc
cmake --workflow --preset native-asan
```

或直接交叉编译 RoboMaster C 的 STM32F407 固件：

```bash
cmake --workflow --preset stm32f407-robomaster-c
```

直接调用 CMake 时，中间文件和固件都位于本模板的 `build/` 下；RM Relay 则额外通过
CMake install 导出稳定 Build Output。个人 IDE 或机器覆盖项应写入未提交的
`CMakeUserPresets.json`。

## 目录职责

- `portable-code/`：可同时编译到 host 和 MCU 的纯计算代码。
- `host-tests/`：在开发机上真实执行的测试。
- `firmware/`：目标启动、链接和最小 observation point（观察点）。
- `cmake/toolchains/`：编译器选择，不包含板卡事实。
- `cmake/target-profiles/`：芯片架构与 ABI 编译参数。
- `cmake/board-profiles/`：具体板型、MCU 容量和链接布局。
