# 开发契约参考

> [!IMPORTANT]
> 本页记录已经确认的概念和不变量。尚未实现的 schema、CLI 和 adapter 不在这里预先命名。
> 当前支持状态见[支持矩阵](../user-guide/support-matrix.md)。

## 核心概念

| 概念 | 定义 | 默认生命周期 | 真相源或所有者 |
|---|---|---|---|
| Project | 用户的一项逻辑软件项目，可以包含多个 package 或仓库 | 数月或一个赛季 | 本地源码与项目声明 |
| Profile | 一组经过验证的环境、架构和 target 兼容要求 | 随 RM Relay 版本发布 | RM Relay Bake 与 profile 配置 |
| Build Job | 一次本地或远程构建操作 | 秒到分钟 | 构建 backend |
| Build Output | 可交给烧录、传输或 debugger 的构建结果 | 由项目本地保存 | 本地 `install` 目录 |
| Target | 能够接收 Build Output 并提供开发能力的物理或虚拟设备 | 独立于项目 | target provider |
| Runtime | Linux target 中承载用户程序的受控运行环境 | 当前 Session | Session/provider |
| Session | 一次连续、排他占用 target 的开发时段 | 通常数小时 | Session 状态与 target lease |
| Pending Data | Session 已释放后仍等待取回的数据 | 短期 | target provider，取回后回到本地 |

Build Output 的具体形式：

- 普通 CMake 项目：CMake Install Tree；
- ROS 2 workspace：colcon Install Space；
- MCU：ELF、BIN、MAP 等固件输出。

build tree 和 cache 不属于 Build Output。

## 身份

### `project_id`

- 标识逻辑项目，跨目录移动、项目改名和普通 clone 保持稳定；
- 在项目初始化时生成并随项目声明保存；
- 模板不得携带一个供所有新项目复用的固定值；
- 将现有项目复制为另一个独立项目时必须重新生成；
- 用于当前 Session 内识别项目工作区和增量传输；
- 不是用户身份、访问凭据或 cache 完整性证明。

### `session_id`

- 每次取得 target 时生成；
- 隔离 Session runtime、临时目录、锁和待取回数据；
- Session 进入 `Closed` 后不得复用。

### `target_id`

- 标识一项稳定的物理或虚拟 target；
- 与设备地址、临时容器名和用户本地别名分开；
- 同一时刻只能由一个 Active Session 持有。

当前不要求使用 `artifact_id` 完成增量传输。构建结果 manifest 和内容摘要可以后续加入，
但不改变 Project、Session 与 Target 的身份语义。

## 构建与路径

默认项目目录约定：

```text
<project-root>/
├── build/<profile>/
├── install/<profile>/
└── .rm-relay/data/<target_id>/<session_id>/
```

- `build/` 是可删除的本地中间目录；远程 backend 使用自己的 job workspace。
- `install/` 保存本地和远程 backend 共同产生的 Build Output。
- `.rm-relay/data/` 保存从 target 取回的数据，默认不进入 Git。

构建 cache 由 backend 管理，不进入这些项目目录的兼容性契约：

```text
BuildKit cache
ccache
依赖下载 cache
```

清空、迁移或切换 cache 不得改变构建输出语义。

## Session 数据映射

```text
Session 内
/workspace/data
RM_RELAY_DATA_DIR=/workspace/data

target provider 内部
<relay-state-root>/pending-data/<project_id>/<session_id>/

取回到本地
<project-root>/.rm-relay/data/<target_id>/<session_id>/
```

只有 `/workspace/data` 和 `RM_RELAY_DATA_DIR` 是用户可依赖的 target 路径。target 宿主机的
实际存储根目录是 provider 实现细节。

## Session 状态

| 状态 | Target lease | Runtime | 项目工作区 | Pending Data |
|---|---:|---:|---:|---:|
| `Active` | 持有 | 可以存在 | 存在 | 可以写入 |
| `Recoverable` | 已释放 | 已停止或删除 | 已删除 | 等待取回或放弃 |
| `Closed` | 无 | 无 | 无 | 无 |

稳定迁移：

```text
Closed → Active
Active → Closed
Active → Recoverable
Recoverable → Closed
```

终端断开不是状态迁移。`Opening`、`Closing` 和同步中的内部步骤也不成为长期公开状态。

Session 允许：

- 多次 build；
- 多次 deploy；
- CLI 断开和重新连接；
- 用户自行运行、停止或调试多个进程。

Session 不表示一个进程、ROS node、package、launch 文件或 Git 仓库。

## 不变量

### 本地是真相源

- 源码和项目声明以开发者本地工作区或其 Git 仓库为准；
- 远程构建服务只处理源码快照，不长期托管 workspace；
- 远程构建结果必须先回到本地，再进入 target；
- target 产生的受管开发数据最终回到本地。

### Target 保持干净

- Session 关闭后删除源码、Build Output、runtime writable layer、临时目录和项目依赖；
- 不保留跨 Session 的项目 cache，新 Session 默认完整传输；
- 目标宿主系统不得因某个项目被临时安装依赖；
- 唯一允许跨 Session 存在的项目相关内容是尚未取回的 Pending Data。

### 环境保持可复现

- 官方 profile 由 Dockerfile、mise 能力配置和 Bake 固定；
- 用户 overlay 必须在派生镜像构建阶段生效；
- 运行中的 development 或 runtime container 不得临时改变环境；
- 算力侧 development 与 runtime 必须共享 target environment lineage；
- 用户应用不打进官方 runtime image。

### 调试不经过构建服务器

- debugger 默认从开发者电脑直连 target；
- 构建服务器只生成并返回带匹配 symbols 的 Build Output；
- 服务器构建路径统一映射为稳定逻辑路径，供本地 IDE 对应源码；
- target 的 ROS 数据、日志和调试文件不因远程构建而绕道构建服务器。

### 平台不接管应用启动

- 平台提供 target 环境、交互式 CLI 和调试连接；
- 用户自行执行普通程序、脚本、`ros2 run` 或 `ros2 launch`；
- 平台不解析应用进程结构，也不提供通用应用 `run/stop` 状态机；
- Session 清理仍必须结束受其执行边界约束的残留进程。

## 兼容性契约

算力侧 profile 至少需要表达以下事实，具体 schema 尚未确定：

- target architecture 与 operating system；
- target package 与 runtime lineage；
- glibc、libstdc++、ROS 2、Python 和厂商 runtime 的兼容范围；
- 必需的 host kernel、driver、device、permission 和 network 条件；
- 已验证的 build、runtime smoke 与 debug 层级。

嵌入式 profile 则描述 MCU、ABI、linker/startup、烧录后端和调试后端。不能用 Linux
runtime 的目录与容器语义约束 MCU。

验证报告继续区分 `configured`、`detected`、`cross-compiled`、`flashed`、
`boot-observed` 和 `debug-tested`；缺少真实 target 证据时不得升级支持结论。
