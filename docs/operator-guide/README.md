# 运维与维护操作

本目录面向部署战队服务，或在正式服务缺失时准备维护资源的人员。运维基线是 Linux
`amd64`/`arm64`；部分命令在 macOS Docker Desktop 上也可能工作，但项目不为运维链路提供完整
macOS 保证。普通开发者的 Win、macOS、Linux 支持不受这一边界影响。

## 长期服务

- [部署 mTLS BuildKit 服务](deploy-buildkit-service.md)：部署远程 workspace builder。

## 备用维护路径

- [准备临时环境来源](prepare-temporary-environment-source.md)：没有正式 Registry 时，在单台 Linux
  主机临时生产候选 OCI reference。

OCI Registry 的长期产品、virtual target 和正式 image automation 尚未交付。镜像定义归
`environments/`，镜像生产归 `services/environment-image-builder/`；本目录只说明如何部署或组合
它们所需的外部资源。
