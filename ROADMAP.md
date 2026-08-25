# RM Relay 路线图

路线图记录公开的建设顺序，不承诺完成时间。具体任务、负责人和外部成果通过 GitHub
Issue 追踪；只有取得真实证据的能力才会写入支持矩阵。

当前已建立 STM32 嵌入式开发基线，包括双架构开发镜像、跨平台 CMake 项目模板、PI
控制示例、STM32F407 交叉编译，以及 RoboMaster C 在 macOS 上的 DFU 烧录校验和
OpenOCD/GDB 源码调试闭环。

项目后续建设遵守[开发平台架构](docs/architecture/README.md)。当前基本链路以
Development Session 完成为终点：

```text
本地源码
  ↓
本地或远程构建
  ↓
Build Output 返回本地
  ↓
物理或虚拟 target
  ↓
交互式 CLI 与调试
  ↓
开发数据返回本地
  ↓
清理 Session
```

## 1. 扩展嵌入式实板调试支持

RoboMaster C 已完成首个 macOS 实板闭环。下一步根据 RM 队伍的真实设备补充 STM32
板卡与调试后端，并完善 Windows、Linux 的宿主接入和 VS Code/VSCodium 配置。board
profile 继续与项目结构分离。

## 2. 建立统一任务与 profile 模型

引入 mise 作为薄任务入口，继续以 CMake Presets、CMake、colcon 和原生调试后端为事实
源。环境镜像通过 Dockerfile 能力层与 Bake 官方 profile 组合；每个正式 profile 提供独立
Dev Container Template，项目扩展以 mise overlay 构建成不可变的派生镜像。

当前嵌入式模板和示例迁移到这套入口后，才能用同一方法扩展算力侧环境。

## 3. 发布镜像与远程构建体验

将正式环境镜像发布到国内 OCI Registry，减少普通用户重复构建产生的大流量访问。在固定
development image 上建立 workspace 构建服务，让 Build Output 返回开发者本地；本地
Docker 与远程 backend 使用同一项目声明和下游 target 链路。

首个远程实例由 [@x12315](https://github.com/x12315) 维护，只服务本战队和受邀友队，通过
标准 mise 任务与 SSH 接入。公开注册、多租户和复杂准入不属于这一阶段。

## 4. 建立算力侧环境与 Development Session

按照普通 Linux、原生 C++ 视觉、ROS 2 视觉、导航和厂商 runtime 的真实需求，逐步形成
有限的官方 profile。算力侧 development 与 runtime 环境共享 target environment lineage；
用户程序以 CMake Install Tree 或 ROS 2 Install Space 进入 runtime，不制作日常应用镜像。

同时完成物理与虚拟 Linux target、排他 Session、交互式 CLI、debugger 直连、受管数据
目录和 Session 清理。首条黄金路径需要真实跑通：

```text
本地源码
→ 远程构建
→ Build Output 返回本地
→ Linux 物理 target
→ Session CLI
→ 用户运行与调试
→ 数据取回
→ Session 清理
```

## 5. 提供战队部署说明

说明如何在一台或多台战队服务器上组合环境镜像构建器、Registry、workspace 构建器和
虚拟执行器，包括版本识别、cache、最低资源限制、更新和 smoke 验证。长期服务原则上由
使用它的战队自行维护。

## 6. 开展社区试用与协作

在首条完整开发链路稳定后，于 RM 论坛发布项目介绍并征集试用队伍。贡献重点包括板卡与
target profile、跨平台验证、算力侧环境和文档。仓库外工作的记录方式见
[社区工作](docs/community/README.md)。

## 后续可选模块

以下能力不阻塞当前开发链路，出现真实需求和维护力量后再设计：

- 比赛用持久部署与开机自启；
- 多机器人批量部署；
- 通过局域网或专用网卡统一分发；
- 目标设备操作系统/ISO 镜像、初始化和高级运维；
- 待取回数据的状态查询、配额、过期回收和自动清理；
- 面向陌生用户的公开注册、多租户、配额和强隔离；
- 硬件 Farm 与集中式开发板管理；
- 原生 ARM64 builder、仅有 ARM 服务器的战队部署和更广泛的跨架构调度；
- 生产级 OTA、A/B 和强事务回滚；
- 大规模仿真、神经网络训练、数据集和实验管理。

这些模块可以复用环境、Build Output、target 和 Session 契约，但不会反向扩大当前实现的
支持声明。
