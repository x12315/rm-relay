# 镜像版本基线

本目录记录 `embedded-development` 的产品级版本契约，不再逐个锁定 APT 包。

`versions.env` 固定 Ubuntu LTS 系列、native GCC major、Arm GNU release 和 uv 版本。
Ubuntu APT 包跟随 `noble`、`noble-updates` 与 `noble-security`；每次发布镜像中的实际
求解结果保存为：

```text
/opt/embedded-development/base-packages.txt
/opt/embedded-development/embedded-packages.txt
```

## 一致性边界

- `stable` 是面向新手的可移动入口。
- 版本 tag 表达项目版本与 Ubuntu LTS，例如 `v0.2.0-ubuntu24.04`。
- OCI manifest digest 是已发布环境的不可变身份。
- 未来重新运行 Dockerfile 会取得同一 LTS 系列的新安全更新，不承诺字节级重建。
- uv 和 CA 引导镜像仍使用 tag 加 digest；Ubuntu 基础镜像按 `24.04` LTS tag 更新。

## 更新方式

1. 只在需要新的语言、架构或缺陷修复能力时升级 GCC major 或 Arm GNU release。
2. 构建前检查所选 Ubuntu 镜像站的同步状态。
3. 分别构建 `linux/arm64` 与 `linux/amd64`，运行镜像 smoke、模板和示例 workflow。
4. 保存并审查两个架构的工具输出和 `dpkg` 清单。
5. 发布不可变版本 tag，并记录 OCI manifest digest。

APT 必须继续使用 `/usr/share/keyrings/ubuntu-archive-keyring.gpg`。不得使用
`Trusted: yes`、`AllowUnauthenticated` 或 `AllowInsecureRepositories`。
