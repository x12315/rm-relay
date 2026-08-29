# 服务拓扑

> [!IMPORTANT]
> Registry、workspace builder 和 K3s virtual target 尚未作为 RM Relay 产品交付。本页记录
> 已确认的服务边界，不能作为现有部署说明。

RM Relay 以服务职责划分拓扑，不把“一台服务器”当作组件。资源有限时，同一台 x86 机器
可以承载多项服务；需要扩容时，各角色可以迁移到不同节点。输入、权限、持久状态和生命周期
决定服务边界。

```text
环境供应
维护者配置 → 环境镜像构建器 → OCI Registry
                                        ├── development image
                                        └── runtime image

日常构建
development image ─┐
                   ▼
开发者 CLI → workspace builder → Build Output → 开发者电脑
                   │
                   └── server-side cache

培训与体验
开发者 CLI → K3s API → user namespace → virtual target
                                            └── isolated storage
```

RM Relay 提供可复现的 Compose/K3s 配置、profile 和验证方法。战队运维人员负责机器、网络、
账号、容量、备份和服务可用性。现有基础设施直接承担控制面职责，首版不增加统一的
`rm-relay-server`。

## 服务角色

| 角色 | 输入与产物 | 持久状态 | 访问者 |
|---|---|---|---|
| 环境镜像构建器 | RM Relay 配置 → 固定版本的 development/runtime image | BuildKit cache | 维护者或 CI |
| OCI Registry | 保存 development、runtime 和经审查的派生镜像 | OCI image 与 metadata | 构建服务、target 和授权用户 |
| workspace builder | development image + 源码快照 → Build Output 返回客户端 | BuildKit cache、ccache 与依赖 cache | 一般开发者 |
| K3s virtual target | Build Output + 用户凭据 → 隔离的应用 runtime | namespace 中的 runtime、quota 与 storage | 一般开发者、战队运维 |

### 环境镜像构建器

该角色消费 RM Relay 的 Dockerfile、mise 能力配置和 Bake target，不编译用户日常修改的
workspace。跨架构镜像生产所需的 BuildKit builder、QEMU、cache 和发布流程属于这里。

### OCI Registry

Registry 保存环境镜像，不保存用户源码、普通 Build Output 或 target 数据。RM Relay 只依赖
标准 OCI 能力，不绑定商业平台；国内实例优先承载大体积镜像流量。

### workspace builder

workspace builder 在固定 development image 中运行 CMake、colcon、测试或交叉编译，并
通过 BuildKit 把 Build Output 返回客户端。服务端只保留可删除的 cache，临时 workspace
不能成为源码真相源。

环境镜像构建器与 workspace builder 可以共享 BuildKit 基础设施，但入口、权限、
cache 和验证必须分开。前者生产环境，后者消费环境。在可信的战队或邀请制实例中，同一
环境与工具链可以共享 ccache；job workspace 和 build tree 仍按任务隔离。

### K3s virtual target

K3s 只编排多用户 virtual target，不接管 Registry、workspace builder 或 physical target。
每位用户或租户取得范围受限的 Kubernetes credential，并对应独立 namespace：

```text
user identity
└── namespace
    ├── virtual target runtime
    ├── resource quota
    ├── isolated storage
    └── RBAC policy
```

K3s API 已经提供编排平面，因此首版不自研用户数据库、调度器或虚拟设备控制服务。
`rm-relay` 隐藏 Kubernetes 细节；战队运维人员负责 K3s 升级、凭据发放、namespace、quota、
storage 和安全策略。

Namespace 隔离只面向战队内部和受邀队伍。匿名不受信任代码需要更强的 sandbox、身份、配额
和合规设计，不在首版范围内。

## 基础设施来源

一般开发者使用同一套 mise task 和 `rm-relay` CLI，只切换基础设施来源：

| 方式 | 基础设施 | 适用场景 |
|---|---|---|
| local | 开发者电脑上的 Docker、cache 和宿主调试后端 | 熟悉 Docker 的个人开发 |
| team | 战队 Registry、workspace builder 与 K3s virtual target | 队内长期使用和培训 |
| invite | 维护者暂时提供的同类服务 | 友队试用和设计验证 |

远程构建与 virtual target 可以组合成快速体验，但不会形成另一套浏览器 IDE 或在线编译
API。邀请制实例只面向本战队和受邀友队；公开注册和匿名多租户属于后续方向。

## 客户端控制链路

```text
本地源码
  ↓
local / remote workspace build
  ↓
Build Output 返回本地
  ↓
客户端选择 target 并传输
  ↓
客户端直连 shell / debugger，接收 Managed Data
```

workspace builder 不参与 target 的实时调试、ROS 数据或文件回传。K3s virtual target 也只
消费客户端持有的 Build Output，不接受 builder 私下直传。构建 backend 和 target provider
因此可以独立替换。

## 部署边界

固定服务可以由 Compose 管理；需要多用户资源隔离和按需 runtime 时才使用 K3s。采用 K3s
不要求把 Registry 和 workspace builder 一并迁入 Kubernetes。

首版不自研 Web 管理界面、浏览器开发环境、通用任务队列、文件同步协议、网络 overlay、
公共 sandbox 或常驻于用户项目的复杂 agent。

workspace builder 的部署文件、并发与磁盘配额、BuildKit 安全参数，以及 K3s 的 storage
和凭据发放方式尚未确定。相应配置经过实际部署验证后，operator guide 才会提供操作命令。
