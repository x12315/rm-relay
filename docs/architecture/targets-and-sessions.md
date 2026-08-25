# Target 与 Development Session

Build Output 回到本地后，由 target adapter 进入设备。RM Relay 统一 target 的外部语义，
不要求物理 Linux、虚拟 Linux 和 MCU 具有相同内部结构。

```text
本地 Build Output
        │
        ▼
target adapter
├── physical Linux：SSH → 设备宿主系统 → runtime 环境
├── virtual Linux：executor → 隔离 runtime 环境
└── physical MCU：OpenOCD / DFU / serial → MCU
```

虚拟 target 是一等 target，但普通容器只证明应用 runtime 层能够工作。只有 VM 或专门的
system container 才能覆盖“目标宿主机安装 Docker、再管理 runtime”这一层。首版不会为了
拓扑相似而默认使用 Docker-in-Docker。

Target adapter 的具体接口尚未确定。它至少要让上层识别 target、取得排他使用权、传入
Build Output，并暴露该 target 实际支持的 shell、flash 或 debug 能力。MCU 不会为了接口
统一而模拟 Linux container、文件系统或交互终端。

## 调试链路

构建发生在哪里，不决定 debugger 必须运行在哪里。构建服务器不进入实时调试链路：

```text
源码与构建请求        开发者电脑 → 构建服务器
Build Output           构建服务器 → 开发者电脑 → target
断点、变量和调用栈    开发者电脑 ↔ target
```

Linux 默认通过 SSH 直接连接 target 中的 GDB、gdbserver 或 debugpy。IDE 使用本地源码，
构建时将不稳定的服务器路径映射为 `/workspace`，再由 IDE 或 GDB 映射回本地工作区。构建
服务器只在无法直达 target 的特殊网络中充当跳板，不是默认 debugger backend。

MCU 继续使用 OpenOCD、GDB、DFU 或串口等原生后端。容器负责生成固件和提供工具；USB
后端运行在宿主还是容器，取决于 host profile 和已经验证的设备接入方式。

ROS CLI、日志、trace、bag 和可视化属于用户在目标环境中的应用工具。RM Relay 只保证
终端、调试连接和受管数据目录，不把 DDS、ROS topic 或可视化协议变成部署协议。

## Session 的定义

Development Session 是一次连续占用 target 的开发时段：

> Session 从取得 target 开始，到显式关闭或租约超时为止；期间允许多次构建、传输、CLI
> 连接和调试。

它不是一次程序执行，也不是项目整个赛季的永久工作区。Session 是否存在由 target 是否仍
被占用决定，与用户程序当前是否运行无关。

```text
Project                         数月或一个赛季
└── Session                     一次连续 target 开发时段
    ├── Build Operation         可以发生多次
    ├── Deploy Operation        在 Session 内串行
    ├── CLI Connection          可以断开和重连
    └── User Process            由用户管理
```

一个 target 同一时间只能属于一个 Session。Session 内可以运行多个普通进程、ROS package
或 node；RM Relay 不读取 launch 文件，也不提供通用应用 `run/stop` 管理。

## 状态模型

公开状态只保留三种：

```text
Closed
  │ open
  ▼
Active
  │ close / expire
  ├──────────────────┐
  │ 数据已取回        │ 数据待取回
  ▼                  ▼
Closed           Recoverable
                     │ recover / discard
                     ▼
                   Closed
```

`Active` 表示 target 被排他占用，Session runtime 可以重复 attach、deploy 和 debug。终端
短暂断开不会关闭 Session，只更新最后活动时间。

`Recoverable` 表示 runtime 已停止、target 已释放，只剩尚未确认取回的数据。取回数据不再
占用原 target。

`Closed` 表示 runtime、项目工作区、临时目录和锁均已清理，目标端不再保留该 Session 的
项目内容。

`Opening` 和 `Closing` 只是一项操作的临时阶段，不进入长期公开状态。具体 idle timeout、
maximum lifetime 和数据保留时间由 target/provider profile 决定，当前不固定数值。

## 项目身份与增量传输

每个项目拥有稳定的 `project_id`。它不从目录名、绝对路径或 Git remote 推导；项目改名、
移动或被不同成员 clone 后仍保持不变。模板不能携带一个供所有项目复用的固定 ID，创建新的
逻辑项目时必须生成新的 ID。

`project_id` 用于当前 Session 内识别项目工作区和增量传输。Session 关闭后不保留项目
cache，因此新 Session 默认完整传输。目标机不会为了下一次调试保存 Install Tree、源码、
runtime writable layer 或其他项目状态。

当前不要求 `artifact_id` 参与增量传输。以后如果出现构建结果校验和追踪需求，可以增加
manifest，但不能因此改变目标机的清理原则。

`project_id` 不是用户身份或认证凭据。远程服务的 owner、访问控制和配额使用独立身份。

## 数据目录

Session 只提供一个稳定的可写入口：

```text
Session 内
/workspace/data
RM_RELAY_DATA_DIR=/workspace/data

目标机实际存储
<relay-state-root>/pending-data/<project_id>/<session_id>/

取回到本地
<project-root>/.rm-relay/data/<target_id>/<session_id>/
```

`/workspace/data` 是用户契约，目标宿主机实际路径由 provider 管理。用户需要保留的日志、
结果或调试文件应写入该目录；平台不能在不理解应用的情况下捕获任意绝对路径产生的文件。

Session 关闭时先尝试取回数据。本地确认完整收到后，删除目标端目录；客户端离线或取回失败
时，释放 target 并进入 `Recoverable`。除待取回数据外，目标机不能保留任何跨 Session 的
项目内容。

当前版本只要求创建目录、建立映射、确认取回并删除。状态查询、容量限制、保留策略、过期
回收和自动清理由后续的持久数据管理模块承担，不在本轮展开。

## 清理不变量

Session 完成时必须满足：

- target 的排他锁已经释放；
- runtime 和其中的用户进程已经停止；
- 项目 Build Output、临时目录和 writable layer 已删除；
- 没有把项目依赖安装进目标宿主系统；
- 只有尚未取回的数据可以进入 `Recoverable`；
- 清理失败可以被检测并再次执行。

容器 target 可以通过销毁 Session runtime 结束其中的进程。原生 Linux、MCU 和其他
provider 如何提供同等清理保证尚未定案；在该边界完成前，不能宣称它们已经通过完整
Session 验证。
