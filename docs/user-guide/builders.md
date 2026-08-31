# 选择本地或远程 Builder

Builder 决定一次 workspace 构建在哪里执行。Profile 决定构建所需的环境与产物，两者互不替代。
Project 只保存逻辑名称；服务器地址和 mTLS 凭据留在开发机的 Buildx 配置中。

开发机已经有 Profile 对应 image 时选择 `local`；战队提供了 BuildKit、客户端证书和可拉取的
environment digest 时选择远程 Builder。两种方式生成同一种本地 Build Output。

## 本地 Builder

`local` 是内建 Builder。它要求开发机已经安装 Docker Desktop 或 Docker Engine，并且 Docker
daemon 可用：

```bash
rm-relay builder check local
```

按[镜像选择与运行](image-selection.md)取得 Profile 对应的 development image 后，即可构建：

```bash
rm-relay build --builder local
```

当前尚未发布可直接拉取的正式 image。以下仅是维护者/候选验证路径：要求开发机已安装 mise、
候选 `rm-relay` CLI 已可用，并从本仓库根目录执行：

```bash
mise trust
mise install
mise run environment:embedded:load
rm-relay builder check local
```

正式 image 发布后，这段维护者路径将替换为 Registry 拉取入口。

若 `rm-relay.toml` 没有声明 `default_builder`，CLI 同样选择 `local`。RM Relay 只检查并调用
Docker，不静默安装软件或修改系统服务。

## 登记远程 Builder

远端服务必须已经由战队运维部署，并向你提供 endpoint、TLS server name、CA、客户端证书和
客户端私钥。登记过程调用 Buildx remote driver；证书路径不会写入 Project。

```bash
rm-relay builder add team \
  --endpoint tcp://build.example.org:1234 \
  --ca /absolute/path/ca.pem \
  --cert /absolute/path/client.pem \
  --key /absolute/path/client-key.pem \
  --server-name build.example.org
```

远程 BuildKit 无法使用开发机的本地 image tag。运维还需提供与 Profile environment 对应、已经
推送到 OCI Registry 的不可变引用：

```bash
rm-relay builder set-environment team embedded-development \
  registry.example.org/rm-relay/embedded-development@sha256:<64位十六进制摘要>
```

RM Relay 不接受 mutable tag。当前仓库也不部署 Registry；该引用必须能由远端 BuildKit 拉取。

最后执行真实检查：

```bash
rm-relay builder check team
rm-relay build --builder team
```

`check` 不只是探测端口：它先让 Buildx bootstrap 对应 builder，再执行一个无网络 scratch solve，
并把结果导出回开发机。该检查证明远程控制面可用；只有随后执行真实 `build`，才能同时证明
environment digest 可被 Registry 拉取并完成项目构建。

## Project 默认值

希望项目成员默认使用同名战队 Builder 时，在 `rm-relay.toml` 声明：

```toml
default_builder = "team"
```

每位成员仍需在自己的开发机登记 `team`。临时切回本地不必修改文件：

```bash
rm-relay build --builder local
```

用 `rm-relay builder list` 查看本机可解析的逻辑名称。远程构建通过 Buildx local exporter 把
Build Output 直接返回 Project 的 `install/<profile>/`；后续烧录仍消费开发机上的同一份输出。

不再使用某个远程 Builder 时，同时删除逻辑登记与 RM Relay 创建的 Buildx 资源：

```bash
rm-relay builder remove team
```

逻辑映射保存在操作系统用户配置目录下的 `rm-relay/builders.toml`，文件由 CLI 以 `0600` 原子
更新，不需要手工编辑。测试或非标准安装可用 `RM_RELAY_CONFIG_DIR` 改变配置根目录。
