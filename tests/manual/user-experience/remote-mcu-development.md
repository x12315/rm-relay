# 远程 MCU 构建体验

本场景检查普通用户能否从一台开发机登记战队 Builder、完成真实远程构建，并继续使用返回本地的
Build Output。它要求已有可达的 mTLS BuildKit 服务和远端可拉取的 environment digest。

## 制备测试用品

维护者先运行自动测试，并按[候选体验环境](../../../docs/operator-guide/candidate-experience-environment.md)
制备仓库外候选 CLI、development image、模板 Git origin 和空 workspace：

```bash
mise run experience:prepare
mise run experience:enter
```

`experience:enter` 成功后，候选 shell 已把 `rm-relay` 放入 `PATH`，并设置
`RM_RELAY_TEMPLATE_URL`。先确认变量非空：

```bash
test -n "$RM_RELAY_TEMPLATE_URL"
rm-relay --version
```

另外准备运维提供的 endpoint、TLS server name、CA、客户端证书、私钥、BuildKit image digest 和
environment digest。
证书内容不得写入测试记录。

## 登记与检查 Builder

在候选 shell 中输入：

```bash
rm-relay builder add team \
  --endpoint tcp://<server>:1234 \
  --ca <absolute-ca-path> \
  --cert <absolute-client-certificate-path> \
  --key <absolute-client-key-path> \
  --server-name <tls-server-name>

rm-relay builder list
rm-relay builder check team
rm-relay environment add embedded-development <image@sha256:64位小写十六进制摘要> --builder team
rm-relay environment list --builder team
```

判断参数错误是否指出问题所在，Builder `list` 是否能区分逻辑名称与实现 kind，`check` 的进度与
失败信息能否让使用者判断是 TLS、Buildx 还是 BuildKit solve 出错；Environment 登记还应明确
区分 Registry 拉取失败、identity 不匹配和登记成功。

## 走完整用户路径

```bash
git clone "$RM_RELAY_TEMPLATE_URL" project
cd project
rm-relay init
rm-relay build --builder team
rm-relay flash --target openocd-stlink --dry-run
```

判断远程构建的输出是否持续返回当前终端，成功后是否清楚指出本地
`install/embedded-stm32f407-robomaster-c/rm-relay-output.json`；不要求使用者登录编译服务器或
手动下载压缩包。dry-run 必须继续消费开发机上的 Build Output。

## 记录与回收

记录 candidate revision、开发机系统与架构、BuildKit 服务版本、environment digest、各步骤的理解
阻力和最高证据。不得记录私钥、证书正文或本机绝对凭据路径。

```text
candidate revision:
workstation OS / architecture:
BuildKit image digest supplied by operator:
environment digest:
builder registration and check:
remote build feedback:
local Build Output and flash discovery:
evidence reached: remote solve completed; cross-compiled; OpenOCD configured
```

先退出候选 shell：

```bash
exit
```

回到仓库根目录后，再回收候选环境：

```bash
mise run experience:clean
```

本场景不验证 Registry 部署、证书签发、网络穿透、真实烧录或源码调试。
