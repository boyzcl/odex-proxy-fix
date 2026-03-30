# Codex Proxy Fix: 开源实现蓝图 / Open-Source Implementation Blueprint

> 文档定位 / Positioning
>
> 本文档是一个面向开发者与大模型执行器的实现设计文档，目标是指导实现一个开源工具，帮助普通 Codex 用户通过“一条命令”长期修复 `reconnecting` 问题，并让 Codex 稳定走本地 HTTP 代理。

## 0. 文档元信息 / Document Metadata

- 项目代号 / Working name: `codex-proxy-fix`
- 推荐仓库名 / Recommended repo name: `codex-proxy-fix`
- 目标读者 / Audience:
  - 负责实现该工具的开发者
  - 基于文档直接开发项目的大模型
  - 需要审阅方案完整性的项目 owner
- 文档语言 / Language:
  - 中文为主
  - 关键术语与命令使用英文，便于工程落地与跨平台一致性

## 1. 目标 / Goals

### 1.1 主要目标 / Primary Goals

实现一个面向普通 Codex 用户的开源工具，满足以下产品结果：

1. 用户完成安装后，只需执行一条命令，例如 `codex-proxy fix`，就能自动检测本机环境并完成修复。
2. 修复后，Codex 在大多数真实用户环境中能够更稳定地通过本地 HTTP 代理联网，从而显著降低 `reconnecting` 频率。
3. 该工具同时提供：
   - 系统级持久化修复 / persistent system-level fix
   - 强制启动兜底 / forced launch fallback
4. 从第一版设计开始就考虑 macOS、Windows、Linux 三平台。
5. 工具默认自动化，只有在少数高不确定场景下才向用户确认。

### 1.2 非目标 / Non-Goals

以下内容不属于第一阶段目标：

1. 不做“通用 AI 客户端代理修复器”。
2. 不试图修复所有可能导致 `reconnecting` 的网络根因。
3. 不修改系统全局网络代理设置作为主路径。
4. 不依赖用户手动编辑复杂配置文件。
5. 不把第一版做成 GUI 应用。

## 2. 问题定义 / Problem Definition

### 2.1 用户视角的问题 / User-Facing Problem

用户在新版 Codex 中频繁看到 `reconnecting`，导致：

- 会话中断
- 流式输出不稳定
- 需要重复等待连接恢复
- 对产品稳定性失去信心

对普通用户而言，这不是“理解底层网络协议”的问题，而是“我希望 Codex 立即稳定可用”的问题。

### 2.2 工程视角的问题 / Engineering View

`reconnecting` 更接近以下现象的外显症状：

- 某种长连接或持续会话通道断开
- 客户端尝试恢复与后端服务或中间 app server 的连接
- 当前网络路径对持续连接、流式事件、TLS、DNS 或代理兼容性不佳

这类问题常见于：

- 直连 OpenAI 线路不稳定
- 本地网络对长连接不友好
- 公司网络、校园网、运营商网络存在中间干扰
- 代理软件已运行，但 Codex 进程没有正确继承代理环境变量
- GUI 应用与 shell 环境变量脱节

## 3. 机制判断 / Mechanism Hypothesis

### 3.1 哪些判断可以写进文档 / What We Can State Confidently

从公开材料和产品行为出发，可以合理写入以下判断：

1. Codex 不是简单的一次性 HTTP 请求工具，它依赖持续会话、流式输出和可恢复的连接语义。
2. `reconnecting` 表示客户端检测到连接或会话通道中断，并在尝试恢复。
3. 本地 HTTP 代理在很多真实用户环境中能显著提升稳定性，因为它改善了出站网络路径的可达性、兼容性和长连接成功率。

### 3.2 哪些判断不要写成定论 / What Not To Overclaim

不要把下面这句话写成设计前提：

> “Codex 一定会优先走 WebSocket，连不上才切 HTTP，所以一直转圈。”

原因：

1. 这句话把不同产品面、不同通信层、不同服务路径混成了单一机制。
2. 公开信息能说明 Codex 存在持续连接、可重连会话和流式通道，但不足以支持“统一的 WebSocket 失败后自动回退 HTTP”这一强结论。
3. 如果产品方案建立在错误的底层假设上，后续实现会误把“代理注入”做成“强行改协议”，这是方向性错误。

### 3.3 更稳妥的表述 / Recommended Wording

建议文档统一采用以下表述：

> `reconnecting` 的根本含义是 Codex 所依赖的持续会话通道发生中断，客户端正在重连。对于普通用户，高命中率、低侵入、可产品化的修复路径不是改造协议本身，而是确保 Codex 通过一个稳定、可控、可持久化继承的本地 HTTP 代理出站。

### 3.4 为什么本地 HTTP 代理有效 / Why Local HTTP Proxy Often Works

本工具的核心产品判断：

