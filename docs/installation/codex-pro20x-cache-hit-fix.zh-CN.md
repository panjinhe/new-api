# Codex-pro20x 缓存低命中排查与修复记录

这份文档记录一次真实的线上排障：`Codex-pro20x-1` 渠道可以正常调用，但同一段稳定前缀反复请求时，使用日志里的 `缓存读` 长时间为 `0`，导致费用明显偏高。

结论先说清楚：

- 不是 `prompt_cache_key` 丢了。
- 不是前缀内容每次都变了。
- 不是单纯“渠道不可用”。
- 真正原因是：直连 ChatGPT Codex 后端时，只保留 JSON 里的 `prompt_cache_key` 不够，还需要把它同步到上游请求头 `Session_id`。

修复后，`Codex-pro20x-1` 已经能在同一稳定前缀下返回 `cached_tokens`。

![缓存命中修复前后对比](/docs/codex-cache/cache-hit-chart.svg)

## 现象

最开始看到的现象很直观：

- `Codex-pro20x-1` 请求能成功。
- `prompt_tokens` 稳定在 3020 左右。
- `codex_prefix_hash` 连续相同。
- 但 `cache_tokens` 连续为 `0`。

这说明请求并不是失败，也不是内容每次都完全不一样，而是上游没有把这些请求识别成同一个可缓存会话。

修复前的线上日志样本：

| 日志 ID | 时间 | 渠道 | Prompt Tokens | Cache Tokens | Prefix Hash |
| --- | --- | ---: | ---: | ---: | --- |
| 3506 | 05:54:12 | 9 | 3020 | 0 | `cd2f13a6d76be645...` |
| 3507 | 05:54:20 | 9 | 3020 | 0 | `cd2f13a6d76be645...` |
| 3508 | 05:54:25 | 9 | 3020 | 0 | `cd2f13a6d76be645...` |
| 3509 | 05:54:29 | 9 | 3020 | 0 | `cd2f13a6d76be645...` |
| 3510 | 05:54:36 | 9 | 3020 | 0 | `cd2f13a6d76be645...` |
| 3511 | 05:54:43 | 9 | 3020 | 0 | `cd2f13a6d76be645...` |

这组数据的关键点是：`prefix_hash` 一样，但 `cache_tokens` 一直是 0。

## 排查过程

### 1. 先排除“请求前缀不稳定”

为了避免靠感觉判断，我先加了一个只给管理员看的诊断字段：

- `other.admin_info.codex_prefix_hash`
- `other.admin_info.codex_prefix_hash_basis`

这个 hash 不记录明文 prompt，只对归一化后的 Responses 请求语义做摘要。它会忽略 `prompt_cache_key`，但会包含真正影响上游前缀的字段，例如：

- `model`
- `input`
- `instructions`
- `tools`
- `tool_choice`
- `reasoning`
- `previous_response_id`

结果显示，同一批测试请求的 `codex_prefix_hash` 是稳定的，所以“前缀每次都变了”不是主因。

### 2. 再排除“只是 channel 9 不能调用”

`Codex-pro20x-1` 本身可以正常返回结果，响应里也有 usage：

```json
{
  "input_tokens": 3020,
  "input_tokens_details": {
    "cached_tokens": 0
  },
  "output_tokens": 26,
  "total_tokens": 3046
}
```

这说明不是普通连通性问题，而是上游认为这次请求没有命中 prompt cache。

### 3. 对照 channel 7 和 channel 9

同样的请求强制走 `channel 7` 时，从第二轮开始可以命中缓存：

| 渠道 | 第 1 轮 | 后续稳定值 | 说明 |
| --- | ---: | ---: | --- |
| channel 7 | 0 | 2816 | 可命中缓存 |
| channel 9 | 0 | 0 | 修复前连续不命中 |

这个对照很重要：它说明兼容层不是全局坏掉，否则 channel 7 也不会命中。

### 4. 证伪 `store=false` 假设

`Codex` 直连适配器里会把 `store` 设为 `false`。我临时删除这个字段测试过，结果上游直接返回：

```text
Store must be set to false
```

所以 `store=false` 不是缓存低命中的原因，反而是直连 Codex 后端的必需字段。

