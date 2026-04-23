# Claude Code 接入 Codex/GPT-5.4 改造与排障记录

这份文档用来回答四个问题：

- 这次为了让 `Claude Code` 接到本项目，我们到底做了什么。
- 最终是怎么实现“前端兼容 Claude Code，后端统一走 GPT-5.4”的。
- 中间踩过哪些坑，为什么会踩。
- 以后如果再做一遍，什么做法最稳。

如果你是第一次接触这套链路，可以先记住一句话：

- `Claude Code` 说的是 Anthropic/Claude 协议。
- 我们的后端真正要打的是 `GPT-5.4`。
- 所以中间必须做一层“协议兼容 + 模型别名映射”。

## 一句话目标

把 `Claude Code` 伪装成在请求 Claude，实际上让请求进入 `new-api`，再由 `new-api` 统一转发到 `GPT-5.4`。

## 最终实现效果

最终做成的是下面这条链路：

```text
Claude Code
  -> /v1/messages（Anthropic 风格）
  -> new-api 兼容层
  -> 模型别名 claude-opus-4-6 -> gpt-5.4
  -> 上游 OpenAI / Responses
  -> new-api 再转换回 Claude Code 能理解的格式
```

从用户视角看，效果是：

- 前端给 `Claude Code` 导出的是 Claude 风格配置。
- 用户在 `Claude Code` 里看到的模型名可以是 `claude-opus-4-6`。
- 后端真正选路和计费时，实际使用的是 `gpt-5.4`。

## 这次具体做了什么

### 1. 补齐 `/v1/messages` 兼容层

这是整件事的基础。

`Claude Code` 默认走的是 Anthropic 的 `/v1/messages` 接口，不是 OpenAI 的 `/v1/chat/completions`。  
如果后端只会说 OpenAI 方言，`Claude Code` 就算拿到了地址和 key，也会直接报错。

这次做的兼容主要有三块：

- 兼容非流式返回体
- 兼容流式 SSE 结束事件
- 在上游是 `Responses API` 时，能正确从 SSE 里提取最终文本

关键文件：

- `relay/channel/openai/chat_via_responses.go`
- `relay/channel/openai/relay_responses.go`
- `service/convert.go`
- `service/http.go`

对应提交：

- `f84b54e07 fix: complete claude messages local compatibility`

简单理解：

- `Claude Code` 期待的是 Claude 风格的“收尾动作”。
- 上游 `Responses API` 返回的是另一套事件流。
- 我们做的事情，就是把上游的流重新“翻译”成 Claude Code 真正听得懂的话。

### 2. 增加模型别名：`claude-opus-4-6 -> gpt-5.4`

这一步解决的是“客户端看到什么模型名”和“后端实际打什么模型”的问题。

为了让前端和 `Claude Code` 看起来更自然，我们给它一个固定别名：

- 对外显示：`claude-opus-4-6`
- 后端实际路由：`gpt-5.4`

关键文件：

- `service/claude_code_alias.go`
- `middleware/distributor.go`

核心逻辑很简单：

- 进入分发器前，先检查模型名是不是 `claude-opus-4-6`
- 如果是，就在后端内部改写成 `gpt-5.4`
- 后面的模型限额、选路、转发都按 `gpt-5.4` 走

对应提交：

- `45e9acd21 feat: add claude code ccswitch compatibility`

这一步的好处是：

- 前端和工具配置更统一
- 不需要让普通用户理解“为什么 Claude Code 配的是 GPT-5.4”
- 后端仍然可以保持一套真实上游模型映射

### 3. 改造 Token 表里的 `CCSwitch` 导出

为了让用户少手填配置，这次还改了前端的 `CCSwitch` 导出逻辑。

关键文件：

- `web/src/components/table/tokens/modals/CCSwitchModal.jsx`

主要改动有三类：

- Claude 模式下固定导出 `claude-opus-4-6`
- 导出 Claude 风格的环境变量
- 根据排障结果调整 `CCSwitch` deep link 参数

这部分经历了多次微调，对应提交包括：

- `45e9acd21 feat: add claude code ccswitch compatibility`
- `1130be129 fix: include api key in claude ccswitch config`
- `2604e73c5 fix: remove auth token from claude ccswitch config`
- `18a673cd6 fix: avoid api key override in claude ccswitch import`
- `30c524f78 fix: restore api key params for claude ccswitch import`

为什么会有这么多小修？

因为 `CCSwitch` 对 Claude 的导入行为并不像看起来那么“直给”：

- 有些字段它会直接用
- 有些字段它会自己二次生成
- 去掉某些 URL 参数后，它会直接报 `API key is required`
- 保留某些参数后，它又可能把配置写成 `ANTHROPIC_AUTH_TOKEN`

所以这部分不是单纯“按文档填字段”，而是边验证边修。

### 4. 用本地 Docker 全链路验证

这次没有一上来就上生产，而是先在本地 Docker 跑通。

本地验证重点包括：

