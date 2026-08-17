# 仓库验证资产

本目录面向开发环境维护者，不是用户项目模板，也不承载控制算法。

- `project-contracts/verify-repository-layout.sh` 检查四类资产边界和局部 README。
- `project-contracts/verify-toolchain-source-policy.sh` 检查 LTS、国内源、APT 安全边界和编译器选择。
- `project-contracts/verify-project-builds.sh` 在正式开发镜像中构建模板与 PI 示例。

从仓库根目录运行：

```bash
sh validation/project-contracts/verify-repository-layout.sh
sh validation/project-contracts/verify-toolchain-source-policy.sh
sh validation/project-contracts/verify-project-builds.sh
```

第二条命令默认使用 `mcu-dev/toolchain:local`。验证其他候选镜像时显式传入：

```bash
DEVELOPMENT_IMAGE=registry.example/embedded-development:candidate \
  sh validation/project-contracts/verify-project-builds.sh
```