1. 用户本机往往已经有代理软件在运行，但 Codex 没有用上它。
2. 本地代理可把复杂的外网问题收敛成一条更稳定的本地出站路径。
3. 相比修改系统全局代理，本地环境变量注入的风险更低、回滚更简单、兼容性更可控。
4. 对普通用户来说，“让 Codex 正确走现成的本地 HTTP 代理”比“让用户理解网络协议”更符合产品价值。

## 4. 产品原则 / Product Principles

本项目应综合借鉴以下产品哲学：

### 4.1 像 `uv`、`bun`、`pnpm` 一样

- 安装简单
- 命令简洁
- 默认自动完成大多数决策
- 错误信息可执行，不空泛

### 4.2 像 `Tailscale`、`Cloudflare WARP`、`OrbStack` 一样

- 系统集成扎实
- 状态可观测
- 修复、状态检查、回滚形成闭环
- 不把复杂性转嫁给用户

### 4.3 像高质量开发者工具一样

- 架构清晰
- 跨平台边界明晰
- 日志和异常分层
- 可自动化测试
- 文档足以驱动实现

### 4.4 产品级原则 / Product-Level Principles

1. 用户只需要理解一个入口命令：`fix`
2. 默认自动检测，异常时再确认
3. 持久化修复与强制启动兜底必须同时存在
4. 不破坏用户现有代理工具和系统网络配置
5. 所有改动必须可回滚

## 5. 技术选型 / Technology Choice

### 5.1 候选栈对比 / Stack Comparison

#### Go

优点 / Pros:

- 编译成单文件二进制，分发成本低
- 跨平台成熟
- 标准库足以覆盖网络探测、文件操作、模板生成、子进程调用
- 非常适合 CLI、守护式系统集成与安装器逻辑
- 适合 Homebrew、winget、Scoop、GitHub Releases 分发

缺点 / Cons:

- UI 表达力不如前端栈
- 某些平台差异需要自己封装
- 文档和模板工程需要一定自律

#### Rust

优点:

- 二进制质量高
- 类型系统和并发安全非常强
- 适合做长期维护的系统工具

缺点:

- 开发速度和 onboarding 成本高于 Go
- 对“让大模型直接快速施工”的友好度略低
- 对第一版产品化节奏不占优

#### Node.js

优点:

- 开发速度快
- CLI 生态成熟
- 模板、文本处理、发布速度快

缺点:

- 运行时依赖增加普通用户安装复杂度
- 系统级集成和跨平台细节处理不如单二进制工具自然
- 不是“一条命令修复普通用户问题”的最佳交付形态

#### Python

优点:

- 原型快
- 脚本能力强
- 适合快速验证

缺点:

- 依赖管理对普通用户不友好
- 打包与跨平台发布体验弱于 Go
- 不适合作为长期主实现栈

### 5.2 明确推荐 / Final Recommendation

第一版与长期主线都推荐使用 `Go`。

推荐理由：

1. 它最符合“普通用户安装后执行一个命令修复”的产品形态。
2. 它在跨平台 CLI 与系统集成之间取得了最佳平衡。
3. 它对大模型执行开发非常友好，模块边界清晰，代码可读性强。
4. 它更适合做开源项目的长期主实现，而不仅是快速原型。

## 5A. 产品化适配性评估 / Productization Fit Assessment

### 5A.1 结论 / Conclusion

把这件事做成一个产品化开源工具，是合适的选择，而且优于“只发几条命令”或“只写一篇教程”。

但更准确的定位不是“独立大而全产品”，而是：

- 一个高价值、强聚焦、低认知负担的实用型基础工具
- a focused utility product rather than a broad platform product

### 5A.2 多维度判断 / Multi-Dimensional Assessment

#### 用户价值 / User Value

高。

原因：

1. `reconnecting` 是强痛点，影响核心使用体验。
2. 普通用户通常不知道问题出在“代理未被继承”还是“网络路径不稳定”。
3. “安装后只跑一个命令”对这类问题的价值远高于论坛帖子、零散命令或长教程。

#### 问题标准化程度 / Problem Standardizability

中高。

原因：

1. 不同用户的网络环境不同，但“本地 HTTP 代理未被 Codex 正确使用”这一类问题高度可抽象。
2. 解决路径可以标准化成：
   - 检测
   - 验证
   - 持久化注入
   - 启动兜底
   - 回滚
3. 这使它非常适合做成 CLI 工具，而不只是案例性脚本。

#### 工程可实现性 / Engineering Feasibility

高。

原因：

1. 技术上主要是系统集成和状态管理，不依赖难以掌控的闭源 SDK。
2. 第一版完全可以在用户级权限范围内完成大部分价值交付。
3. 核心复杂度可拆分，不需要一开始就解决所有平台的所有细节。

#### 维护成本 / Maintenance Cost

中等。

原因：

1. 平台差异会带来持续维护成本。
2. Codex 的安装路径和行为未来可能变化。
3. 但只要架构把检测层、平台层、状态层拆开，维护成本是可控的。

#### 发布传播性 / Distribution and Adoption

中高。

原因：

