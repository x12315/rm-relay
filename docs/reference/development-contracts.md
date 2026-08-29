# 开发契约参考

本文是跨组件术语和不变量的查阅页。遇到“这个对象归谁管理”“某个状态能否推出另一结论”
或“实现是否破坏架构边界”时，按标题定位；第一次了解项目，请先线性阅读
[开发平台架构](../architecture/README.md)。

> [!IMPORTANT]
> 本页记录已经确认的外部契约，不预先命名尚未实现的 schema、CLI、协议或 target 内部
> 目录。当前支持状态见[支持矩阵](../user-guide/support-matrix.md)。

## 术语索引

| 概念 | 准确定义 | 真相源或管理者 |
|---|---|---|
| Project | 用户的一项逻辑软件项目，可以包含多个 package 或仓库 | 开发机上的源码与项目声明 |
| Profile | 一组经过验证的环境、架构和 target 兼容要求 | RM Relay Bake 与 profile 配置 |
| Build Job | 一次 local 或 remote build 操作 | build backend |
| Build Output | 可以交给烧录、传输或 debugger 的构建结果 | 开发机上的项目工作区 |
| Target | 接收 Build Output 并提供开发能力的物理或虚拟设备 | provider；MCU 由开发机 adapter 直管 |
| Target adapter | 开发机侧能力接口，把 Build Output 与交互请求转换为 target 操作 | `rm-relay` |
| Target provider | 管理 Target/Target Environment 生命周期、内部状态和资源的目标侧实现 | `rm-relay-node` 或 virtual target provider；MCU 无独立 provider daemon |
| Target Environment | Linux target 中承载用户程序的受控环境 | `rm-relay-node` 或 virtual target provider |
| Managed Data | 用户放入受管目录、需要从 target 返回开发机的开发数据 | 取回后由开发机保管；此前由 target 暂存 |

本文的 **RM Relay Profile** 指平台层的验证组合。用户项目中的
`cmake/target-profiles/` 只描述 MCU 架构与 ABI，是 CMake 配置目录，不是另一类 RM Relay
Profile。

Build Output 沿用现有构建系统的输出形式：

| 项目类型 | Build Output |
|---|---|
| 普通 CMake 项目 | CMake Install Tree |
| ROS 2 workspace | colcon Install Space |
| MCU | ELF、BIN、MAP 等固件文件 |

Build tree、compiler cache 和依赖 cache 都不属于 Build Output。当前 MCU 模板已将固件导出
到 `install/<profile>`，并以 `rm-relay-output.json` 记录 Project、Profile、image 与内容
校验信息。

## 身份与兼容性

| 对象 | 必须满足的契约 | 不能用作替代的值 |
|---|---|---|
| Target | 具有稳定身份 | IP、临时容器名、用户本地别名 |
| Environment | 用版本或内容摘要确认 development image、Build Output 与 Target Environment 属于兼容 lineage | 可变 tag 或“容器能启动” |
| Project | 具有稳定身份 | 绝对路径、目录名、Git remote |
| 基础设施身份 | 用户、战队、K3s namespace 与 remote build 权限相互分开 | Project 或 Target 身份 |
| Cache | 只参与性能优化 | 身份、权限或构建完整性证明 |

Project identity 用于关联项目声明、build tree 和增量传输，不得充当用户身份或访问凭据。
模板保留空 ID，`rm-relay init` 为复制后的项目生成 UUID v4；不能把模板 ID、路径或 Git
remote 当作项目身份。Linux Target manifest、兼容字段与握手协议仍需等对应组件设计完成。

## 开发机路径

统一构建链路使用三类相互独立的目录：

```text
<project-root>/
├── build/<profile>/       开发机上可删除的构建中间目录
├── install/<profile>/     开发机可见的 Build Output
└── .rm-relay/data/        从 target 取回的 Managed Data
```

统一 CLI 已在 MCU 模板采用这一路径。直接调用 CMake 时仍可查看 `build/.../firmware/`，但
target adapter 只消费经过 manifest 校验的 `install/<profile>`。

Remote job workspace、BuildKit cache、ccache 和依赖下载 cache 由 backend 管理，不进入
项目目录契约。开发机 cache 与编译服务器 cache 互不复制；删除或切换 cache 不得改变构建语义。

物理 target 的宿主目录、container mount 和虚拟 target 的 storage 布局由各 provider 管理。
客户端只能依赖公开的输入与数据入口，不能依赖内部路径。具体目录需要等组件实现和迁移
策略确定后再写入 reference。

## 生命周期

