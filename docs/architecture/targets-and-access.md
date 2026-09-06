# Target 接入与数据链路

本文从 Build Output 已经回到开发机的位置开始，说明它如何进入物理或虚拟设备，以及 shell、
debugger 和开发数据如何回到开发机。构建这一输出的过程见
[构建与输出](builds-and-outputs.md)。

> [!IMPORTANT]
> 当前仓库实现了 MCU OpenOCD adapter 与部分实板烧录、调试路径。物理 Linux、虚拟 Linux
> 及 `rm-relay-node` 仍是设计基线；真实支持状态见
> [支持矩阵](../user-guide/support-matrix.md)。

## Target 从开发机 Build Output 接手

```text
Build Output（开发机）
        │
        ▼
target adapter
        │
        ├──→ MCU：flash / reset / serial / debug
        ├──→ 物理 Linux：transfer / shell / debug / Managed Data
        └──→ 虚拟 Linux：transfer / runtime / shell / Managed Data
```

三类 target 共享能力名称，让上层入口可以询问“这个 target 能不能 debug 或 transfer”；
adapter 不强迫它们共享容器、目录、daemon 或生命周期。MCU 没有 Linux Target Environment，
虚拟 target 也不承诺物理宿主机行为。

### Adapter 与 provider 的调用关系

| 角色 | 所在侧 | 负责 | 不负责 |
|---|---|---|---|
| Target adapter | 开发机 | 消费开发机上的 Build Output；把统一能力请求转换为对应 target 操作 | 保存唯一项目资产；管理 build backend |
| Target provider | 目标侧或远程基础设施 | 管理 Target/Target Environment 的生命周期、内部状态和资源 | 构建用户源码；定义应用启动方式 |

物理 Linux 的 provider 角色由 `rm-relay-node` 承担；virtual target provider 以 K3s namespace
和相关资源实现。MCU 没有独立 provider daemon，adapter 直接调用宿主机上的
OpenOCD、DFU、serial 或 GDB 后端。Adapter/provider 的具体接口尚未确定，但这条责任边界
不会随协议选择改变。

## 开发机保持控制权

Target 链路中的方向是固定的：

```text
源码与构建请求   开发机 → local / remote backend
Build Output      remote backend → 开发机 → target
shell / debugger  开发机 ↔ target
Managed Data      target → 开发机
```

Workspace builder 不成为 debugger 或数据中转站。特殊网络可以增加独立 jump host，但不能
把 target 的实时连接变成构建服务职责。

Linux 调试可以使用 target 中的 GDB/gdbserver 或 debugpy；IDE 保留开发机源码，并根据稳定
构建路径映射 symbols。ROS CLI、日志、trace、bag 和可视化由用户在应用层选择。RM Relay
提供 Target Environment、交互入口、调试连接和受管数据路径，不解析应用进程结构。

## 物理 Linux target

首条物理 Linux 路径面向能通过 SSH 和 `sudo` 初始化的 Debian/Ubuntu LTS，覆盖 `amd64`
与 `arm64`。设计中，`rm-relay-node` 以原生 `.deb` 安装在宿主机：

```text
开发机上的 rm-relay
        │
        ▼
物理宿主机上的 rm-relay-node
        ├── environment image / container
        ├── 受管挂载
        ├── 版本或内容摘要
        └── target 能力与连接信息
```

普通客户端不直接访问 target 的 Docker socket。`rm-relay-node` 封装镜像、容器、挂载和
基础配置，但不保存用户源码，也不决定普通程序、脚本、`ros2 run` 或 `ros2 launch` 如何
启动。首次登记可以复用已有 SSH 权限安装对应架构软件包；具体命令、认证和通信协议要在
组件设计中确定。

### Target Environment 是长期环境，不是一次 session

物理 Target Environment 在不存在时创建，在环境版本或摘要不匹配时重建。终端断开、一次
Build Job 结束或用户程序退出都不会销毁它。