1. 这类工具天然适合 GitHub + Homebrew + winget/Scoop 分发。
2. 目标用户群体与开发者工具生态重合，传播路径清晰。
3. 单命令修复的传播文案非常强。

#### 商业独立性 / Standalone Business Potential

低到中。

原因：

1. 这更像一个开源基础设施工具，而不是独立商业产品。
2. 适合作为开源项目、个人品牌项目、开发者工具 portfolio，而不是第一天就按商业 SaaS 去设计。

#### 长期价值 / Long-Term Value

中高。

原因：

1. 即使 Codex 后续改善网络稳定性，用户侧“代理继承”和“GUI 会话注入”问题仍会持续存在。
2. 该项目在开源语境下具有长期参考价值，可演进为一类典型的 desktop developer network fix utility。

### 5A.3 最终判断 / Final Judgement

从用户价值、工程可行性、传播效率、维护成本和产品清晰度综合看，这是一个合适的产品化方向。  
但应控制边界，保持“高聚焦实用工具”定位，避免扩张成模糊的通用代理管家。

## 6. 产物形态 / Product Shape

### 6.1 最终交付形态 / End Product

一个独立开源 CLI 工具：

- 名称 / Name: `codex-proxy-fix`
- 主命令 / Primary executable: `codex-proxy`

### 6.2 核心用户体验 / Core User Experience

用户安装完成后：

```bash
codex-proxy fix
```

工具完成：

1. 检测操作系统
2. 检测 Codex 安装位置
3. 检测本地代理端口
4. 验证该端口是否真的是可用 HTTP 代理
5. 安装持久化修复机制
6. 创建强制启动兜底入口
7. 输出清晰结果和下一步建议

## 6A. 安装与首次运行体验 / Installation and First-Run UX

### 6A.1 主安装路径 / Primary Install Paths

推荐给普通用户的安装路径：

- macOS:
  - `brew install codex-proxy-fix`
- Windows:
  - `winget install codex-proxy-fix`
  - 或 `scoop install codex-proxy-fix`
- Linux:
  - 官方安装脚本
  - 或 Homebrew Linux

### 6A.2 首次运行体验 / First-Run Experience

用户安装后最短路径：

```bash
codex-proxy fix
```

期望体验：

1. 在 3 到 10 秒内完成大部分检测
2. 输出一页以内的结果
3. 若失败，直接告诉用户下一步动作
4. 若成功，告诉用户：
   - 可以正常从系统图标打开 Codex
   - 如果仍然出现 `reconnecting`，可运行 `codex-proxy launch`

### 6A.3 为什么不是“一键安装自动修复” / Why Not Auto-Fix During Install

不建议把修复逻辑塞进安装脚本本身。

原因：

1. 安装和修复是两件不同的事
2. 用户需要一个可以重复执行的显式修复入口
3. `fix`、`doctor`、`unset` 形成闭环，产品边界更清晰

## 7. CLI 设计 / CLI Design

### 7.1 命令总览 / Command Surface

```text
codex-proxy fix
codex-proxy status
codex-proxy doctor
codex-proxy launch
codex-proxy unset
codex-proxy version
codex-proxy completion
```

### 7.2 各命令职责 / Command Responsibilities

#### `codex-proxy fix`

主入口，面向普通用户。

职责：

- 自动检测环境
- 自动选择最佳代理端口
- 自动安装系统级持久化方案
- 自动创建强制启动兜底
- 默认静默完成，必要时询问用户确认

建议参数：

```text
codex-proxy fix
codex-proxy fix --port 7897
codex-proxy fix --codex-path /path/to/Codex
codex-proxy fix --yes
codex-proxy fix --dry-run
codex-proxy fix --verbose
```

#### `codex-proxy status`

展示当前修复状态。

输出应至少包括：

- 操作系统
- Codex 安装路径
- 已选择代理地址
- 持久化修复是否已安装
- 强制启动入口是否可用
- 最近一次诊断结论

#### `codex-proxy doctor`

只诊断，不修改。

职责：

- 枚举候选代理端口
- 验证监听状态与协议可用性
- 诊断持久化机制是否生效
- 诊断 Codex 启动链路是否能继承环境变量
- 给出具体可执行建议

#### `codex-proxy launch`

强制启动兜底入口。

职责：

- 显式注入代理环境变量
- 启动 Codex
- 即使系统级持久化未生效，也尽量保证本次会话可用

#### `codex-proxy unset`

回滚所有由本工具写入的内容。

职责：

- 删除持久化注入机制
- 删除工具创建的包装器、快捷方式或桌面入口
- 保留用户自己的代理软件与系统其他设置

### 7.2A 推荐退出码 / Recommended Exit Codes

建议定义稳定退出码，便于自动化与 issue 排查：

- `0`: success
- `10`: Codex not found
- `11`: no usable HTTP proxy found
- `12`: persistent fix install failed
- `13`: fallback launcher install failed
- `14`: verification incomplete
- `20`: internal unexpected error

### 7.3 输出风格 / Output Style

输出应参考高质量 CLI：

