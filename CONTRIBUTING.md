# 参与贡献

## 提交前

提交前先搜索已有 Issue 和 Pull Request。文档修正与范围明确的小改动可以直接提交；新增
开发环境、工具链、平台，或改变项目边界时，请先创建 Issue 说明问题和使用场景。

## 分支与 PR

普通修改从 `develop` 创建 `feature/*`，并通过 PR 合回 `develop`。准备正式版本时，从
`develop` 创建 `release/*`，发布后合入 `main` 并同步回 `develop`；针对已发布版本的
紧急修复使用 `hotfix/*`，完成后同样同步回 `develop`。

一个 PR 只处理一项清晰变更。新建 PR 时，GitHub 会自动载入
[PR 模板](.github/pull_request_template.md)，提示提交者说明问题、解决方案、发布说明条目、
备选方案、测试覆盖和补充材料。填写时删除首尾的 HTML 注释标记、模板提示及不适用的章节，
不保留空标题。长期保留的逻辑和缺陷修复应有测试；使用方式发生变化时同步修改文档。验证入口见
根级 mise task；修改 environment 定义时另按
[`embedded-development` 维护指南](environments/embedded-development/MAINTAINING.md)完成构建与验证。

## 审查规则

`main` 和 `develop` 只接受 PR，不允许直接推送、force push 或删除。当前仓库由一名维护者
建立首版基线，required approvals 暂为 `0`，合并前由维护者在 PR 中审查 diff、验证结果
与未覆盖范围。出现第二名具有 `write` 权限的稳定维护者后，将 required approvals 提升为
`1`；建立稳定 CI 后，再把对应检查设为合并前的 required checks。

本项目采用 [Apache License 2.0](LICENSE)。提交贡献表示你有权提供相关内容，并同意被
项目接纳的部分按同一许可证发布；引入第三方内容时请保留其许可证和必要归属。
