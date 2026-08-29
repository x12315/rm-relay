# Target 接入与数据链路

> [!IMPORTANT]
> 当前仓库只实现 MCU 的部分烧录与调试路径。物理 Linux、虚拟 Linux 和统一的 target
> adapter 仍是设计基线；实际支持状态以[支持矩阵](../user-guide/support-matrix.md)为准。

Build Output 返回本地后，`rm-relay` 通过对应的 target adapter 完成传输、交互和调试。
三类 target 共享能力语义，但各自使用符合设备特征的实现：

| Target | 接入实现 | 主要能力 |
|---|---|---|
| 物理 Linux | `rm-relay-node`、Docker、Mutagen | shell、文件传输、debug、Managed Data 回收 |
| 虚拟 Linux | K3s namespace 与 virtual target provider | shell、文件传输、应用 runtime、Managed Data 回收 |
| 物理 MCU | OpenOCD、DFU、serial、GDB 和兼容工具 | flash、reset、serial、debug |

Target 只声明已经具备的能力。MCU 不模拟 Linux container、文件系统或 daemon；Linux
runtime smoke 也不能代替真实宿主机、设备与驱动验证。

## 开发者电脑是控制端

```text
源码与构建请求        开发者电脑 → local / remote builder
Build Output           remote builder → 开发者电脑 → target
shell / debugger       开发者电脑 ↔ target
Managed Data           target → 开发者电脑
```

远程构建不会让 workspace builder 成为 debugger 或数据中转站。特殊网络可以使用独立的
jump host，但不改变组件责任。

Linux 调试可以使用 target 中的 GDB、gdbserver 或 debugpy。IDE 保留本地源码，并根据稳定
构建路径映射 symbols。ROS CLI、日志、trace、bag 和可视化由用户在应用层选择；RM Relay
提供 Target Environment、终端、调试连接和受管数据路径。

## 物理 Linux target

首版物理 target 面向可以通过 SSH 和 `sudo` 初始化的 Debian/Ubuntu LTS，覆盖 `amd64` 与
`arm64`。`rm-relay-node` 以原生 `.deb` 安装在宿主机，负责：

- 管理目标环境镜像、容器和挂载；
- 检查环境版本或内容摘要；
- 维护 RM Relay 的宿主侧目录与基础配置；
- 报告设备能力和连接信息。

普通客户端通过 `rm-relay-node` 操作 target，不直接访问目标机的 Docker socket；容器细节
由 node 封装。首次登记可以复用已有 SSH 权限安装对应架构的软件包；具体命令、认证方式和
通信协议留给组件设计。

Target Environment 是长期可用、可重建的开发容器。它在不存在时创建，在环境版本变化时
重建，普通断线不会销毁。容器根文件系统保持只读；Build Output 和 Managed Data 位于
宿主机受管挂载中。用户进入 shell 后自行启动、停止或调试应用，`rm-relay-node` 不解析
应用进程结构。

## Build Output 与 Managed Data

Mutagen 是物理 Linux target 的首个同步实现：

```text
本地 Build Output ── Mutagen ──→ target 输入目录
本地数据目录       ←─ Mutagen ─── target 数据目录
```

连接与同步状态由 Mutagen session 保存。RM Relay 提供受约束的默认操作流程，不再维护平行
的 Development Session 数据库，也不把容器生命周期绑定到终端连接。熟悉 Mutagen 的用户
仍可检查和管理底层 session。

同一 target 输入端同一时间只允许一个写入者。取回数据前，用户先停止相关程序，再由平台
执行同步。平台保证操作顺序；应用负责确认文件已经写完并且可以安全读取。

未及时取回的数据可以暂存在目标宿主机的受管目录。除此之外，target 不保存唯一项目资产。
容量、保留期限、自动回收和人工管理界面属于后续持久数据管理模块。

## 连接与发现

物理 Linux target 的黄金路径是有线直连和同一局域网 Wi-Fi。战队或设备维护者预先配置
Wi-Fi 名称、密码和基础网络；RM Relay 不提供 Tailscale 一类 overlay network。

完成首次信任后，target 可以通过 mDNS/Avahi 广播稳定身份和能力，客户端因此能在 IP 地址
变化后重新发现设备。公网反向隧道、复杂跨网段发现和网络准入不属于首版 target 契约。

## 虚拟 Linux target

虚拟 target 用于培训、快速体验和没有物理设备时的链路验证。K3s 为每位用户或租户提供
独立 namespace、RBAC、quota、storage 和 runtime 生命周期；一般用户通过 `rm-relay`
选择 target，不直接操作 Kubernetes。

虚拟 target 与物理 target 共享 Build Output、交互和数据返回的外部语义，不复制
物理宿主机拓扑，也不默认使用 Docker-in-Docker。它只承诺应用 runtime 层；kernel、驱动和
完整操作系统行为需要真实设备或后续的 VM provider 验证。

## MCU target

MCU 沿用原生调试链路。本地 adapter 把固件交给 OpenOCD、ROM DFU、serial bootloader 或
厂商兼容工具，再由 GDB 连接调试后端。USB 设备位于开发者电脑还是远程调试主机，由 host
profile 决定。

MCU 不安装 `rm-relay-node`，不运行 Mutagen，也没有 Linux Target Environment。Board
profile 直接声明可用的 flash、reset、serial 和 debug 后端。

## 后续组件设计

Target adapter 接口、`rm-relay-node` 通信协议、Mutagen 同步模式、目标宿主目录和 K3s
内部传输方式尚未确定。后续实现必须保持本页的外部边界：本地是控制端，物理 target
环境可重建，物理 Linux 传输状态由 Mutagen 管理，虚拟 target 的多用户隔离由 K3s
管理，应用启动方式由用户项目决定。
