# 服务拓扑

本文解释 RM Relay 的环境供应、远程构建和虚拟 target 如何部署。它面向需要判断“某项
服务应该放在哪里、保存什么、由谁访问”的实现者和战队运维人员，不是现成部署手册。

> [!IMPORTANT]
> Environment image 的共用 build/verify/push 入口、Workspace builder 代码与 mTLS Compose
> 配置已经交付；真实 Registry push 和战队服务器仍缺少部署证据。OCI Registry 与 K3s virtual
> target 尚未交付。Registry 采用托管服务还是战队自部署尚未决定。本页记录服务边界，不为待决
> 组件提供假想命令。

## 先按责任划角色，再决定机器数量

RM Relay 不把“一台服务器”当作一个组件。服务角色由四个问题划分：消费什么输入、产生
什么输出、谁有权限访问、必须保存什么状态。

```text
环境供应
维护者配置 → environment builder → OCI Registry
                                      ├── development image
                                      └── runtime image

日常构建
development image + 源码快照 → workspace builder → Build Output → 开发机
                                  └── 可删除 cache

培训与体验
开发机 → K3s API → user namespace → virtual target
                                      └── 隔离 runtime 与 storage
```

资源有限时，以上角色可以部署在同一台 x86 服务器；扩容时可以分别迁移。物理共址不会
消除权限、cache 和生命周期边界。

## 四种服务角色

| 角色 | 输入与输出 | 持久状态 | 访问者 |
|---|---|---|---|
| Environment builder | RM Relay 配置 → development/runtime image | BuildKit cache | 维护者或 CI |
| OCI Registry | 接收并分发固定环境镜像 | OCI image 与 metadata | builder、target、授权用户 |
| Workspace builder | Development image + 源码快照 → Build Output 回开发机 | BuildKit、ccache、依赖 cache | 一般开发者 |
| K3s virtual target | Build Output + credential → 隔离应用 runtime | namespace runtime、quota、storage | 一般开发者、运维人员 |

### Environment builder 生产环境

它消费 RM Relay 的 Dockerfile、mise 能力配置和 Bake target，不编译用户每天修改的
workspace。跨架构镜像生产所需的 BuildKit builder、QEMU、cache 和发布流程都属于这里。
官方自动构建与战队自行构建必须消费同一份 Dockerfile、Bake target 和验证规则；两条路径只
在触发者、运行位置、cache 与推送凭据上不同。

Environment builder 是一个执行角色，不要求部署新的 RM Relay daemon。当前
`environment:embedded:publish` 接收现成 Buildx Builder、带版本的 OCI tag 和仓库外 handoff
路径，完成 Bake check、双架构构建、镜像内 smoke、push 与 manifest 核验。GitHub Actions、
战队 CI 或人工操作只负责触发并注入这些输入。

### Registry 只保存环境镜像

Registry 不保存用户源码、普通 Build Output 或 target 数据。RM Relay 只依赖标准 OCI
push/pull 与 digest，不绑定具体产品。托管服务、自部署实现及其运维流程留待后续决议；无论
选择哪条路径，都不能改变 Builder 的 environment ID 到 immutable digest 映射。

### Workspace builder 消费环境

它在固定 development image 中运行 CMake、colcon、测试或交叉编译，并把 Build Output
写回客户端。临时 workspace 不能成为源码真相源，服务端只保留可删除 cache。

Environment builder 与 workspace builder 可以部署在同一台机器，但入口、权限、cache 和
验证必须分开：前者生产并推送环境，后者只消费已经确定的 digest。Environment builder 需要
对应 Registry namespace 的写权限；workspace builder 只需要拉取权限。每个 workspace job 的
源码和 build tree 仍相互隔离，workspace ccache 也不属于 image-production cache。

### K3s 只管理 virtual target

K3s 不接管 Registry、workspace builder 或 physical target。每位用户或租户获得范围受限的
credential，并对应独立 namespace、RBAC、quota、storage 和 runtime 生命周期。

K3s API 已经提供编排控制面，因此首版不自研用户数据库、调度器或虚拟设备控制服务。
`rm-relay` 隐藏 Kubernetes 细节；战队运维负责升级、凭据、namespace、quota、storage 和
安全策略。

这种隔离只面向可信战队和受邀队伍。匿名不受信任代码需要更强的 sandbox、身份、配额和
合规设计，不在首版范围内。

## 服务不能绕过开发机控制链路

不同服务组合后，数据仍沿同一条开发闭环移动：

```text
开发机源码
  │
  ▼
local / remote workspace build
  │
  ▼
Build Output 返回开发机
  │
  ▼
客户端选择 physical / virtual target
  │
  ▼
客户端直连 shell / debugger，并接收 Managed Data
```

Workspace builder 不参与 target 的实时调试、ROS 数据或文件回传。K3s virtual target 也只
消费客户端持有的 Build Output，不接受 builder 私下直传。这样 build backend 和 target
provider 才能独立替换。

## 用户看到的是同一入口，不是同一基础设施

一般开发者始终使用 `rm-relay` CLI，只改变基础设施来源。宿主 mise 当前只服务 OpenOCD
一类宿主工具 adapter，不作为 `init`、build backend 或另一套用户任务入口：

| 来源 | 使用的基础设施 | 场景 |
|---|---|---|
| `local` | 本机受管 BuildKit、独立 cache 和宿主调试后端 | 已安装 Docker 的个人开发 |
| `team` | 战队 Registry、workspace builder、K3s virtual target | 队内长期使用与培训 |
| `invite` | 维护者临时提供的同类服务 | 友队试用和设计验证 |

Remote build 与 virtual target 可以组合成快速体验，但不会另建浏览器 IDE 或在线编译 API。
邀请制实例只服务本战队和受邀友队；公开注册与匿名多租户是后续方向。

## Compose 与 K3s 的使用边界

固定服务可以由 Compose 管理。只有 virtual target 需要多用户资源隔离和按需 runtime 时，
才使用 K3s。采用 K3s 不要求把 Registry 和 workspace builder 一并迁入 Kubernetes。

RM Relay 提供可复现配置、profile 和验证方法；战队运维负责机器、网络、账号、容量、备份
和服务可用性。首版复用现有控制面，不增加统一 `rm-relay-server`，也不自研 Web 管理界面、
通用任务队列、文件同步协议或网络 overlay。

## 仍待部署验证的内容

Workspace builder 已确定单节点 rootless BuildKit、mTLS、逻辑 Builder catalog 与不可变
environment 映射，并提供[部署说明](../operator-guide/deploy-buildkit-service.md)。本地 Builder
使用相同 workspace frontend，但由 CLI 管理独立的 Buildx `docker-container` resource。
真实服务器上的并发、磁盘配额和长期 cache 参数仍待验证；真实 environment push、Registry
实现、官方 CI adapter、K3s storage 与 credential 发放方式也尚未确定。本页继续只说明拓扑，
不代替可执行安装说明。
