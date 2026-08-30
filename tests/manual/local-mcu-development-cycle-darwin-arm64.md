# macOS arm64 本地 MCU 开发链路

## 目的与证据

本场景核验一个候选 commit 能否沿正式交付边界完成：构建 development image、生成并运行当前
平台 CLI、通过 Git 得到独立项目、建立 Project Identity、在真实 Docker 中交叉编译、校验
Build Output，并解析 OpenOCD dry-run。

本场景不连接开发板，最高只产生 STM32F407 `cross-compiled` 与 OpenOCD `configured` 证据。

## 适用组合

| 维度 | 值 |
|---|---|
| 开发机 | Apple Silicon macOS，`darwin/arm64` |
| shell | zsh |
| build backend | `local-container` |
| development image | `mcu-dev/toolchain:local`，`linux/arm64` |
| Profile | `embedded-stm32f407-robomaster-c` |
| target capability | `openocd-stlink`，仅 `--dry-run` |
| 硬件 | 不需要 |

Linux、Windows、Intel macOS、远程 backend 和真实 ST-Link 不在本场景范围内，应使用各自的
独立场景。

## 前置条件

- 从仓库根目录开始，候选改动已经提交，工作区 clean。
- Git 提供 `subtree` 命令。
- Docker Desktop 正在运行，Buildx 可用。
- 宿主 mise 版本不低于根 `mise.toml` 的 `min_version`。
- 网络能够取得镜像构建所需的已锁定上游资产。
- `manual/local-mcu-template` 分支不存在；`dist/manual-workspace/` 不包含需要保留的数据。

首先确认宿主和候选版本。

**操作**

```bash
uname -s
uname -m
git status --short --branch
git rev-parse HEAD
```

**预期结果**

- 前两条分别输出 `Darwin` 与 `arm64`。
- `git status` 显示候选 feature 分支，且没有修改或未跟踪文件。
- 记录完整 commit，后续结果只对该 commit 有效。

**失败说明**

宿主组合不匹配时停止并选择其他场景。工作区不 clean 时先处理现有改动，不能让人工测试覆盖
来源不明的文件。

确认测试专用分支尚未存在。

**操作**

```bash
git branch --list manual/local-mcu-template
```

**预期结果**

没有输出。

**失败说明**

该分支属于上一次未清理的人工测试。先确认其用途，不要直接覆盖或删除来源不明的分支。

## 准备候选产物

### 1. 安装维护工具并检查静态契约

**操作**

```bash
mise trust
mise install
mise --version
mise tasks ls
mise run verify
```

**预期结果**

- mise 版本满足根配置，八个维护 task 均可发现。
- `verify` 分别报告 repository layout 与 toolchain/source policy 通过。

**失败说明**

此处失败属于维护工具版本、仓库拓扑、版本锁定或软件源策略，不应继续用旧 image 或旧 CLI
掩盖问题。

### 2. 构建并检查当前架构 development image

**操作**

```bash
docker buildx bake --file container-images/embedded-development/docker-bake.hcl mcu-dev --load
```

**预期结果**

Buildx 成功构建本机架构的 `mcu-dev` target，并加载为 `mcu-dev/toolchain:local`。

**失败说明**

失败属于 Docker builder、网络、软件源或 image 定义；此时尚未进入用户项目构建。

**操作**

```bash
docker inspect mcu-dev/toolchain:local --format 'id={{.Id}} arch={{.Architecture}}'
docker run --rm mcu-dev/toolchain:local sh -lc '/usr/local/lib/embedded-development/smoke/verify-base-tools.sh && /usr/local/lib/embedded-development/smoke/verify-embedded-tools.sh'
```

**预期结果**

- 记录不可为空的 image ID，架构为 `arm64`。
- smoke 实际运行 native GCC/Clang、C++20、Catch2、mise、Arm GNU、GDB、DFU 与 OpenOCD 检查，
  进程退出码为 0。

**失败说明**

image 能被加载但 smoke 失败时，不能把它作为后续构建环境。

### 3. 生成并解压当前平台 CLI snapshot

`distribution:snapshot` 会替换根目录下 gitignored 的 `dist/`。其中有需要保留的本地产物时先
移走，再执行本步骤。

