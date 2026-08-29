# 服务拓扑

本文解释 RM Relay 的环境供应、远程构建和虚拟 target 如何部署。它面向需要判断“某项
服务应该放在哪里、保存什么、由谁访问”的实现者和战队运维人员，不是现成部署手册。

> [!IMPORTANT]
> OCI Registry、workspace builder 和 K3s virtual target 尚未作为 RM Relay 产品交付。
> 本页记录服务边界；实际部署命令要等配置经过验证后写入 operator guide。

## 先按责任划角色，再决定机器数量

RM Relay 不把“一台服务器”当作一个组件。服务角色由四个问题划分：消费什么输入、产生
什么输出、谁有权限访问、必须保存什么状态。

```text
环境供应
维护者配置 → environment builder → OCI Registry
                                      ├── development image
                                      └── runtime image

日常构建
development image + 源码快照 → workspace builder → Build Output → 开发者电脑
                                  └── 可删除 cache

培训与体验
开发者电脑 → K3s API → user namespace → virtual target
                                      └── 隔离 runtime 与 storage
```

资源有限时，以上角色可以部署在同一台 x86 服务器；扩容时可以分别迁移。物理共址不会
消除权限、cache 和生命周期边界。

## 四种服务角色

| 角色 | 输入与输出 | 持久状态 | 访问者 |
|---|---|---|---|
| Environment builder | RM Relay 配置 → development/runtime image | BuildKit cache | 维护者或 CI |
| OCI Registry | 接收并分发固定环境镜像 | OCI image 与 metadata | builder、target、授权用户 |
| Workspace builder | Development image + 源码快照 → Build Output 回本地 | BuildKit、ccache、依赖 cache | 一般开发者 |
| K3s virtual target | Build Output + credential → 隔离应用 runtime | namespace runtime、quota、storage | 一般开发者、运维人员 |

### Environment builder 生产环境

它消费 RM Relay 的 Dockerfile、mise 能力配置和 Bake target，不编译用户每天修改的
workspace。跨架构镜像生产所需的 BuildKit builder、QEMU、cache 和发布流程都属于这里。

### Registry 只保存环境镜像

Registry 不保存用户源码、普通 Build Output 或 target 数据。RM Relay 只依赖标准 OCI
能力，不绑定商业平台；国内 Registry 实例优先承载大体积镜像流量。

### Workspace builder 消费环境

它在固定 development image 中运行 CMake、colcon、测试或交叉编译，并把 Build Output
写回客户端。临时 workspace 不能成为源码真相源，服务端只保留可删除 cache。

Environment builder 与 workspace builder 可以共享底层 BuildKit，但入口、权限、cache 和
验证必须分开：前者生产环境，后者消费环境。在可信战队或邀请制实例中，相同工具链可以
共享 ccache；每个 job 的 workspace 和 build tree 仍相互隔离。

### K3s 只管理 virtual target

K3s 不接管 Registry、workspace builder 或 physical target。每位用户或租户获得范围受限的
credential，并对应独立 namespace、RBAC、quota、storage 和 runtime 生命周期。

K3s API 已经提供编排控制面，因此首版不自研用户数据库、调度器或虚拟设备控制服务。
`rm-relay` 隐藏 Kubernetes 细节；战队运维负责升级、凭据、namespace、quota、storage 和
安全策略。

这种隔离只面向可信战队和受邀队伍。匿名不受信任代码需要更强的 sandbox、身份、配额和
合规设计，不在首版范围内。

## 服务不能绕过本地控制链路

不同服务组合后，数据仍沿同一条开发闭环移动：

```text
本地源码
  │
  ▼
local / remote workspace build
  │
  ▼
Build Output 返回本地
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

一般开发者使用同一套 mise task 和 `rm-relay` CLI，只改变基础设施来源：

| 来源 | 使用的基础设施 | 场景 |
|---|---|---|
| `local` | 本机 Docker、cache 和宿主调试后端 | 熟悉 Docker 的个人开发 |
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

Workspace builder 的配置格式、并发与磁盘配额、BuildKit 安全参数，以及 K3s 的 storage
和 credential 发放方式尚未确定。这些内容经过真实部署验证后，才会进入 operator guide；
在此之前不能把本页当作可执行安装说明。
