# 本地 MCU 开发体验

本场景从已经进入候选 shell 开始，检查一名普通使用者能否从 Git Project Template 完成本地
STM32 构建，并找到后续烧录入口。当前人工证据来自 Apple Silicon macOS；其他宿主不能继承
这次结果。

开始前应已按[Candidate 说明](../../support/candidate/README.md)选择 `local` Builder、制备并进入
候选环境。Candidate shell 会显示 CLI、Builder、environment 与 template identity；当前目录是
仓库外的空工作区。

## 取得项目

```bash
git clone "$RM_RELAY_TEMPLATE_URL" project
cd project
git status --short
rm-relay --version
```

判断以下问题：

- clone 命令是否接近正式用户将采用的 Git 流程；
- 项目入口和文件命名能否说明从哪里开始；
- 候选 CLI version 是否容易识别；
- clean project 是否没有 RM Relay 内部测试资产。

## 初始化

```bash
rm-relay init
git diff -- rm-relay.toml
```

判断 CLI 是否解释了修改对象和生成的 Project identity，配置变化是否容易理解。UUID 格式、幂等性
和 schema 保真由自动测试负责，不在这里手工复算。

## 构建

```bash
rm-relay environment check embedded-development --builder local
rm-relay build
```

先判断 Environment 检查能否区分 Builder、Registry 和 image identity 问题；再判断构建输出能否
回答当前使用哪个 Profile、工作进行到哪一步、成功后产物在哪里。ELF 类型、artifact hash 和
manifest 字段由 E2E 自动断言。

## 找到烧录入口

```bash
rm-relay flash --help
rm-relay flash --target openocd-stlink --dry-run
```

判断帮助是否足以发现 target 与 `--dry-run`，以及 dry-run 是否清楚说明命令没有执行、将使用哪个
产物。该步骤没有访问 OpenOCD、ST-Link 或 MCU，因此最高只能记录 OpenOCD `configured`。

## 记录与退出

在 PR 或审查记录中写明 candidate revision、宿主 OS/architecture、每一步遇到的理解阻力，以及
最高证据：

```text
candidate revision:
host OS / architecture:
clone and project entry:
initialization feedback:
build feedback:
flash discovery and dry-run:
highest evidence: cross-compiled; OpenOCD configured
```

完成后退出候选 shell 并回收环境：

```bash
exit
mise run experience:clean
```

`experience:clean` 删除本场景产生的外置 workspace 和隔离配置，不删除借用的
`rm-relay-local-workspace-buildx`、
environment image 或 build cache。若本次使用了临时 Registry，回到对应运维 how-to 另行清理。