**操作**

```bash
mise run distribution:snapshot
find dist -maxdepth 1 -type f -name 'rm-relay_*' -print | sort
```

**预期结果**

列出 Darwin、Linux、Windows 的 amd64/arm64 六个 archive，以及一个 checksum 文件。

**失败说明**

目标缺失、重复或命名错误属于 GoReleaser 支持矩阵与分发契约，不进入 CLI 功能核验。

**操作**

```bash
tar -tzf dist/rm-relay_*_darwin_arm64.tar.gz
mkdir -p dist/manual-workspace/bin
tar -xzf dist/rm-relay_*_darwin_arm64.tar.gz -C dist/manual-workspace/bin rm-relay
chmod +x dist/manual-workspace/bin/rm-relay
dist/manual-workspace/bin/rm-relay --version
```

**预期结果**

- archive 只包含 `LICENSE` 和 `rm-relay`。
- `--version` 输出 snapshot 版本，不能是 `development`。
- 记录该 CLI version。

**失败说明**

archive 内容或版本错误属于 CLI 分发，不应用 `go run` 或仓库内临时二进制继续测试。

### 4. 通过 Git 建立独立测试项目

**操作**

```bash
git subtree split --prefix=project-templates/cross-platform-cpp --branch manual/local-mcu-template
git clone --branch manual/local-mcu-template --single-branch . dist/manual-workspace/project
cd dist/manual-workspace/project
export PATH="$(git rev-parse --show-toplevel)/../bin:$PATH"
git status --short
```

**预期结果**

- clone 中只有 Project Template 的历史与文件，不包含 RM Relay 主仓库其他资产。
- `rm-relay` 从 `dist/manual-workspace/bin` 解析。
- 项目工作区 clean。

**失败说明**

subtree 或 clone 失败属于当前 monorepo 模板分发边界；不能改为复制目录，因为本场景要求验证
独立 Git 项目。

## 执行步骤

### 1. 验证 Project Identity 及幂等性

**操作**

```bash
grep '^project_id = ""$' rm-relay.toml
rm-relay init
grep -E '^project_id = "[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}"$' rm-relay.toml
shasum -a 256 rm-relay.toml
```

**预期结果**

- 第一次 `init` 返回成功，并写入 UUID v4。
- 记录第一次 `shasum`。

**失败说明**

空 ID 未被替换、生成值不是 UUID v4 或配置其他字段被意外改写，均属于 Project 模块失败。

**操作**

```bash
rm-relay init
shasum -a 256 rm-relay.toml
```

**预期结果**

第二次 `init` 仍返回相同 Project ID，两次 `shasum` 完全一致。

**失败说明**

第二次执行改变配置或身份，说明初始化不幂等，不能继续用该项目验证 Build Output identity。

### 2. 构建并检查 Build Output

**操作**

```bash
rm-relay build
find install/embedded-stm32f407-robomaster-c -maxdepth 1 -type f -print | sort
sed -n '1,220p' install/embedded-stm32f407-robomaster-c/rm-relay-output.json
```

**预期结果**

- CLI 通过真实 Docker 和 image 内受控 CMake workflow 完成构建。
- 输出目录只包含 ELF、BIN、MAP 与 `rm-relay-output.json`。
- manifest 的 Project ID 与当前项目一致；Profile、development image、image ID、producer version
  均非空且与本场景记录一致。
- 记录 Profile digest。

**失败说明**

Docker 启动前失败通常属于 Project/Profile/Plan；容器内失败属于 image 或 CMake workflow；构建
完成但 manifest 缺失属于 Build Output 交接面。

**操作**

```bash
file install/embedded-stm32f407-robomaster-c/robomaster-c-starter.elf
stat -f '%N %z' install/embedded-stm32f407-robomaster-c/robomaster-c-starter.elf install/embedded-stm32f407-robomaster-c/robomaster-c-starter.bin install/embedded-stm32f407-robomaster-c/robomaster-c-starter.map
shasum -a 256 install/embedded-stm32f407-robomaster-c/robomaster-c-starter.elf install/embedded-stm32f407-robomaster-c/robomaster-c-starter.bin install/embedded-stm32f407-robomaster-c/robomaster-c-starter.map
```