- 结论先行
- 描述具体，不抽象
- 明确写出修复了什么、未修复什么
- 错误建议包含下一步动作

示例：

```text
Codex proxy fix completed.

Detected:
- OS: macOS 15
- Codex: /Applications/Codex.app
- HTTP proxy: http://127.0.0.1:7897

Installed:
- Login-session environment injection
- Fallback launcher

Next:
- You can launch Codex normally from the app icon.
- If reconnecting still appears, run: codex-proxy launch
```

## 8. 自动检测策略 / Detection Strategy

### 8.1 检测原则 / Detection Principles

不要只看端口是否监听。

应采用“三层判定”：

1. 监听存在 / listener exists
2. 进程特征合理 / process looks like a proxy
3. 代理能力验证成功 / proxy capability verified

### 8.2 操作系统检测 / OS Detection

支持：

- `darwin`
- `windows`
- `linux`

输出统一数据结构：

```json
{
  "os": "darwin",
  "version": "15.x",
  "arch": "arm64"
}
```

### 8.3 Codex 安装位置检测 / Codex Installation Detection

优先级建议：

1. 用户显式参数 `--codex-path`
2. 已知默认安装路径
3. PATH 中的 `codex`
4. 平台常见应用位置扫描

平台示例：

- macOS:
  - `/Applications/Codex.app`
  - `/Applications/Codex.app/Contents/MacOS/Codex`
  - `/Applications/Codex.app/Contents/Resources/codex`
- Windows:
  - `%LocalAppData%`
  - `%ProgramFiles%`
  - PATH 中的 `codex.exe`
- Linux:
  - `/usr/bin/codex`
  - `/usr/local/bin/codex`
  - AppImage / desktop entry / PATH

### 8.4 代理端口检测 / Proxy Port Detection

优先级来源：

1. 用户显式参数 `--port`
2. 环境变量中的现有代理配置
3. 常见本地代理端口扫描
4. 本地代理进程反查监听端口

默认候选端口：

- `7897`
- `7890`
- `1087`
- `1080`
- `8080`
- `3128`

### 8.5 代理进程识别 / Proxy Process Heuristics

进程名匹配可加权，但不能作为唯一依据。

候选关键字：

- `clash`
- `mihomo`
- `surge`
- `v2ray`
- `xray`
- `sing-box`
- `nekoray`
- `loon`
- `quantumult`

### 8.6 代理能力验证 / Proxy Capability Verification

必须执行一次真实的短超时验证。

推荐验证方式：

1. 构造带 HTTP proxy transport 的请求
2. 发往一个稳定、轻量、可替换的测试 URL
3. 只验证“经由该代理能建立外部连接”，不依赖返回体内容

实现要求：

- 超时短，默认 `2s-4s`
- 不把目标站点写死成唯一依赖
- 允许多个 fallback endpoint
- 清晰区分：
  - 端口未监听
  - 端口可连但不是 HTTP 代理
  - 代理可用但外网不可达

建议测试目标策略：

1. 默认使用多个可配置 endpoint
2. 优先 HEAD 请求
3. 允许通过 `--check-url` 覆盖
4. 支持在未来版本中切换到更适合长期稳定性的健康检查 endpoint 列表

### 8.7 评分与选择 / Port Scoring

为每个候选端口计算分数：

- 用户显式指定: `+100`
- 已存在于环境变量: `+40`
- 命中常见代理进程: `+20`
- 验证成功: `+50`
- 仅监听无验证成功: `-30`

选择原则：

1. 优先验证成功的端口
2. 分数相同优先回环地址 `127.0.0.1`
3. 仍冲突时提示用户确认

## 9. 修复策略 / Fix Strategy

### 9.1 双层修复模型 / Two-Layer Fix Model

第一层：系统级持久化修复

- 目标：用户正常从系统图标启动 Codex 时也尽量生效

第二层：强制启动兜底

- 目标：即使持久化未正确继承，也保证用户还有一个高成功率启动入口

### 9.2 为什么必须双层 / Why Two Layers Are Mandatory

单做持久化修复不够，因为：

- GUI 应用对环境变量继承在不同平台和会话中并不稳定
- 系统更新、用户注销重登、桌面环境差异都可能影响结果

单做强制启动兜底也不够，因为：

- 这会把长期复杂性转嫁给普通用户
- 用户仍会从默认图标打开 Codex

因此架构上必须同时提供：

- best-effort persistent fix
- deterministic forced launcher

## 10. 平台实现方案 / Platform Implementations

### 10.0 权限模型总原则 / Permission Model

默认情况下，第一版应坚持用户级安装，不要求管理员权限。

理由：

1. 这最符合普通用户实际可操作路径
2. 降低安装阻力
3. 降低误伤系统配置的风险
4. 与“可回滚、低侵入”原则一致

仅在未来版本需要更深系统集成时，才考虑管理员权限路径，而且必须是可选项而非默认项。

### 10.1 macOS

#### 10.1.1 持久化方案 / Persistent Path

推荐主路径：

