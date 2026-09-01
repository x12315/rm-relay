# 选择本地或远程 Builder

Builder 决定一次 workspace 构建在哪里执行。Profile 声明所需 environment，Builder 再把该
environment 映射到不可变 OCI image。Project 只保存逻辑 Builder 名称；具体 Buildx 资源、
服务器地址和 mTLS 凭据留在开发机。

开发机已有 Docker 与 Buildx 时选择 `local`；战队提供了 BuildKit、客户端证书和可拉取的
environment digest 时选择远程 Builder。两者执行同一 BuildKit frontend，最终都把 Build Output
返回开发机。

本页的构建步骤以“已经从维护者取得 Profile 对应的 immutable environment digest”为前提。
正式 digest 尚未发布时，可以审查和验证 Builder 控制面，但不能据此声称普通用户入口已经可用。

## 本地 Builder

`local` 是内建 Builder。RM Relay 不安装 Docker；开发机必须已有 Docker Desktop 或 Docker
Engine，并提供 Buildx。

先登记 Profile environment 对应的不可变 OCI 引用：

```bash
rm-relay builder set-environment local embedded-development \
  registry.example.org/rm-relay/embedded-development@sha256:<64位十六进制摘要>
```

然后直接构建：

```bash
rm-relay build --builder local
```

第一次构建时，CLI 会准备名为 `rm-relay-local` 的 Buildx `docker-container` resource，并把
BuildKit image 固定到 OCI digest；后续构建复用该资源及其 cache。CLI 不执行 `docker buildx use`，不会
改变用户在其他项目中选择的 Builder。也可以单独准备或检查：

```bash
rm-relay builder prepare local
rm-relay builder check local
```

`prepare` 只保证受管 Buildx resource 可用；`check` 还会执行一个无网络 scratch solve，并把
结果导回开发机。真正的 `build` 才能证明 environment digest 可被拉取并完成项目构建。

若 `rm-relay.toml` 没有声明 `default_builder`，CLI 同样选择 `local`。RM Relay 只检查并调用
现有 Docker，不静默安装软件或修改系统服务。当前尚未发布正式 environment digest，因此
普通用户还没有稳定的本地构建入口。

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

远程与本地 Builder 都只接受已经推送到 OCI Registry 的不可变引用：

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

`check` 不只是探测端口：它先让 Buildx bootstrap 对应 Builder，再执行与本地相同的 scratch
solve。该检查证明远程控制面可用；只有随后执行真实 `build`，才能同时证明 environment digest
可被 Registry 拉取并完成项目构建。

## Project 默认值

希望项目成员默认使用同名战队 Builder 时，在 `rm-relay.toml` 声明：

```toml
default_builder = "team"
```

每位成员仍需在自己的开发机登记 `team`。临时切回本地不必修改文件：

```bash
rm-relay build --builder local
```

用 `rm-relay builder list` 查看本机可解析的逻辑名称。本地和远程构建都通过 Buildx local
exporter 把 Build Output 返回 Project 的 `install/<profile>/`；后续烧录消费同一份输出。

不再使用某个远程 Builder 时，同时删除逻辑登记与 RM Relay 创建的 Buildx 资源：

```bash
rm-relay builder remove team
```

逻辑映射保存在操作系统用户配置目录下的 `rm-relay/builders.toml`，文件由 CLI 以 `0600` 原子
更新，不需要手工编辑。测试或非标准安装可用 `RM_RELAY_CONFIG_DIR` 改变配置根目录。
