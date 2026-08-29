# 开发契约参考

> [!IMPORTANT]
> 本页只定义跨组件已经确认的概念和不变量，不预先命名尚未实现的 schema、CLI、协议或
> target 内部目录。当前支持状态见[支持矩阵](../user-guide/support-matrix.md)。

## 核心概念

| 概念 | 定义 | 真相源或管理者 |
|---|---|---|
| Project | 用户的一项逻辑软件项目，可以包含多个 package 或仓库 | 本地源码与项目声明 |
| Profile | 一组经过验证的环境、架构和 target 兼容要求 | RM Relay Bake 与 profile 配置 |
| Build Job | 一次本地或远程构建操作 | build backend |
| Build Output | 可交给烧录、传输或 debugger 的构建结果 | 本地项目工作区 |
| Target | 能接收 Build Output 并提供开发能力的物理或虚拟设备 | target adapter |
| Target Environment | Linux target 中承载用户程序的受控环境 | `rm-relay-node` 或 virtual target provider |
| Managed Data | 用户放入受管目录、需要从 target 返回本地的开发数据 | 本地工作区；未取回时由 target 暂存 |

Build Output 按项目类型采用现有构建系统的输出形式：

| 项目类型 | Build Output |
|---|---|
| 普通 CMake 项目 | CMake Install Tree |
| ROS 2 workspace | colcon Install Space |
| MCU | ELF、BIN、MAP 等固件文件 |

build tree、compiler cache 和依赖 cache 都不属于 Build Output。

## 身份与兼容性

| 对象 | 契约 |
|---|---|
| Target | 必须具有稳定身份，并且独立于 IP 地址、临时容器名和用户本地别名 |
| Environment | 必须用版本或内容摘要确认 development image、Build Output 和 Target Environment 属于兼容 lineage |
| Project | 必须具有稳定身份；不得从绝对路径、目录名或 Git remote 直接推导 |
| 基础设施身份 | 用户、战队、K3s namespace 和远程构建权限必须与 Project/Target 身份分开 |
| Cache | ccache 命中不得作为身份、权限或构建完整性证明 |

Project identity 用于关联项目声明、build tree 和增量传输，不得充当用户身份或访问凭据。
公开 schema 确定前，模板不得包含所有新项目共用的固定 ID。Target manifest、兼容性字段和
握手协议也要等组件设计完成后再进入本 reference。

## 本地路径

项目工作区保留三类相互独立的资产：

```text
<project-root>/
├── build/<profile>/       可删除的本地构建中间目录
├── install/<profile>/     本地可见的 Build Output
└── .rm-relay/data/        从 target 取回的 Managed Data
```

远程 job workspace、BuildKit cache、ccache 和依赖下载 cache 由 build backend 管理，不
进入项目目录契约。本地 cache 与远程 cache 互不复制；删除或切换 cache 不得改变构建语义。

物理 target 的宿主目录、容器 mount 和虚拟 target 的存储布局由各 provider 管理。客户端
只能依赖公开的输入与数据入口，不得依赖内部路径。具体路径需要等组件实现和迁移策略确定后
再写入 reference。

## 生命周期

| 对象 | 创建与保留 | 结束或重建条件 |
|---|---|---|
| Build Job | 只处理当前源码快照；server-side cache 可以跨 job 保留 | Build Output 返回本地后，remote workspace 可以删除 |
| 物理 Target Environment | 开发容器长期可用；唯一项目资产必须位于受管挂载或开发者本地 | 不存在时创建，环境版本不匹配时重建；终端断开、一次构建结束或程序退出都不会销毁 |
| Mutagen session | 保存物理 target 的连接与同步状态 | 不代表 Project、用户登录、程序运行或容器生命周期 |
| 虚拟 Target Environment | runtime 与存储位于用户或租户的 K3s namespace | 由 provider 按 credential、RBAC 和 quota 管理；回收不改变本地资产所有权 |
| MCU Target | 按 board profile 提供 flash、reset、serial 和 debug | 没有 Linux Target Environment、Mutagen session 或受管容器 |

同一物理 target 输入端同一时间只允许一个写入者。RM Relay 不维护平行的 Development
Session 数据库。

## 跨组件不变量

### 本地是真相源

- 源码和项目声明以开发者本地工作区或 Git 仓库为准。
- 远程构建只处理源码快照，不长期托管 workspace。
- 远程 Build Output 必须先回到本地，再进入 target。
- Managed Data 最终回到本地；取回失败时可以由 target 在受管目录短期暂存。

### Target 可恢复

- 项目依赖不能安装进目标宿主系统。
- Target Environment 由固定镜像创建，并能按已知版本重建。
- Target 上的 Build Output 可以覆盖或清理，不能成为唯一项目资产。
- 容器 writable layer、临时文件和内部 cache 不能成为唯一项目资产。
- 除尚未取回的 Managed Data 外，物理 target 不承诺保存跨环境重建的项目状态。
- Managed Data 的容量、保留期限和自动回收由后续持久数据管理模块定义。

### 环境可复现

- 官方 profile 由 Dockerfile、mise 能力配置和 Bake 固定。
- 项目 overlay 必须在派生镜像构建阶段生效。
- 运行中的 development/runtime container 不得安装依赖改变环境。
- 算力侧 development 与 runtime image 必须共享 target environment lineage。
- 用户应用不得进入官方 runtime image。

### 调试不经过 workspace builder

- debugger 默认从开发者电脑直连 target。
- workspace builder 只生成并返回带匹配 symbols 的 Build Output。
- 服务器使用稳定逻辑构建路径，供本地 IDE 映射源码。
- ROS 数据、日志和调试文件不因远程构建而绕道 workspace builder。

### 平台不接管应用启动

- 平台提供 Target Environment、交互式 CLI、调试连接和受管数据入口。
- 用户自行执行普通程序、脚本、`ros2 run` 或 `ros2 launch`。
- 平台不解析应用进程结构，也不提供通用应用 `run/stop` 状态机。
- 取回数据前由用户停止相关程序；平台保证操作顺序，应用保证文件完整性。

## Profile 兼容要求

算力侧 profile 至少必须表达以下事实；具体 schema 尚未确定：

- target architecture 与 operating system；
- target package 与 runtime lineage；
- glibc、libstdc++、ROS 2、Python 和厂商 runtime 的兼容范围；
- 必需的 host kernel、driver、device、permission 和 network 条件；
- 已验证的 build、runtime smoke 与 debug 层级。

嵌入式 profile 描述 MCU、ABI、linker/startup、烧录后端和调试后端。Linux target 的目录、
容器和 daemon 契约不适用于 MCU。

验证报告必须区分 `configured`、`detected`、`cross-compiled`、`flashed`、
`boot-observed` 和 `debug-tested`。缺少真实 target 证据时不得升级支持结论。