- 前端构建是否正常
- 后端 `/v1/messages` 是否能返回 Claude Code 能接受的响应
- `claude-opus-4-6` 是否真的被映射到 `gpt-5.4`
- `Claude Code` 命令行是否能返回 `OK`

典型验证方式：

```powershell
$env:ANTHROPIC_BASE_URL='http://127.0.0.1:3000'
$env:ANTHROPIC_API_KEY='你的测试 token'
$env:ANTHROPIC_MODEL='claude-opus-4-6'
$env:ANTHROPIC_DEFAULT_HAIKU_MODEL='claude-opus-4-6'
$env:ANTHROPIC_DEFAULT_OPUS_MODEL='claude-opus-4-6'
$env:ANTHROPIC_DEFAULT_SONNET_MODEL='claude-opus-4-6'
$env:ANTHROPIC_REASONING_MODEL='claude-opus-4-6'
claude -p --setting-sources local --model claude-opus-4-6 --tools "" --max-budget-usd 0.05 "Reply exactly with OK"
```

如果最后返回 `OK`，说明整条链路至少本地是通的。

### 5. 按运维手册重新部署生产

生产不是直接在服务器上临时 `go build`，而是按项目现有运维手册走：

- 本地重建前端
- 本地编译 Linux `amd64` 二进制
- 校验产物必须是 Linux `ELF`
- 把源码包和二进制上传到服务器
- 替换容器内 `/new-api`
- 重启并检查健康状态

关键文档：

- `ops/README.md`
- `docs/installation/deployment-workflow.zh-CN.md`

关键脚本：

- `scripts/build-linux-release.ps1`

常用命令：

```powershell
pwsh ./scripts/build-linux-release.ps1
```

这一步最大的价值是：

- 能同时更新前端和后端
- 不用整镜像来回搬运
- 能显式避免错误平台二进制被发到 Linux 容器里

## 遇到过什么困难，怎么解决的

## 困难 1：`Claude Code` 报 `Auth conflict`

典型现象：

- `Claude Code` 提示同时存在：
  - `ANTHROPIC_AUTH_TOKEN`
  - `/login managed key`

问题本质：

- 这不是后端协议错了
- 是本机 `Claude Code` 已经登录过官方账号
- 同时又从配置文件或 `CCSwitch` 导入了 token
- 客户端不知道该优先用谁，于是报冲突

解决方式：

1. 退出 Claude 本机登录态

```powershell
claude auth logout
```

2. 改用 `ANTHROPIC_API_KEY`，不要再优先依赖 `ANTHROPIC_AUTH_TOKEN`

3. 必要时使用 `--bare`

```powershell
claude --bare
```

`--bare` 的意义很重要：

- 它会忽略 `/login`
- 忽略 keychain
- 只使用你明确传入的 `ANTHROPIC_API_KEY`

对排障特别有帮助。

## 困难 2：`CCSwitch` Claude 导入并不稳定

这次排障里最绕的一段，就是 `CCSwitch` 对 Claude 的导入。

我们先后碰到过两种相反的问题：

### 情况 A：去掉 `apiKey` 后，CCSwitch 报导入失败

典型报错：

- `无效输入: API key is required (either in URL or config file)`

这说明：

- 对 Claude provider 来说，`CCSwitch` 并不总是只认 `config`
- 它有时还要求 URL 上带 `apiKey`

解决方式：

- 恢复 `endpoint` 和 `apiKey` URL 参数

对应提交：

- `30c524f78 fix: restore api key params for claude ccswitch import`

### 情况 B：保留 `apiKey` 后，临时文件又变成 `ANTHROPIC_AUTH_TOKEN`

这说明：

- `CCSwitch` 对 Claude 导入很可能会自己重新组装配置
- 它不一定完全照搬我们传进去的 `config.env`

所以这部分的最终结论不是“网站全错”或者“客户端全错”，而是：

- `CCSwitch` 在 Claude 这条链路上有自己的实现细节
- 不能简单假设 `config` 里写什么，最终本地文件就是什么

最终经验：

- `Codex` 走 `CCSwitch` 比较直接
- `Claude` 如果想最稳，优先手工写 `settings.json`

## 困难 3：看起来像协议问题，实际是 token 无效

这次有一个很典型的误判风险：

- 前端能导出配置
- `Claude Code` 也会发请求
- 但请求最后返回 `401 Unauthorized`

实际原因不是协议兼容没做好，而是：

- 这枚 token 在对应网关上根本无效

也就是说，排障顺序一定要分两层：

1. 协议通不通
2. token 本身是否有效

如果第二层都没过，第一层改得再漂亮也没用。

## 困难 4：`Exec format error`

这是部署时非常典型的“平台不匹配”问题。

它的意思不是程序逻辑错了，而是：

- Linux 容器想执行一个不是 Linux 可执行格式的文件

最常见原因：

- 在 Windows 上编出了 `MZ` 头的 Windows 可执行文件
- 然后把它塞进 Linux 容器里运行