### 5. 找到真正差异：`Session_id`

继续对比后发现，直连 ChatGPT Codex 后端不只是看 JSON 里的 `prompt_cache_key`，还需要上游请求头里有稳定的 `Session_id`。

临时把 `prompt_cache_key` 同步到 `Session_id` 后，`channel 9` 立刻从连续 0 命中变成开始返回 `cached_tokens=2816`。

这一步确认了根因：

```text
请求 JSON 有 prompt_cache_key
但直连 Codex 上游没有 Session_id
=> 上游没有按稳定会话复用 prompt cache
```

## 最终修复

修复点在 `Codex` 直连适配器里完成，不依赖后台手工配置。

核心逻辑：

```text
如果请求里存在 prompt_cache_key
并且上游 header 里还没有 session_id
则设置：

Session_id = prompt_cache_key
```

同时保留一个原则：如果客户端已经显式传了 `Session_id`，就不覆盖用户传入的值。

修复链路如下：

![修复链路](/docs/codex-cache/fix-flow.svg)

关键提交：

- `6296cec41 fix codex prompt cache session header`

关键文件：

- `relay/channel/codex/adaptor.go`
- `relay/channel/codex/adaptor_test.go`

## 修复后数据

上线后强制 `channel 9` 跑了 8 轮，同一稳定前缀、同一 `prompt_cache_key` 语义下，结果如下：

| 日志 ID | 时间 | 渠道 | Prompt Tokens | Cache Tokens | 费用 |
| --- | --- | ---: | ---: | ---: | ---: |
| 3589 | 14:05:55 | 9 | 3020 | 0 | `$0.004053` |
| 3590 | 14:06:00 | 9 | 3020 | 2816 | `$0.000757` |
| 3592 | 14:06:04 | 9 | 3020 | 0 | `$0.004053` |
| 3593 | 14:06:08 | 9 | 3020 | 2816 | `$0.000802` |
| 3595 | 14:06:12 | 9 | 3020 | 2816 | `$0.000802` |
| 3596 | 14:06:15 | 9 | 3020 | 2816 | `$0.000787` |
| 3598 | 14:06:24 | 9 | 3020 | 2816 | `$0.000765` |
| 3599 | 14:06:28 | 9 | 3020 | 2816 | `$0.000765` |

使用日志截图如下：

![使用日志截图](/docs/codex-cache/usage-log-screenshot.svg)

这组数据说明：

- 修复前：6 轮连续 `cache_tokens=0`。
- 修复后：8 轮里 6 轮返回 `cache_tokens=2816`。
- 命中时缓存读比例约为 `2816 / 3020 = 93.2%`。
- 单次费用从约 `$0.0040` 降到约 `$0.0008`。

仍然可能出现个别 `cache_tokens=0`。这通常是上游 prompt cache 的冷启动、缓存写入延迟或偶发未命中，不等于本地代码再次失效。判断是否复发时，不要看单条日志，要看同一 `prompt_cache_key`、同一 `codex_prefix_hash` 下连续多轮是否都为 0。

## 后续排查口径

以后如果再遇到“缓存命中突然变低”，建议按这个顺序查：

1. 看 `cache_tokens` 是否只是偶发为 0，还是连续多轮为 0。
2. 看 `codex_prefix_hash` 是否稳定。
3. 看 `prompt_cache_key` 是否稳定。
4. 看请求是否实际落在同一个渠道和同一个上游账号。
5. 对 Codex 直连渠道，确认 `prompt_cache_key` 已同步到上游 `Session_id`。

最重要的一条判断：

```text
prefix_hash 稳定 + prompt_cache_key 稳定 + channel 9 连续 cache_tokens=0
=> 优先检查 Codex 上游会话头，而不是先怀疑用户 prompt。
```

## 经验

这次问题容易误判成“渠道 9 本身不行”。实际上更准确的说法是：

- channel 9 能调用。
- channel 9 的上游也支持缓存。
- 但直连 Codex 后端需要更接近 Codex CLI 的会话语义。

`prompt_cache_key` 是请求体层面的缓存 key；`Session_id` 是 Codex 上游识别同一会话的重要头。两者同步后，缓存命中才恢复到可接受状态。