- 生成用户级脚本
- 安装 `LaunchAgent`
- 在用户登录会话中执行 `launchctl setenv`

推荐写入位置：

- 脚本:
  - `~/Library/Application Support/codex-proxy-fix/setenv.sh`
- LaunchAgent:
  - `~/Library/LaunchAgents/com.codexproxyfix.env.plist`

注入变量：

- `HTTP_PROXY`
- `HTTPS_PROXY`
- `ALL_PROXY`
- `NO_PROXY`

关键说明：

- `LaunchAgent + launchctl setenv` 是 best-effort 的登录会话注入，不应向用户承诺“所有 GUI 进程在任何时刻都百分百继承”
- 工具应在文案中诚实表达这一点

#### 10.1.2 强制启动兜底 / Fallback Launch

推荐方式：

- 提供 `codex-proxy launch`
- 显式以环境变量方式启动：
  - `/Applications/Codex.app/Contents/MacOS/Codex`

如果找不到 GUI 可执行文件，则退回到 CLI 可执行入口。

#### 10.1.3 验证方式 / Verification

`fix` 完成后：

1. 检查 LaunchAgent 是否存在
2. 检查脚本是否存在且可执行
3. 检查当前用户会话变量是否已设置
4. 可选地尝试一次 `launch --dry-run`

#### 10.1.4 不建议的路径 / Paths to Avoid

第一版不建议：

- 修改系统级网络代理
- 注入 shell profile 作为 GUI 修复主方案
- 覆盖原始 `Codex.app` 包内容
- 使用需要关闭 SIP 或更高系统权限的方案

### 10.2 Windows

#### 10.2.1 持久化方案 / Persistent Path

推荐主路径：

- 写入用户级环境变量
- 创建用户登录时运行的轻量启动脚本或计划任务
- 可选更新用户桌面快捷方式或生成专用快捷方式

推荐优先级：

1. 用户环境变量
2. 用户级计划任务或登录脚本
3. Fallback launcher

环境变量：

- `HTTP_PROXY`
- `HTTPS_PROXY`
- `ALL_PROXY`
- `NO_PROXY`

注意：

- Windows 上 GUI 进程对新增环境变量的感知与启动链路有关
- 需要工具通过状态检查明确告诉用户“已安装持久化修复，但现有桌面会话可能需要重新启动 Codex 或重新登录”

#### 10.2.2 强制启动兜底 / Fallback Launch

推荐生成：

- `codex-proxy launch`
- 可选 `.cmd` 或 PowerShell wrapper

作用：

- 对启动进程显式注入代理变量
- 启动 `Codex.exe`

#### 10.2.3 不建议的路径 / Paths to Avoid

第一版不建议：

- 直接修改系统级代理为默认策略
- 依赖管理员权限注册复杂服务
- 改写用户现有开始菜单中的原始 Codex 项

### 10.3 Linux

#### 10.3.1 持久化方案 / Persistent Path

推荐主路径：

- `systemd --user`
- 或 `~/.config/environment.d/*.conf`
- 桌面环境下可选 `.desktop` launcher override

推荐顺序：

1. `environment.d` for session env
2. `systemd --user` 辅助检查
3. fallback launcher

#### 10.3.2 强制启动兜底 / Fallback Launch

- `codex-proxy launch`
- 桌面环境可选生成 `Codex (Proxy).desktop`

#### 10.3.3 不建议的路径 / Paths to Avoid

第一版不建议：

- 强依赖 root 权限
- 强耦合某一个桌面环境
- 修改系统级 `/etc/environment` 作为默认方案

### 10.4 平台差异结论 / Cross-Platform Conclusion

统一产品体验，不统一底层实现。

也就是说：

- 用户看到的命令和结果尽量一致
- 平台内部的持久化机制按各自最佳实践实现

## 11. 配置与文件布局 / Config and File Layout

### 11.1 用户配置目录 / User Config Home

建议：

- macOS:
  - `~/Library/Application Support/codex-proxy-fix/`
- Linux:
  - `~/.config/codex-proxy-fix/`
- Windows:
  - `%AppData%\codex-proxy-fix\`

### 11.2 持久化状态文件 / State File

建议维护：

`state.json`

示例：

```json
{
  "version": 1,
  "platform": "darwin",
  "selected_proxy": "http://127.0.0.1:7897",
  "codex_path": "/Applications/Codex.app/Contents/MacOS/Codex",
  "persistent_fix_installed": true,
  "fallback_launcher_installed": true,
  "last_fix_time": "2026-03-30T22:00:00+08:00"
}
```

### 11.3 为什么需要状态文件 / Why State Matters

因为 `unset`、`status`、`doctor` 都不能靠重新猜测系统状态完成全部逻辑。状态文件用于：

- 精确回滚
- 明确知道本工具写入了哪些文件
- 区分“用户原本就有的配置”和“工具新增的配置”

## 12. 状态机 / State Machine

### 12.1 主流程状态 / Main Flow States

```text
START
  -> detect_os
  -> detect_codex
  -> detect_proxy_candidates
  -> verify_proxy
  -> select_proxy
  -> install_persistent_fix
  -> install_fallback_launcher
  -> verify_installation
  -> write_state
  -> DONE
