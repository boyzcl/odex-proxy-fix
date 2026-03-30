# codex-proxy-fix

[English README](./README.md)

用一条命令，让 Codex 更稳定地使用你本机已经在运行的本地 HTTP 代理，并降低 `Reconnecting...` 出现的概率。

`codex-proxy-fix` 是一个以 macOS 为优先的 CLI 工具，面向这样一类用户：

- 你的本地代理本来就在跑
- 但 Codex 没有稳定继承或使用这个代理
- 于是你在使用过程中反复看到 `Reconnecting...`

当前定位：

- 发布阶段：`alpha`
- 当前在本仓库内实际验证的平台：`macOS`
- Windows / Linux：代码路径已存在，但尚未正式验证

快速开始：

```bash
codex-proxy doctor --verbose
codex-proxy fix
codex-proxy status
```

如果正常从图标启动 Codex 仍然会重连：

```bash
codex-proxy launch
```

如果你想移除修复并恢复之前捕获到的环境变量状态：

```bash
codex-proxy unset
```

## 为什么会有这个工具

`Reconnecting...` 只是一个症状，不是单一根因。

在实际使用中，这个状态常见于几类完全不同的问题：

1. 本地代理继承问题
   - 你的本地代理已经在运行，但 Codex 启动时没有拿到正确的代理环境变量
   - 这在 macOS 的 GUI 应用启动链路里尤其常见，因为 shell 环境和 app 环境并不总是一致

2. 本地网络路径不稳定
   - 机器到服务端的直连路径不够稳定
   - 普通网页可能还能打开，但长连接、流式输出、持续会话更容易掉

3. 本地代理节点或出口 IP 不稳定
   - 代理程序在运行，但当前节点、上游链路、出口 IP 质量不稳定
   - 简单请求可能成功，但 Codex 这种持续会话型产品更容易反复重连

4. 服务端或区域性问题
   - OpenAI 在某个地区、某个时间段、某个模型后端可能存在降级或故障
   - 这种情况下，通常会出现“很多用户同时都在报 reconnecting”

5. 认证或会话状态问题
   - 登录态、授权、会话恢复链路有问题时，也可能表现为不断重连

这个工具刻意聚焦的是前 3 类，尤其是第 1 类。

## 我们认为实际发生了什么

Codex 不是一个只发一次请求就结束的产品。  
它依赖更长生命周期的会话、流式输出和重连恢复语义。

这意味着它比普通网页浏览更容易受到下面这些因素影响：

- 代理环境没有正确继承
- 出口路径不稳定
- DNS / TLS 摩擦
- 长连接被中途重置

所以“网页没问题，但 Codex 一直 reconnecting”这件事是完全可能发生的。  
网络可能足够支撑普通浏览，但不足以支撑 Codex 的持续会话。

## 这个工具能解决什么场景

这个工具适合下面这些情况：

- 本机已经有可用的本地 HTTP 代理
- 你发现从终端带代理启动 Codex 更稳，而从图标启动容易出问题
- 这个问题主要是你这台机器、你的本地网络路径或你的本地代理继承方式导致的
- 你想要一个可重复、可回滚、低侵入的修复方式

更具体一点，它主要解决这些高命中场景：

- Codex 没有继承到代理环境变量
- Codex 没有走到你期望的本地代理路径
- GUI 启动和终端启动行为不一致
- 当流量被强制走正确的本地 HTTP 代理时，Codex 稳定性明显改善

## 这个工具不能解决什么

这个工具不能解决：

- OpenAI 服务端事故
- 区域性服务退化
- 某个模型后端不稳定
- 账号、认证、会话异常
- 本地根本没有可用代理的情况

它也不会凭空为你创建外网连接能力。

如果问题根因在服务端，这个工具可能完全没有帮助。

## 这个工具实际做了什么

它专门解决一个问题：

- Codex 因为没有稳定使用你本机现有的本地 HTTP 代理而变得不稳定

它提供这些命令：

- `codex-proxy fix`
  - 检测 Codex
  - 检测并验证本地 HTTP 代理
  - 安装 best-effort 的持久化修复
  - 创建兜底启动器
- `codex-proxy doctor`
  - 只检查，不修改
- `codex-proxy status`
  - 展示当前安装状态
- `codex-proxy launch`
  - 用显式代理环境变量启动 Codex，作为可靠兜底
- `codex-proxy unset`
  - 删除工具写入的内容，并在可能的情况下恢复原来的环境变量状态

## 修复方案是怎么工作的

在 macOS 上，`fix` 当前会安装：

