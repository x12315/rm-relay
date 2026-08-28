# Target 接入与数据链路

Build Output 回到本地后，由 `rm-relay` 连接对应的 target adapter。物理 Linux、虚拟
Linux 和 MCU 共享“接收构建结果并提供开发能力”的上层语义，但不强求相同的内部结构。

```text
本地 Build Output
        │
        ▼
rm-relay
├── physical Linux：rm-relay-node + Docker + Mutagen
├── virtual Linux：K3s virtual target
└── physical MCU：OpenOCD / DFU / serial + GDB
```

Target 需要声明自己实际具备的能力，例如 shell、文件传输、flash 或 debug。MCU 不会为了
接口统一而模拟 Linux container、文件系统或 daemon；普通 Linux 容器也只证明应用 runtime
层能够工作，不能代替真实宿主机与驱动验证。

## 开发者电脑是控制端

本地保存源码、Build Output 和取回的数据，并直接连接 target：

```text
源码与构建请求        开发者电脑 → 本地或远程 builder
Build Output           远程 builder → 开发者电脑 → target
shell / debugger       开发者电脑 ↔ target
开发数据               target → 开发者电脑
```

远程构建不会让构建服务器成为 debugger 或数据中转站。特殊网络可以使用独立的 jump host，
但这不改变组件职责。

Linux 调试可以使用 target 中的 GDB、gdbserver 或 debugpy。IDE 使用本地源码，并根据稳定
构建路径映射 symbols。ROS CLI、日志、trace、bag 和可视化仍由用户在应用层选择；RM Relay
只保证目标环境、终端、调试连接和受管数据路径。

## 物理 Linux target

首版物理 target 面向可通过 SSH 和 `sudo` 初始化的 Debian/Ubuntu LTS，覆盖 `amd64` 与
`arm64`。`rm-relay-node` 以原生 `.deb` 安装在宿主机，维护以下内容：

- 目标环境镜像、容器和挂载；
- 环境版本或内容摘要的匹配检查；
- RM Relay 的宿主侧目录和基础配置；
- 设备能力与连接信息。

客户端不直接操作目标机 Docker socket，也不把容器细节暴露给普通用户。首次登记可以由
客户端通过现有 SSH 权限安装对应架构的软件包；具体命令、认证和通信协议留给组件设计。

目标环境是长期可用、可重建的开发容器：不存在时创建，环境版本变化时重建，普通断线不会
销毁。容器根文件系统保持只读；Build Output 和开发数据位于宿主机受管挂载中，容器内的
临时变化不构成项目资产。用户进入 shell 后自行启动、停止或调试应用，`rm-relay-node`
不理解应用内部进程结构。

## 文件传输与开发数据

Mutagen 是物理 Linux target 的首个正式同步实现，负责开发者电脑与目标宿主机之间的增量
传输：

```text
本地 Build Output ── Mutagen ──→ target 输入目录
本地数据目录       ←─ Mutagen ─── target 数据目录
```

Mutagen 自身的 session 保存连接和同步状态。RM Relay 不再维护一套重复的 Development
Session 状态机，也不把容器生命周期绑定到一次终端连接。一般用户可以通过 `rm-relay` 使用
受约束的默认流程；熟悉 Mutagen 的用户仍可检查和管理底层 session。

同一 target 输入端同一时间只允许一个写入者。平台保证操作顺序：需要取回数据时，先由用户
停止相关程序，再完成同步。文件何时写完、是否能安全读取仍由用户程序保证，RM Relay 不尝试
理解任意文件格式或复制应用自己的事务协议。

未能及时取回的数据可以暂存在目标宿主机的受管目录；除这类待取回数据外，目标机不保存
唯一项目资产。容量、保留期限、自动回收和人工管理界面属于后续持久数据管理模块。

## 连接与发现

RM Relay 的黄金路径是有线直连和同一局域网 Wi-Fi。Wi-Fi 名称、密码和基础网络由战队或
设备维护者预先配置；RM Relay 不建设 Tailscale 一类 overlay network。

完成首次信任后，物理 target 可以通过 mDNS/Avahi 广播身份和能力，让客户端在地址变化后
重新发现设备。公网反向隧道、复杂跨网段发现和网络准入不属于首版目标接入契约。

## 虚拟 Linux target

虚拟 target 用于培训、快速体验和没有物理设备时的链路验证。多用户服务由 K3s 提供
namespace、RBAC、资源配额、存储和 runtime 生命周期；普通用户仍通过 `rm-relay` 选择
target，不直接操作 Kubernetes。

虚拟 target 与物理 target 共享 Build Output、交互和数据返回的外部语义，不复制物理宿主
机的内部拓扑，也不默认使用 Docker-in-Docker。它只承诺应用 runtime 层；需要验证 kernel、
驱动或完整操作系统时，应使用真实设备或以后引入的 VM provider。

## MCU target

MCU 保持原生调试链路。固件由本地 adapter 交给 OpenOCD、ROM DFU、串口 bootloader 或
厂商兼容工具，再由 GDB 连接对应调试后端。USB 设备位于开发者电脑还是远程调试主机，由
host profile 决定。

MCU 不安装 `rm-relay-node`，不运行 Mutagen，也没有 Linux 目标环境。上层只根据 board
profile 和能力选择烧录、复位、串口或源码调试方法。

## 尚未定案

Target adapter 接口、`rm-relay-node` 通信协议、Mutagen 的具体同步模式、目标宿主目录和
K3s 内部传输方式仍属于组件设计。实现时必须保持本页已经确认的边界：本地是控制端，物理
target 环境可重建，传输状态交给 Mutagen，多用户虚拟 target 交给 K3s，应用启动仍由用户
负责。