```

### 12.2 失败分类 / Failure Classes

#### F1: 未发现 Codex

处理：

- 输出平台相关安装位置建议
- 不继续安装修复

#### F2: 未发现可用 HTTP 代理

处理：

- 告诉用户发现了哪些候选端口
- 为什么它们不合格
- 提示用户启动本地代理软件后重试

#### F3: 找到代理但持久化安装失败

处理：

- 不隐瞒失败
- 仍尝试安装 `launch` 兜底
- 输出明确建议：当前可用 `codex-proxy launch`

#### F4: 所有安装成功，但验证不完整

处理：

- 标记为 partial success
- 明确指出哪些环节是 best-effort

## 13. 安全与边界 / Security and Boundaries

### 13.1 安全原则 / Security Principles

1. 不修改系统全局代理作为默认策略
2. 不覆盖用户自己的代理软件配置
3. 不偷偷上传诊断数据
4. 不在后台常驻复杂守护进程
5. 所有写入都必须可回滚

### 13.2 用户信任 / User Trust

该工具本质上在做“环境变量注入 + 启动器包装 + 会话集成”，因此必须：

- 让所有写入路径可见
- 让 `status` 可解释
- 让 `unset` 可逆
- 让 README 诚实说明边界

### 13.3 支持边界 / Support Boundary

README 和 CLI 输出中必须明确以下边界：

1. 本工具旨在让 Codex 更可靠地使用本地 HTTP 代理，不保证修复所有网络问题
2. 如果用户本地没有可用代理，本工具不会凭空创建外网通路
3. 某些系统会话或桌面环境下，系统级持久化修复是 best-effort
4. `launch` 是可靠兜底，不是失败后才“临时凑合”的次等功能

## 14. 代码架构 / Code Architecture

### 14.1 推荐目录结构 / Recommended Repo Structure

```text
cmd/
  codex-proxy/
    main.go

internal/
  app/
    fix.go
    doctor.go
    status.go
    unset.go
    launch.go
  detect/
    os.go
    codex.go
    proxy_ports.go
    proxy_verify.go
    processes.go
  platform/
    common/
      env.go
      paths.go
    darwin/
      install.go
      launchagent.go
      launcher.go
      verify.go
      rollback.go
    windows/
      install.go
      envvars.go
      launcher.go
      shortcut.go
      verify.go
      rollback.go
    linux/
      install.go
      environmentd.go
      launcher.go
      desktop.go
      verify.go
      rollback.go
  state/
    state.go
  ui/
    print.go
    prompt.go
  templates/
    templates.go

assets/
  darwin/
    launchagent.plist.tmpl
  linux/
    environment.conf.tmpl
    codex-proxy.desktop.tmpl
  windows/
    launch.cmd.tmpl
    launch.ps1.tmpl

scripts/
  release/
  ci/

.github/
  workflows/

