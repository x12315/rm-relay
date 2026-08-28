# 服务拓扑

RM Relay 依赖服务器，但“一台服务器”不是一个产品组件。同一台 x86 机器可以承载多个
职责不同的服务；资源充足后，这些职责也可以迁移到不同节点。边界由输入、权限和生命周期
决定，不由物理机器数量决定。

```text
环境供应
维护者配置 → 环境镜像构建器 → OCI Registry
                               ├── development image
                               └── runtime image

日常构建
development image ─┐
                   ▼
开发者 CLI → workspace builder → Build Output → 返回本地
                   │
                   └── server-side cache

培训与体验
开发者 CLI → K3s API → user namespace → virtual target
                                            └── isolated storage
```

服务器上不设置首版自研的 `rm-relay-server`。RM Relay 提供可复现的 Compose、K3s 配置、
profile 和验证方法；战队运维人员负责机器、网络、账号、容量、备份和服务可用性。

## 服务角色

### 环境镜像构建器

它消费 RM Relay 的 Dockerfile、mise 能力配置和 Bake target，生产固定版本的官方镜像。
这项服务由维护者或 CI 使用，不向普通开发者开放，也不编译用户每天修改的 workspace。

跨架构镜像生产所需的 BuildKit builder、QEMU、cache 和镜像发布属于这一角色。

### OCI Registry

Registry 保存 development、runtime 和经战队审查的派生镜像，不保存用户源码、普通
Build Output 或 target 数据。RM Relay 只依赖标准 OCI 能力，不绑定商业平台；国内实例
优先承载大体积镜像流量，少量上游源码仍可按项目需要获取。

### workspace builder

workspace builder 取得固定 development image 和用户源码快照，运行 CMake、colcon、测试
或交叉编译，再由 BuildKit 把 Build Output 返回客户端。服务端可以长期保存 BuildKit
cache、ccache 和依赖 cache，不能把临时 workspace 当作源码真相源。

镜像构建器和 workspace builder 可以共享 BuildKit 基础设施，但不是同一项服务：前者生产
环境，后者消费环境。入口、权限、cache 与验证应分别配置。

在战队内部和邀请制实例中，同一环境与工具链可以共享 ccache；job workspace 和 build tree
仍按任务隔离。本地 cache 与服务器 cache 互不同步。

### K3s 虚拟 target 服务

K3s 只负责多用户虚拟 target，不接管 Registry、workspace builder 或物理设备。每位用户
或租户取得范围受限的 Kubernetes credential，并对应独立 namespace：

```text
user identity
└── namespace
    ├── virtual target runtime
    ├── resource quota
    ├── isolated storage
    └── RBAC policy
```

K3s API 本身就是这项服务的编排平面，因此首版不再开发用户数据库、调度器或虚拟设备控制
服务。`rm-relay` 隐藏 Kubernetes 细节，战队运维人员负责安装升级 K3s、发放凭据、设置
namespace、配额、存储和安全策略。

这种隔离适合战队内部或受邀队伍的培训环境，不足以承载匿名不受信任代码。公开服务需要
更强 sandbox、身份、配额和合规设计，属于后续模块。

## 三种接入方式

一般开发者使用同一套 mise task 和 `rm-relay` CLI，只切换基础设施来源：

| 方式 | 基础设施 | 适用场景 |
|---|---|---|
| local | 开发者电脑上的 Docker、cache 和宿主调试后端 | 熟悉 Docker 的个人开发 |
| team | 战队 Registry、workspace builder 与 K3s virtual target | 队内长期使用和培训 |
| invite | 维护者暂时提供的同类服务 | 友队试用和设计验证 |

“快速体验”是远程构建与虚拟 target 串联后的结果，不是浏览器 IDE 或另一套在线编译 API。
当前邀请制实例只服务本战队和受邀友队。公开注册和匿名多租户不属于首版。

## 客户端仍是控制端

```text
本地源码
  ↓
local / remote workspace build
  ↓
Build Output 返回本地
  ↓
客户端选择 target 并传输
  ↓
客户端直连 shell / debugger，接收开发数据
```

构建服务器不进入 target 的实时调试、ROS 数据或文件回传链路。K3s 虚拟 target 也消费
客户端持有的 Build Output，不接受 workspace builder 私下直传产物。这样，本地、远程构建
和不同 target 后端可以独立替换。

## 部署边界

固定服务可以由 Compose 管理；需要多用户资源隔离和按需 runtime 时才使用 K3s。引入 K3s
不意味着把整套服务器迁入 Kubernetes。

首版不自研以下设施：

- Web 管理界面和浏览器开发环境；
- 服务器用户数据库与通用任务队列；
- 文件同步协议和网络 overlay；
- 面向匿名用户的公共 sandbox；
- 常驻于每个用户项目的复杂 agent。

workspace builder 的部署文件、并发限制、磁盘配额、BuildKit 安全参数，以及 K3s 的存储和
凭据发放方式尚未定案。operator guide 只会在相应配置经过实际部署验证后给出命令。