容器根文件系统保持只读；Build Output 与 Managed Data 位于宿主机受管挂载中。环境可以
重建，Build Output 可以重新传输，因此两者都不能保存唯一项目资产。

### Mutagen 保存同步状态

物理 Linux target 的首个传输设计采用 Mutagen：

```text
开发机 Build Output ── Mutagen ──→ target 输入目录
开发机数据目录       ←─ Mutagen ─── target 数据目录
```

Mutagen session 已经保存连接与同步状态，RM Relay 不再维护平行的 Development Session
数据库，也不把容器生命周期绑定到一次终端连接。熟悉 Mutagen 的用户可以检查底层 session，
普通用户通过 RM Relay 的受约束入口操作。

同一 target 输入端同一时间只允许一个写入者。取回数据前，用户先停止仍在写文件的应用；
平台保证操作顺序，应用负责确认文件已经写完且可安全读取。

未及时取回的数据可以暂存在目标宿主机的受管目录。容量、保留期限、自动回收和人工管理
界面属于后续持久数据模块；在此之前，target 除待取回数据外不保存唯一项目资产。

### 连接与发现

黄金路径是有线直连或同一局域网 Wi-Fi。维护者预先配置网络；RM Relay 不提供 Tailscale
一类 overlay network。

首次信任完成后，target 可以通过 mDNS/Avahi 广播稳定身份与能力，使客户端在 IP 变化后
重新发现它。Target 身份不能由 IP、临时容器名或用户本地别名推导。公网反向隧道、复杂
跨网段发现和网络准入不属于首版契约。

## 虚拟 Linux target

虚拟 target 用于培训、快速体验和没有物理设备时的应用链路验证。K3s 为每个用户或租户
提供独立 namespace：

```text
用户 credential
└── namespace
    ├── virtual target runtime
    ├── RBAC 与 resource quota
    ├── isolated storage
    └── 生命周期状态
```

一般用户通过 `rm-relay` 选择 target，不直接操作 Kubernetes。Virtual target 与物理 target
共享 Build Output、交互和数据返回语义，但不复制物理宿主机拓扑，也不默认采用
Docker-in-Docker。它只承诺应用 runtime 层；kernel、驱动、物理设备和完整 OS 行为需要
真实设备或后续 VM provider 验证。

## MCU target

MCU adapter 把固件交给 OpenOCD、ROM DFU、serial bootloader 或厂商兼容工具，再由 GDB
连接调试后端。USB 设备位于开发机还是远程调试主机，由 host profile 决定。

MCU 不安装 `rm-relay-node`，不运行 Mutagen，也没有 Linux Target Environment。Board
profile 直接声明可用的 flash、reset、serial 和 debug 后端。当前 RoboMaster C 已在 macOS
完成部分真实闭环；不要把该结论外推到 Linux、Windows 或其他板卡。

## 三类 target 的边界对照

| 问题 | 物理 Linux | 虚拟 Linux | MCU |
|---|---|---|---|
| 环境载体 | 可重建 container | namespace 内 runtime | 固件自身 |
| 接入管理者 | `rm-relay-node` | virtual target provider | 开发机侧 adapter |
| 结果传入 | Mutagen（首个设计） | provider 内部传输待定 | flash/debug backend |
| 数据返回 | Managed Data 同步 | 隔离 storage 返回 | serial/debugger 等设备链路 |
| 不能替代的证据 | 真实设备、驱动、kernel | 物理宿主行为 | 其他 OS/板卡实测 |

## 仍待组件设计确定的内容

Target adapter 接口、`rm-relay-node` 通信协议、Mutagen 同步模式、目标宿主内部目录和 K3s
内部传输方式尚未确定。实现必须保留本页的外部边界：开发机是控制端；物理环境可重建；同步
状态由 Mutagen 管理；虚拟隔离由 K3s 管理；应用启动由用户项目决定。
