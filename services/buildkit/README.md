# BuildKit 服务模块

本目录是 RM Relay 远程 workspace 构建的单节点部署定义，面向战队运维。BuildKit 使用
`moby/buildkit:v0.32.2-rootless@sha256:504731e577c20559c00f968f33219f30115e70be29ab96728d1d06e963fc494b`
固定版本与 OCI digest，保留 BuildKit cache，并只提供启用 mTLS 的 TCP listener。

- `compose.yaml`：服务、端口、TLS 只读挂载和持久 state volume；
- `buildkitd.toml`：OCI worker 与 GC 基线；
- `tasks.toml`：Compose 配置检查、启停和真实 Builder 检查；
- `verify_config_test.go`：检查唯一 mTLS listener、TLS 目录只读挂载和 rootless 所需的
  `systempaths=unconfined`，并禁止 Docker socket、privileged 容器和 no-process-sandbox。

证书签发、DNS、防火墙、NAT、VPN 和 OCI Registry 不由本模块管理。部署步骤、网络边界与客户端
登记方式见[部署 mTLS BuildKit 服务](../../docs/operator-guide/deploy-buildkit-service.md)。服务默认
只发布在宿主 `127.0.0.1:1234`；由运维决定何时改为可达地址。
