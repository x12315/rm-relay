# 验证体系

RM Relay 的验证按证据层级组织，不用一条端到端脚本代替模块深度。日常修改先跑快速
门禁，改动镜像、模板或主链路时再进入 Docker acceptance；硬件结论始终单独记录。

## 默认顺序

从仓库根目录运行：

```bash
go test ./...
go vet ./...
sh validation/contracts/verify-repository-layout.sh
sh validation/contracts/verify-toolchain-source-policy.sh
sh validation/platform/verify-cli-build-matrix.sh
```

这组检查不启动 development image，适合开发过程中反复执行。

修改并发执行或共享缓存边界时，再运行 race detector：

```bash
go test -race ./internal/execution/...
```

修改 Dockerfile、CMake workflow、Project Template、Profile、Build Output 或 target adapter 后，
追加以下检查。它们要求 Docker daemon 可用，并消费已经构建或拉取的 development image；
不会代替镜像构建流程：

```bash
sh validation/acceptance/verify-project-builds.sh
sh validation/acceptance/verify-local-mcu-cycle.sh
```

## 每层负责什么

| 层级与入口 | 证明的事实 | 不能证明 |
|---|---|---|
| Module tests：`go test ./...` | schema、digest、Workflow、命令参数、取消和稳定 CLI 结果 | Docker、跨平台运行或硬件行为 |
| Module boundaries：同属 `go test ./...` | `project`、`profile` 保持叶模块，其余内部模块的直接依赖方向没有倒置 | 模块组合后的运行行为 |
| Contracts：`validation/contracts/*.sh` | 列举的关键资产、禁止路径、工具版本和软件源策略没有回退 | 未列举资产的任意归属，或 development image 能运行 |
| Platform builds：`verify-cli-build-matrix.sh` | CLI 可为三种 OS 的 amd64/arm64 目标编译 | 二进制已在对应系统运行 |
| Project acceptance：`verify-project-builds.sh` | 已有 development image 能构建、测试模板与示例 | CLI 主链路或实机烧录 |
| MCU cycle acceptance：`verify-local-mcu-cycle.sh` | CLI、Docker、CMake Install、manifest 与 OpenOCD dry-run 能组合工作 | OpenOCD 已连接或固件已写入 |
| Hardware evidence：独立实机记录 | `detected`、`flashed`、`boot-observed`、`debug-tested` 等实际达到的状态 | 未记录的更高等级状态或其他平台 |

Module integration test 使用真实 Project、Profile、Plan 和 Build Output，只在 Docker、OpenOCD
等进程边界使用 fake。这样既能定位模块回归，也不会把模拟进程误报成真实工具
或硬件通过。

模块内部行为测试就近放置；跨模块依赖方向属于仓库契约，集中放在
`validation/module-boundaries/`。Acceptance 只验证模块组合后的真实工具链，不替代前两者。

## 证据边界

构建、平台与硬件证据不能互相升级；`cross-compiled` 不能写成 `boot-observed` 或
`debug-tested`。硬件状态的完整定义见
[开发契约参考](../docs/reference/development-contracts.md#证据等级)。

## 候选镜像

Acceptance 默认使用 `mcu-dev/toolchain:local`。验证待发布镜像时显式覆盖：

```bash
DEVELOPMENT_IMAGE=registry.example/embedded-development:candidate \
  sh validation/acceptance/verify-project-builds.sh
```

`verify-local-mcu-cycle.sh` 验证 builtin Profile 当前固定的 development image，不接受镜像
覆盖，避免验证结果偏离 Profile 契约。
