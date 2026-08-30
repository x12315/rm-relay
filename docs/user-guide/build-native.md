# native 构建、测试与分析

CMake Presets 属于各个可独立复制的项目，不在仓库根目录。先选择一个项目：

- `project-templates/cross-platform-cpp` 是用户复制、重命名并继续开发的模板；
- `examples/deterministic-pi-control` 是体验完整功能和审查实现的示例。

以下以 PI 示例为例，从仓库根目录启动容器并把工作目录设为该项目：

```bash
docker run --rm -it \
  -v "$PWD:/workspace" -w /workspace/examples/deterministic-pi-control \
  mcu-dev/toolchain:local bash
```

## 标准 workflow 入口

```bash
cmake --workflow --preset native-clang
cmake --workflow --preset native-gcc
cmake --workflow --preset native-asan
```

每个 workflow 都依次执行 configure、build 和 test。需要单独控制某一步时，仍可使用
标准 configure/build/test preset：

```bash
cmake --preset native-clang
cmake --build --preset native-clang
ctest --preset native-clang
```

`native-asan` 使用 Clang 的 AddressSanitizer 与 UndefinedBehaviorSanitizer，在测试实际
运行时检测越界、use-after-free（释放后使用）和未定义行为等问题。它不是静态流分析器；
`clang-tidy` 和 clangd 承担静态检查。

## 静态分析与格式检查

```bash
clang-format --dry-run --Werror \
  portable-controller/include/pi_control_example/pi_controller.hpp \
  portable-controller/src/pi_controller.cpp \
  host-tests/pi_controller_test.cpp
clang-tidy -p build/native-clang \
  portable-controller/src/pi_controller.cpp
```

Clang 的编译数据库位于 `build/native-clang/compile_commands.json`，可直接提供给
编辑器、语言服务器和 CLI agent。模板项目采用相同 preset 名称和操作方式。每个项目的
`build/` 都是派生资产，不提交到仓库。