1. `~/Library/Application Support/codex-proxy-fix/` 下的代理环境脚本
2. `~/Library/LaunchAgents/` 下的用户级 `LaunchAgent`
3. 同目录下的 fallback launcher
4. 一个本地 state 文件，记录：
   - 检测到的 Codex 路径
   - 选中的代理
   - 工具管理的文件路径
   - 修改前捕获到的原始持久化环境变量快照

这个方案有两层：

1. best-effort 持久化修复
   - 让你正常从 GUI 图标启动 Codex 时，也更有机会继承到正确的代理环境

2. 显式兜底启动
   - 如果正常从图标启动仍然会 reconnect，那么 `codex-proxy launch` 会在本次会话里显式注入代理环境变量再启动 Codex

之所以一定要这两层都做，是因为 GUI 环境变量继承在不同系统和不同会话下并不是 100% 可控的。

## 为什么我们选择这种做法

这个工具刻意选择了下面这些原则：

- 低侵入，而不是直接改系统全局代理
- 基于环境变量和用户级集成，而不是去改 Codex 内部逻辑
- 可回滚，而不是一次性粗暴覆盖系统状态

这样做的好处是：

- 回滚更容易
- 对系统其他软件影响更小
- 用户更容易理解“到底改了什么”
- 更适合做成一个小而专注的 CLI 工具

## 这样做的好处

- 能一条命令解决最常见的本地代理继承失败问题
- 修改前先诊断，而不是盲改
- 持久化修复不够时，还有显式兜底启动
- 支持保存环境变量快照，方便回滚
- 默认不改系统全局代理

## 边界和局限性

- macOS 的持久化修复是 best-effort，因为 GUI 应用环境变量继承是平台行为，不是这个工具可以完全控制的
- 即使检测到了本地代理，也不代表它的上游节点或出口 IP 一定足够稳定
- Windows / Linux 虽然已有代码路径，但现在还不应该视为正式可用
- 这个工具只覆盖一类高命中的 reconnecting 原因，不是所有 reconnecting 报告的万能修复器

## 支持矩阵

- `macOS`：alpha，已在当前仓库中本地验证
- `Windows`：代码路径存在，尚未正式测试
- `Linux`：代码路径存在，尚未正式测试

如果你现在对外发布 GitHub Release，建议这样描述：

> macOS-first alpha。Windows 和 Linux 仍在开发中，尚未正式验证。

## 安装方式

### 方式 1：下载 GitHub Release 产物

从 release 页面下载对应平台的压缩包，解压后把二进制放到你的 `PATH` 里。

macOS 示例：

```bash
tar -xzf codex-proxy_0.1.0-alpha.1_darwin_arm64.tar.gz
chmod +x codex-proxy
mv codex-proxy /usr/local/bin/codex-proxy
```

### 方式 2：从源码构建

```bash
go build -o ./bin/codex-proxy ./cmd/codex-proxy
```

## 快速使用

先诊断：

```bash
codex-proxy doctor --verbose
```

如果检测到了可用的本地代理，就执行修复：

```bash
codex-proxy fix
```

如果正常启动 Codex 仍然重连，就用显式兜底启动：

```bash
codex-proxy launch
```

查看当前状态：

```bash
codex-proxy status
```

移除修复并恢复之前捕获的环境状态：

```bash
codex-proxy unset
```

## 它会写入哪些内容

在 macOS 上，这个工具会写入类似下面这些文件：

- `~/Library/Application Support/codex-proxy-fix/state.json`
- `~/Library/Application Support/codex-proxy-fix/setenv.sh`
- `~/Library/Application Support/codex-proxy-fix/launch-codex.sh`
- `~/Library/LaunchAgents/com.codexproxyfix.env.plist`

它还会通过 `launchctl setenv` 更新当前登录会话中的代理环境变量。

## 回滚

`codex-proxy unset` 的目标是：

1. 卸载它管理的 `LaunchAgent`
2. 删除工具写入的文件
3. 如果存在快照，就恢复之前捕获到的持久化环境变量

如果某个变量原本不存在，`unset` 会把它从受管持久化环境中移除。

## 开发

运行测试：

```bash
go test ./...
```

构建 release 产物：

```bash
bash scripts/release/build.sh
```

产物和 `checksums.txt` 会生成在 `dist/` 目录下。

## 仓库文档

- [English README](./README.md)
- [实现蓝图](./CODEX_PROXY_FIX_IMPLEMENTATION_BLUEPRINT.md)
- [发布准备计划](./RELEASE_READINESS_PLAN.md)
- [首个版本 Release Notes](./RELEASE_NOTES_v0.1.0-alpha.1.md)
