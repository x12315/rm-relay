# 服务拓扑

RM Relay 依赖服务器，但“一台服务器”不是一个产品组件。同一台机器可以承载多个职责不同
的容器；资源充足后，这些职责也可以迁移到不同节点。服务边界由输入、权限和生命周期决定，
不由物理机器数量决定。

```text
环境供应
维护者配置 → 环境镜像构建器 → OCI Registry
                               ├── development image
                               └── runtime image

日常开发
development image ─┐
                   ▼
开发者 CLI → workspace 构建器 → Build Output → 返回本地
local/team/invite                                  │
                                                   ▼
                                            target adapter
                                            ├──物理设备
                                            └──虚拟执行器
                                                   ▲
                                                   │
                                             runtime image
```

图中的箭头表示职责与数据流，不要求所有服务常驻，也不要求它们部署在同一台机器。

## 服务角色

### 环境镜像构建器

它消费 RM Relay 的 Dockerfile、mise 能力配置和 Bake target，生产固定版本的官方镜像。
这项服务由维护者或 CI 使用，不向普通开发者暴露，也不编译用户每天修改的 workspace。

跨架构镜像生产所需的 BuildKit builder、QEMU、cache 和签名发布都属于这一角色。

### OCI Registry

Registry 保存 development、runtime 和经战队审查的派生镜像。Registry 不保存用户源码、
普通 Build Output 或 Session 数据。国内部署优先承载大体积镜像流量，少量源码仍可从上游
获取。

RM Relay 只依赖标准 OCI 能力，不绑定某个商业平台。Registry 的部署、备份和访问策略在
功能实现后进入 operator guide。

### workspace 构建器

workspace 构建器取得固定 development image 和用户源码快照，运行 CMake、colcon、测试
及交叉编译，然后把 Build Output 返回客户端。服务端可以保留 BuildKit cache、ccache 和
依赖 cache，不能把远程 workspace 当作源码真相源。

镜像构建器和 workspace 构建器可以共享 BuildKit 基础设施，但它们不是同一项服务：前者
生产环境，后者消费环境。两者应拥有不同入口、权限、cache namespace 和验证方法。

### 虚拟执行器

虚拟执行器按需创建 Linux virtual target，用于培训、快速体验和没有真实设备时的链路
验证。它消费与物理 Linux target 相同的本地 Build Output 和 runtime profile，不接受构建
服务器私下直传产物。

首版虚拟 target 只承诺应用 runtime 层；需要完整模拟目标宿主机时，后续可以增加 VM 类型
的 provider。虚拟执行器不要求默认采用 Docker-in-Docker，也不应把宿主 Docker socket
直接交给用户任务。

## 三种接入方式

一般开发者使用同一套 mise 任务，只切换 backend：

```text
local
开发者电脑上的 Docker 与宿主调试后端

team
战队内部的 Registry、workspace 构建器和可选虚拟执行器

invite
项目维护者临时提供给受邀战队的同类服务
```

“快速体验”是这些组件串联后的使用结果，不是一项独立的 Web 服务。它通过项目标准任务
入口发起，不建设浏览器 IDE，也不另写一套在线编译 API。

当前远程试用采用邀请制：独立 SSH 公钥、SSH tunnel 和明确的服务账号。它只服务本战队
与受邀友队，不开放注册，也不把内部 BuildKit 裸露给互联网。公网多租户需要身份、配额、
任务隔离和合规设计，列入后续可选模块。

## 客户端是控制端

远程构建不会改变开发者电脑的角色：

```text
本地源码
  ↓
远程 workspace 构建
  ↓
Build Output 返回本地
  ↓
客户端选择 target 并传输
  ↓
客户端直连 target shell / debugger
```

构建服务器不处在 target 的实时调试、ROS 数据或 Session 数据回传链路中。特殊网络可以
使用 SSH jump host，但这只是连接方式，不改变组件职责。

## 首版不建设控制平台

服务端需要由 RM Relay 提供可复现配置、固定镜像、cache 目录、资源限制、启动方法和
smoke 验证；不因此自研以下组件：

- Web 管理界面；
- 用户数据库；
- 通用调度器；
- 自定义文件同步协议；
- 浏览器开发环境；
- 常驻于每个项目的复杂 agent；
- 与成熟工具重复的构建 daemon。

mise 负责用户入口，BuildKit 负责构建平面，Compose 管理 Session runtime，OpenSSH 提供
连接。Dev Containers 保留为 IDE 与第三方远程开发工具的兼容标准，不成为另一套构建事实
源。

## 尚未定案

workspace 构建器的具体部署文件、并发模型、磁盘配额、BuildKit 安全参数和服务发现方式尚未
确定。当前文档只固定服务职责和数据流，operator guide 应在实现并验证后再给出部署命令。