| 对象 | 创建与保留 | 结束或重建条件 |
|---|---|---|
| Build Job | 只处理当前源码快照；server-side cache 可跨 job 保留 | Build Output 返回开发机后，remote workspace 可删除 |
| 物理 Target Environment | 长期可用，唯一项目资产位于受管挂载或开发机 | 不存在时创建，环境版本不匹配时重建；断线、构建结束或程序退出不销毁 |
| Mutagen session | 保存物理 target 的连接与同步状态 | 不代表 Project、用户登录、程序运行或 container 生命周期 |
| 虚拟 Target Environment | runtime 与 storage 位于用户/租户的 K3s namespace | provider 按 credential、RBAC、quota 管理；回收不改变开发机资产所有权 |
| MCU Target | 按 board profile 提供 flash、reset、serial、debug | 没有 Linux Target Environment、Mutagen session 或受管 container |

同一物理 target 输入端同一时间只允许一个写入者。RM Relay 不维护与 Mutagen 平行的
Development Session 数据库。

## 跨组件不变量

### 开发机是真相源

- 源码和项目声明以开发机工作区或 Git 为准。
- Remote build 只处理源码快照，不长期托管 workspace。
- Remote Build Output 必须先回到开发机，再进入 target。
- Managed Data 最终回到开发机；取回失败时可由 target 在受管目录短期暂存。

### Target 可恢复

- 项目依赖不能安装进 target 宿主系统。
- Target Environment 由固定 image 创建，并能按已知版本重建。
- Target 上的 Build Output 可覆盖或清理，不能成为唯一项目资产。
- Container writable layer、临时文件和内部 cache 不能成为唯一项目资产。
- 除尚未取回的 Managed Data 外，物理 target 不承诺保存跨环境重建的项目状态。
- Managed Data 的容量、保留期限和自动回收由后续持久数据模块定义。

### 环境可复现

- 官方 profile 由 Dockerfile、mise 能力配置和 Bake 固定。
- 项目 overlay 必须在派生镜像构建阶段生效。
- 运行中的 development/runtime container 不得安装依赖改变正式环境。
- Linux development 与 runtime image 必须共享 Target Environment lineage。
- 用户应用不得进入官方 runtime image。

### 调试不经过 workspace builder

- Debugger 默认从开发机直连 target。
- Workspace builder 只生成并返回带匹配 symbols 的 Build Output。
- Remote build 使用稳定逻辑路径，供开发机 IDE 映射源码。
- ROS 数据、日志和调试文件不因 remote build 而绕道 workspace builder。

### 平台不接管应用启动

- 平台提供 Target Environment、交互式 CLI、调试连接和受管数据入口。
- 用户自行执行普通程序、脚本、`ros2 run` 或 `ros2 launch`。
- 平台不解析应用进程结构，也不提供通用 `run/stop` 状态机。
- 取回数据前由用户停止相关程序；平台保证操作顺序，应用保证文件完整性。

## Profile 最低兼容信息

算力侧 profile 至少要表达以下事实；具体 schema 尚未确定：

- target architecture 与 operating system；
- target package 与 runtime lineage；
- glibc、libstdc++、ROS 2、Python 和 vendor runtime 的兼容范围；
- 必需的 host kernel、driver、device、permission 和 network 条件；
- 已验证的 build、runtime smoke 与 debug 层级。

嵌入式 profile 描述 MCU、ABI、linker/startup、烧录后端和调试后端。Linux target 的目录、
container 和 daemon 契约不适用于 MCU。

## 证据等级

验证结论分属两个证据域，只能记录实际达到的状态。

构建状态：

| 状态 | 证明的事实 |
|---|---|
| `built` | 某个构建命令成功 |
| `host-tested` | Native 测试在宿主架构容器中通过 |
| `cross-compiled` | 目标产物已生成并通过规定的静态检查 |

硬件后端状态：

| 状态 | 证明的事实 |
|---|---|
| `configured` | 配置和命令入口存在 |
| `detected` | 工具已只读枚举目标设备 |
| `flashed` | 固件已成功写入目标芯片 |
| `flash-verified` | 写入内容已由工具校验或回读比对 |
| `boot-observed` | 复位后观察到约定启动行为 |
| `debug-tested` | 调试器已连接、命中观察点并核对约定状态 |

两组状态不可互相替代，也不能把较弱证据升级为更强结论：配置解析不等于设备已连接，
`cross-compiled` 不等于已在目标运行，`flash-verified` 也不等于应用启动或源码调试成功。
完整定义与当前结果见
[STM32 固件构建](../user-guide/build-stm32.md#验证状态)和
[支持矩阵](../user-guide/support-matrix.md)。