**预期结果**

- `file` 报告 32-bit little-endian ARM ELF。
- 三个文件的实际 size 与 SHA-256 分别等于 manifest 中相同 role 的值。

**失败说明**

文件存在但身份不匹配时，不能把构建结果传给 target adapter。

### 3. 验证 OpenOCD dry-run 的公开输出

**操作**

```bash
rm-relay flash --target openocd-stlink --dry-run
rm-relay --format json flash --target openocd-stlink --dry-run
```

**预期结果**

- human 输出给出未执行的 OpenOCD 命令。
- JSON stdout 是单个对象，包含 `"ok":true`、`"operation":"flash"`、当前 Project/Profile、
  `"executed":false`，command 中包含 `mise`、`openocd`、board cfg 与已验证 ELF 路径。

**失败说明**

此处失败属于 Build Output 重新验证、target capability、OpenOCD adapter 或输出契约；dry-run
不会检查宿主 OpenOCD、ST-Link 或 USB。

### 4. 验证篡改产物会被拒绝

**操作**

```bash
cp install/embedded-stm32f407-robomaster-c/robomaster-c-starter.bin ../robomaster-c-starter.bin.manual-backup
printf '\0' >> install/embedded-stm32f407-robomaster-c/robomaster-c-starter.bin
rm-relay --format json flash --target openocd-stlink --dry-run
echo $?
```

**预期结果**

- `flash` 退出码为 `1`。
- JSON error code 为 `build_output_invalid`，message 指出 `firmware.bin` 的 size 或 SHA-256 已改变。
- OpenOCD 命令没有执行。

**失败说明**

篡改后仍生成 target 命令，说明 Build Output 与 target 之间的内容身份边界失效，应停止候选
版本验收。

恢复产物并确认主链路可继续使用。

**操作**

```bash
mv ../robomaster-c-starter.bin.manual-backup install/embedded-stm32f407-robomaster-c/robomaster-c-starter.bin
rm-relay --format json flash --target openocd-stlink --dry-run
```

**预期结果**

dry-run 再次返回 `"ok":true` 与 `"executed":false`。

**失败说明**

恢复后仍失败时，重新运行 `rm-relay build`，并检查是否还有其他测试外修改。

## 结果记录

人工核验结果至少记录以下内容；可以粘贴到 PR 或审查记录，不提交本机结果文件：

```text
candidate commit:
host OS / architecture:
mise version:
Docker / Buildx version:
CLI version:
development image ID:
Profile ID / digest:
prepare image: PASS | FAIL
prepare distribution: PASS | FAIL
clone project: PASS | FAIL
project identity: PASS | FAIL
build output: PASS | FAIL
flash dry-run: PASS | FAIL
tamper rejection: PASS | FAIL
highest evidence: cross-compiled; OpenOCD configured
notes:
```

## 清理

返回主仓库根目录，先确认测试分支与工作目录名称，再只删除本场景创建的资产。

**操作**

```bash
cd ../../..
git branch --list manual/local-mcu-template
test -d dist/manual-workspace/project
git branch -D manual/local-mcu-template
rm -rf dist/manual-workspace
git status --short --branch
```

**预期结果**

- 只删除 `manual/local-mcu-template` 与 `dist/manual-workspace/`。
- 候选分支和源文件保持 clean。
- `dist/` 中 GoReleaser 生成的 snapshot 可以保留供自动 E2E 使用；development image 与 ccache
  也可保留供后续核验。

**失败说明**

路径或分支名称与预期不一致时停止清理并人工检查，不扩大删除范围。

## 未覆盖

本场景没有验证：

- Linux、Windows 或 Intel macOS 上运行 CLI；
- ST-Link/USB 设备检测与 OpenOCD 实际启动；
- Flash 写入、回读、目标启动、GDB 断点或变量观察；
- remote backend、物理 Linux target、虚拟 target 或数据回收。

这些能力不能从本场景的 PASS 推导，必须由匹配组合的独立人工场景和支持矩阵证据确认。
