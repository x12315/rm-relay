# VS Code / Cortex-Debug 接入示例

仓库不提交 `.vscode/` 或 `.devcontainer/`，也不自动安装扩展。IDE 配置属于用户层；
构建、测试和调试契约仍以 Docker、CMake Presets、OpenOCD 和 GDB 命令为准。
外部 OpenOCD 的启动方式和验收标准见
[OpenOCD/GDB 烧录与调试后端](backends/openocd-gdb.md)。

以下片段可由用户放入自己的 `.vscode/launch.json`。它假设 VS Code 打开的
`workspaceFolder` 是 `examples/deterministic-pi-control`，Cortex-Debug 已附加到正在
运行的 `mcu-dev` 容器，且 macOS 宿主 OpenOCD 已监听 3333 端口：

```json
{
  "type": "cortex-debug",
  "request": "launch",
  "name": "RoboMaster C via external OpenOCD",
  "servertype": "external",
  "gdbPath": "/usr/bin/gdb-multiarch",
  "gdbTarget": "host.docker.internal:3333",
  "executable": "${workspaceFolder}/build/stm32f407-robomaster-c/firmware/robomaster-c-pi-control-example.elf",
  "device": "STM32F407IGH6",
  "runToEntryPoint": "pi_control_example_observation_ready"
}
```

在原生 Linux 容器直连流程中，将 `gdbTarget` 改为 `localhost:3333`。如果 OpenOCD
并非运行在同一容器，则按实际网络边界填写，不要把个人地址提交到仓库。

C/C++ IntelliSense 或 clangd 应读取：

```text
build/native-clang/compile_commands.json
```

这个数据库描述 PI 示例的 native C++ 目标。STM32 专用补全可改用
`build/stm32f407-robomaster-c/compile_commands.json`。
