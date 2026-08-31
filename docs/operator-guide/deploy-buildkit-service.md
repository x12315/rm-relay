# 部署 mTLS BuildKit 服务

本页供战队运维在 Linux `amd64` 或 `arm64` 服务器上部署远程 workspace builder。普通开发者只需
按[选择本地或远程 Builder](../user-guide/builders.md)登记服务。

## 准备

服务器需要 Docker Engine、Docker Compose plugin、mise，以及本仓库 checkout。下列命令都从
仓库根目录执行。首次 checkout 先执行 `mise trust` 与 `mise install`。另准备一个持久目录保存三份
PEM 文件：

```text
ca.pem       信任客户端证书的 CA
cert.pem     BuildKit 服务端证书
key.pem      服务端私钥
```

服务端证书的 SAN 必须覆盖开发者在 `--server-name` 中使用的名称。客户端证书由同一信任体系签发，
但不放到服务器的 Compose 目录。RM Relay 不提供 CA 或证书签发服务。

私钥目录只允许部署运维者和运行 BuildKit 的容器用户读取，禁止 world-readable；挂载前应在目标
服务器上核对 owner 与权限，而不是把密钥复制进仓库。

## 检查并启动

服务固定监听容器内 TCP 1234。先只把该端口绑定到宿主 loopback，核对 Compose 展开结果：

```bash
export RM_RELAY_BUILDKIT_TLS_DIR=/absolute/path/to/server-tls
export RM_RELAY_BUILDKIT_LISTEN_ADDRESS=127.0.0.1
mise run service:buildkit:config
```

展开结果中应看到宿主 `127.0.0.1:1234`、只读 `/run/rm-relay-tls`、三个 TLS daemon 参数和
`buildkit-state` volume。`systempaths=unconfined` 是 BuildKit 官方 rootless 容器方案为独立
`/proc` sandbox 要求的设置；本配置不使用 privileged 或隔离更弱的
`--oci-worker-no-process-sandbox`。确认无误后启动：

```bash
mise run service:buildkit:up
docker compose --file services/buildkit/compose.yaml ps
docker compose --file services/buildkit/compose.yaml logs buildkit
```

若 Ubuntu 24.04 或更高版本日志报告 unprivileged user namespace 被 AppArmor 拒绝，按
[BuildKit rootless 官方说明](https://github.com/moby/buildkit/blob/master/docs/rootless.md#ubuntu-2404-or-later)
检查宿主内核策略。RM Relay 不自动修改 sysctl；这是部署者需要明确评估的主机级安全设置。

默认地址不会被局域网访问。需要对外服务时，由运维显式设置可达监听地址，并负责主机防火墙、
路由、DNS 或其他网络基础设施：

```bash
export RM_RELAY_BUILDKIT_LISTEN_ADDRESS=0.0.0.0
mise run service:buildkit:config
mise run service:buildkit:up
docker compose --file services/buildkit/compose.yaml ps
```

环境变量只影响下一次 Compose apply；重新执行 `up` 后，新的宿主端口绑定才会生效。

不要在没有 mTLS 的反向代理或端口转发后暴露 buildkitd。持有受信客户端证书的用户具备提交任意
BuildKit solve 的能力，因此首版只适合可信战队成员和受邀用户。

## 从开发机验证

运维需要向每位开发者交付 endpoint、TLS server name、CA、客户端证书、客户端私钥，以及该
Builder 可拉取的 environment digest；维护者验收时还应记录 Compose 使用的 BuildKit image
digest。开发者使用 `rm-relay builder add` 登记后再运行：

```bash
rm-relay builder check team
```

该命令验证 TLS、Buildx remote driver、BuildKit worker、scratch solve 和 local exporter。它不
访问 workspace environment；通过后，再用
`builder set-environment` 登记远端可拉取的 environment digest；Registry 的部署和登录方式由战队
自行选择。

任一已经登记该 Builder 的开发机也可以通过模块 task 组合同一检查：

```bash
export RM_RELAY_BUILDER=team
mise run service:buildkit:verify
```

## Cache 与停止

BuildKit state 保存在 Compose volume `buildkit-state`。普通停止不会删除它：

```bash
mise run service:buildkit:down
```

源码、Build Output 和用户凭据不应写入 development image。Build context 只服务一次 solve；
Build Output 经 local exporter 返回开发机，服务端长期保留的是 BuildKit 管理的 layer/cache 数据。
GC 基线在 `buildkitd.toml` 中集中维护。
