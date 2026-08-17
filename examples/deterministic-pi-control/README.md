# 确定性 PI 控制示例

这份 PI 控制项目演示跨平台 C++ 模板的完整用法，也用于功能体验和仓库验收。它不是
供用户复制的模板，也不作为通用控制库发布。

示例用同一个 `PiController` 完成四类验证：

- Clang 与 GCC host 单元测试；
- ASan/UBSan 运行时检查；
- STM32F407 freestanding 交叉编译；
- 固件 observation point（观察点）中的确定性输出检查。

## 运行

```bash
cmake --workflow --preset native-clang
cmake --workflow --preset native-gcc
cmake --workflow --preset native-asan
cmake --workflow --preset stm32f407-robomaster-c
```

## 目录职责

- `portable-controller/`：不依赖 OS、驱动或通信协议的 PI 控制示例。
- `host-tests/`：固定输入、时间和期望输出的六组测试向量。
- `firmware/`：在 F407 固件中调用同一控制器并暴露 GDB 观察点。
- `cmake/`：示例自己的独立工具链、芯片目标和板型配置。

`portable-controller` 只描述本示例内部“可移植控制器”的角色。它不应被其他项目通过
仓库相对路径直接引用；需要复用时，应先建立正式 API 与版本契约。