README.md
LICENSE
```

### 14.2 模块职责 / Module Responsibilities

#### `internal/app`

聚合命令级业务流程，不承载平台细节。

#### `internal/detect`

负责发现事实，不做写入。

#### `internal/platform/*`

负责平台相关的写入、验证、回滚。

#### `internal/state`

负责状态文件读写和版本迁移。

#### `internal/ui`

负责 CLI 输出、确认交互、错误信息整形。

## 15. 关键实现细节 / Key Implementation Details

### 15.1 `fix` 的交互策略 / Interaction Strategy

默认无交互。

只有以下场景提示用户确认：

1. 找到多个同分高置信代理端口
2. 找到多个 Codex 安装位置且都像是有效安装
3. 平台持久化写入即将覆盖本工具先前生成但已与当前状态不一致的文件

### 15.2 `doctor` 的价值 / Doctor as a First-Class Feature

不要把 `doctor` 当附属命令。

`doctor` 是产品可信度的关键，因为：

- 用户要知道“为什么这次 fix 成功”
- 用户要知道“为什么 fix 失败”
- issue 报告和自动化支持会依赖 `doctor`

### 15.3 强制启动器 / Forced Launcher

`launch` 应始终优先读取 state 中的已选代理，而不是重新猜测。  
只有当 state 缺失时，才回退到动态检测。

### 15.4 环境变量策略 / Env Strategy

统一写入：

- `HTTP_PROXY=http://127.0.0.1:<port>`
- `HTTPS_PROXY=http://127.0.0.1:<port>`
- `ALL_PROXY=http://127.0.0.1:<port>`
- `NO_PROXY=localhost,127.0.0.1`

说明：

- 即便某些链路未使用 `ALL_PROXY`，统一写入能降低平台差异和上游实现差异带来的问题
- `NO_PROXY` 应尽量保守，不要覆盖用户已有值；需要 merge

### 15.5 不要做的事情 / Anti-Patterns

1. 不要假设只有 `7897`
2. 不要假设代理软件名字固定
3. 不要只看端口监听就判定成功
4. 不要只做 alias 而自称“长期修复”
5. 不要无提示覆盖用户已有全局配置

## 16. 可观测性 / Observability

### 16.1 日志级别 / Logging

建议支持：

- `error`
- `warn`
- `info`
- `debug`

默认只展示高信号结果，`--verbose` 开启更多细节。

### 16.2 诊断快照 / Diagnostic Snapshot

建议 `doctor --json` 输出机器可读诊断：

```json
{
  "platform": "darwin",
  "codex_found": true,
  "codex_path": "/Applications/Codex.app/Contents/MacOS/Codex",
  "proxy_candidates": [
    {
      "url": "http://127.0.0.1:7897",
      "listening": true,
      "verified": true,
      "score": 110
    }
  ],
  "persistent_fix": {
    "installed": true,
    "verified": true
  },
  "fallback_launcher": {
    "installed": true
  }
}
```

这个能力对 CI、自助排障、issue 模板都很有价值。

## 17. 测试策略 / Testing Strategy

### 17.1 单元测试 / Unit Tests

覆盖：

- 端口评分逻辑
- 配置合并逻辑
- state 读写与迁移
- 路径检测
- 模板渲染

### 17.2 集成测试 / Integration Tests

覆盖：

- 模拟本地 HTTP 代理服务
- 验证 `fix`、`status`、`doctor`、`unset` 的主流程
- 平台模板输出正确性

### 17.3 手工验收矩阵 / Manual Acceptance Matrix

至少验证：

1. macOS + Clash/Mihomo on `7897`
2. macOS + no proxy running
3. Windows + local proxy on common port
4. Linux + `environment.d` + desktop launcher
5. 多端口冲突场景
6. `unset` 回滚完整性

### 17.4 兼容性支持矩阵 / Compatibility Matrix

第一版建议在 README 中明确支持级别：

- Tier 1:
  - macOS 最新两个主版本
- Tier 2:
  - Windows 11
  - 主流 Linux 桌面发行版
- Tier 3:
  - 其他环境 best-effort

这样可以防止项目在早期过度承诺。

## 18. 发布与分发 / Distribution

### 18.1 主路径 / Main Installation Path

推荐：

- macOS: Homebrew
- Windows: winget 和 Scoop
- Linux: Homebrew Linux + 官方安装脚本

### 18.2 备用路径 / Secondary Path

提供官方安装脚本：

```bash
curl -fsSL https://example.com/install.sh | sh
```

Windows 提供 PowerShell 安装命令。

### 18.2A 发布资产 / Release Assets

每次 release 建议提供：

- macOS:
  - `darwin-arm64`
  - `darwin-amd64`
- Windows:
  - `windows-amd64.zip`
  - `windows-arm64.zip`
- Linux:
  - `linux-amd64.tar.gz`
  - `linux-arm64.tar.gz`

并附带：

- checksums
- Homebrew formula metadata
- winget/Scoop manifest source

### 18.3 为什么两条路径都要有 / Why Both

因为：

- 开发者用户偏爱包管理器
- 普通用户经常只愿意复制一条安装命令
- 开源项目增长阶段需要降低试用门槛

## 19. README 应表达的用户承诺 / README Promises

README 不应承诺“修复所有 reconnecting 根因”，而应承诺：

1. 自动检测本地 HTTP 代理
2. 自动为 Codex 安装持久化代理继承机制
3. 提供强制启动兜底
4. 提供可解释的诊断与回滚

推荐一句话定位：

> One command to make Codex reliably use your local HTTP proxy and reduce reconnecting issues.

还应包含一个明确免责声明：

> This tool improves Codex's ability to use an existing local HTTP proxy. It does not create network access on its own and does not guarantee recovery from every reconnecting cause.

## 20. MVP 范围 / MVP Scope

### 20.1 第一阶段必须完成 / Must-Have

1. Go CLI 工程骨架
2. `fix/status/doctor/launch/unset`
3. macOS 完整支持
4. Windows 和 Linux 的基础实现骨架
5. HTTP 代理检测与验证
6. state 管理
7. 日志与 JSON 诊断输出

### 20.2 第二阶段 / V2

1. 更多代理软件识别增强
2. 桌面快捷方式自动修复
3. 更完善的 GUI 桌面环境集成
4. 自助 issue bundle 导出

### 20.3 不进入 V1 的内容 / Explicitly Deferred

1. GUI 前端
2. 遥测平台
3. 云端配置同步
4. 多应用代理治理

## 21. 里程碑拆分 / Milestones

### Milestone 1: Foundation

- 初始化 Go 项目
- 定义 CLI 命令
- 完成检测层抽象
- 完成 state 结构

### Milestone 2: macOS First

- 完成 macOS `fix/status/doctor/unset/launch`
- 验证 LaunchAgent 和 fallback launcher
- 补齐验收测试

### Milestone 3: Cross-Platform Baseline

- 完成 Windows 与 Linux 基础实现
- 统一状态和输出格式
- 完成跨平台文档

### Milestone 4: Distribution

- Homebrew formula
- winget/Scoop manifests
- 官方安装脚本
- GitHub Release pipeline

## 22. 成功标准 / Success Criteria

项目完成度不应以“代码能跑”为标准，而应以以下结果衡量：

1. 普通用户安装后执行 `codex-proxy fix`，能在大多数真实环境中完成自动修复
2. 修复失败时，用户能从 `doctor` 输出中看懂原因
3. `launch` 能作为可靠兜底
4. `unset` 能干净回滚
5. 三平台实现边界清晰，不把 macOS 特性硬套给其他系统

建议增加可量化指标：

1. 在支持矩阵内，`fix` 命令主流程成功率达到可接受水平
2. `doctor` 的失败分类能够覆盖绝大多数真实问题
3. issue 中因“没有可解释错误信息”产生的反馈比例持续下降

## 23. 风险与应对 / Risks and Mitigations

### 风险 1 / Risk 1

不同平台对 GUI 进程环境变量继承并不稳定。

应对：

- 文档中明确这是 best-effort
- 始终提供 `launch` 兜底

### 风险 2 / Risk 2

用户本地代理存在，但不是 HTTP 代理或行为不标准。

应对：

- 真实验证，不只看监听
- 错误分类明确

### 风险 3 / Risk 3

用户已有复杂自定义代理环境。

应对：

- 增量写入
- 记录 state
- 提供 `unset`

### 风险 4 / Risk 4

不同版本的 Codex 安装路径变化。

应对：

- 路径检测做成可扩展策略
- 支持 `--codex-path`

### 风险 5 / Risk 5

上游 Codex 未来可能改变环境变量读取方式或安装结构。

应对：

- 将探测逻辑与平台写入逻辑解耦
- 保持 `launch` 命令为最小可靠闭环
- 在 CI 中保留 smoke test 入口

## 24. 参考实现建议 / Implementation Guidance for LLMs

如果由大模型直接实现，建议遵循以下执行纪律：

1. 先完成 `internal/detect` 和 `internal/state`
2. 再实现 `macOS` 全流程，不要一开始同时铺开三平台细节
3. `doctor` 与 `fix` 共享探测逻辑，避免重复分叉
4. 每个平台都实现：
   - install
   - verify
   - rollback
   - launcher
5. 先保证 `fix -> status -> launch -> unset` 闭环，再做分发
6. 不要在第一版引入多余依赖

## 25. 给大模型的开发起始提示 / Starter Prompt for Codex

下面这段文字可以直接作为后续大模型执行开发的起始提示：

```text
Build a cross-platform Go CLI tool named codex-proxy that fixes Codex reconnecting issues by making Codex reliably use the user's local HTTP proxy.

Read the implementation blueprint first and follow it strictly.

Project goals:
- One-command fix for normal users: `codex-proxy fix`
- Support macOS, Windows, Linux from the design stage
- Provide both persistent system-level fix and forced launch fallback
- Implement: fix, status, doctor, launch, unset
- Use Go as the primary language
- Prioritize macOS first, then add Windows/Linux baseline support

Execution constraints:
- Keep architecture modular
- Do not assume a single proxy port
- Verify proxy capability, not just listening status
- Use state tracking for rollback
- Avoid changing global system proxy settings by default
- Keep user-facing output concise and actionable

Deliver the project in milestones:
1. project scaffold
2. detection layer
3. state management
4. macOS full flow
5. Windows/Linux baseline
6. tests
7. packaging
```

## 26. 公开参考资料 / Public References

以下公开资料适合在实现阶段复核，不要求逐字依赖：

- OpenAI, Introducing the Codex app
  - https://openai.com/index/introducing-the-codex-app/
- OpenAI, Unlocking the Codex harness: how we built the App Server
  - https://openai.com/index/unlocking-the-codex-harness/
- OpenAI, Introducing GPT-5.3-Codex-Spark
  - https://openai.com/index/introducing-gpt-5-3-codex-spark/

这些参考的用途是帮助理解：

- Codex 存在持续会话和可恢复连接语义
- 不同产品面可能使用不同传输方式
- `reconnecting` 不应被草率简化成单一“WebSocket 失败再回退 HTTP”的故事

## 27. 最终结论 / Final Recommendation

这不是一个“给用户几条 shell 命令”的项目，而应被当作一个真正的跨平台产品来设计。

最终推荐方案是：

1. 用 `Go` 实现一个独立 CLI 工具 `codex-proxy`
2. 核心命令为 `fix/status/doctor/launch/unset`
3. 通过“自动检测 + 系统级持久化 + 强制启动兜底”双层策略解决问题
4. 以 macOS 为第一优先完成高质量实现，同时从架构上原生支持 Windows 和 Linux
5. 坚持可观测、可回滚、可解释，避免过度承诺底层机制

如果后续实现严格遵循本蓝图，大模型应能直接完成该开源项目的第一版高质量开发。