这次的解决方式是把发布流程固定为：

```powershell
pwsh ./scripts/build-linux-release.ps1
```

这个脚本做了两件关键保护：

- 强制 `GOOS=linux GOARCH=amd64`
- 编译后检查文件头是否为 `ELF`

如果不是 `ELF`，脚本会直接报错并阻止上线。

对新手来说，可以这样理解：

- `Exec format error` 不是“代码语法错了”
- 而是“把 Windows 程序拿去 Linux 上运行了”

## 困难 5：生产容器重启后出现 `no route to host`

这次线上发布还碰到过一次比较隐蔽的运维问题：

- `new-api-prod` 重启后连不上 `postgres`
- 日志里报 `no route to host`

这个问题最容易被误判成：

- 新版本代码把数据库连坏了

但实际根因是：

- 宿主机 Docker bridge 网络状态异常

为什么能确认不是代码问题？

- PostgreSQL 容器内部是正常的
- 同网段临时容器去连 `postgres:5432` 也失败
- 说明坏的是 Docker 网络，不是应用逻辑

最终解决方式：

1. 先重建 compose 网络和容器
2. 如果还不恢复，重启宿主机 Docker 服务
3. 再重新拉起 compose

这是一次很典型的“不要把一切数据库错误都归咎于代码改动”的案例。

## 最终推荐做法

如果目标是“让 Claude Code 稳定走本项目，再由后端统一打到 GPT-5.4”，最稳的做法是下面这套。

### 1. 手工配置优先，不要过度依赖 Claude 的 CCSwitch 导入

最稳配置文件示例：

路径：

- `%USERPROFILE%\\.claude\\settings.json`

示例：

```json
{
  "env": {
    "ANTHROPIC_API_KEY": "你的有效 token",
    "ANTHROPIC_BASE_URL": "https://你的网关地址",
    "ANTHROPIC_DEFAULT_HAIKU_MODEL": "claude-opus-4-6",
    "ANTHROPIC_DEFAULT_OPUS_MODEL": "claude-opus-4-6",
    "ANTHROPIC_DEFAULT_SONNET_MODEL": "claude-opus-4-6",
    "ANTHROPIC_MODEL": "claude-opus-4-6",
    "ANTHROPIC_REASONING_MODEL": "claude-opus-4-6"
  }
}
```

重点是：

- 用 `ANTHROPIC_API_KEY`
- 尽量不要再用 `ANTHROPIC_AUTH_TOKEN`

### 2. 如果本机登录过 Claude，先退出登录态

```powershell
claude auth logout
```

否则很容易出现：

- 你以为自己在测试网关 token
- 实际上客户端还在偷偷读 `/login managed key`

### 3. 排障时优先用 `--bare`

```powershell
claude --bare -p --model claude-opus-4-6 --tools "" "Reply exactly with OK"
```

这样做的意义是：

- 把干扰项降到最低
- 只验证你当前配置是否真能直连网关

### 4. 先测 token，再怀疑协议

最容易浪费时间的做法是：

- 配置一通改
- 最后发现 token 其实是无效的

正确顺序应该是：

1. 直接打 `/v1/messages`
2. 看是不是 `401 Invalid token`
3. 先确认 key 可用，再调 `Claude Code`

### 5. 生产发布优先走“本地构建 + 二进制热更新”

原因很简单：

- 这项目前端是嵌进 Go 二进制的
- 你这套环境已经有成熟的二进制优先发布流程
- 比整镜像传输更快，也更稳定

## 这次改动对应的关键提交

- `f84b54e07 fix: complete claude messages local compatibility`
- `45e9acd21 feat: add claude code ccswitch compatibility`
- `1130be129 fix: include api key in claude ccswitch config`
- `2604e73c5 fix: remove auth token from claude ccswitch config`
- `18a673cd6 fix: avoid api key override in claude ccswitch import`
- `30c524f78 fix: restore api key params for claude ccswitch import`

## 给小白的最终总结

如果把这次事情压缩成最容易理解的一版，其实就是：

1. `Claude Code` 默认不会说 OpenAI 语言，所以后端要补 `/v1/messages` 兼容层。
2. 为了让用户看起来像在用 Claude，我们把 `claude-opus-4-6` 这个名字映射到后端真实的 `gpt-5.4`。
3. `CCSwitch` 在 Claude 导入上有自己的行为，不一定完全照搬我们传进去的配置，所以不要把它当成绝对可信的“直通器”。
4. 真正最稳的方案，通常还是：
   - 手工写 `settings.json`
   - 用 `ANTHROPIC_API_KEY`
   - 必要时先 `claude auth logout`
   - 排障时用 `claude --bare`
5. 如果线上报错，不要只盯代码。  
   这次我们就同时遇到了：
   - token 无效
   - `Exec format error`
   - Docker 网络异常

所以一条成熟的结论是：

- 协议问题、配置问题、凭证问题、构建问题、运维问题，必须分层排查。

