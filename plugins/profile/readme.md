# Profile 插件（群成员人物画像）

基于群成员的历史发言，经 AI 生成/增量更新「人物画像」的独立插件。
通过**发消息触发**；历史发言经 `statistics.query_messages` 能力获取（不直接读 statistics.db），
LLM 经 `ai.chat`，画像渲染成图片经 `text.to.image`；自身管理 `profile.db`。

## 功能特性

- **发消息触发**：群里 / 私聊发「人物画像 <昵称>」即生成或更新画像
- **跨插件协作**：历史发言来自 `statistics` 插件（`statistics.query_messages`），LLM 复用 `ai` 插件（`ai.chat`），图片渲染用 `gg` 插件（`text.to.image`）
- **双范围**：默认按当前群；`--global` 跨群全局画像
- **增量更新**：水位线 `last_msg_id`，只分析新增发言
- **冷启动切片**：按 token 预算切块、Map-Reduce，超限丢弃最早块
- **图片输出**：默认渲染图片发送，`render_image=false` 发纯文本
- **权限**：群聊任意成员可查群成员；私聊仅主人；主人画像仅主人可查

## 依赖（运行需要）

| 插件 | 提供能力 | 作用 |
|---|---|---|
| `statistics` | `statistics.query_messages` | 查询成员历史发言 |
| `ai` | `ai.chat` | 调用大模型 |
| `gg` | `text.to.image` | 画像文本渲染成 PNG（可关，回退纯文本） |

## 触发形式

| 消息 | 范围 | 说明 |
|------|------|------|
| `人物画像` | 群聊 | 查自己（未指定成员时默认发起人自己） |
| `人物画像` | 私聊 | 仅主人可用，查自己 |
| `人物画像 张三` | 当前群 | 按昵称/群显示名/备注/wxid 匹配 |
| `人物画像 @张三` | 当前群 | @ 提人，用 Reminds wxid 直接定位（最可靠） |
| `人物画像@张三` | 当前群 | 同上（@ 直接跟，无空格也触发） |
| `人物画像 张三 --global` | 跨群 | 全局画像（按联系人缓存匹配，多为好友） |
| `人物画像 张三 --rebuild` | 当前群 | 忽略已有画像，从头冷启动 |

> 前缀「人物画像」后必须紧跟**空白**或**@**才触发（兼容微信 @ 提人的非常规空格如 NBSP）；
> 「人物画像张三」「人物画像功能真不错」不触发，放行给 ai。

## 权限

| 场景 | 可查询范围 |
|------|-----------|
| 群聊 | 任意群成员均可触发，仅限当前群成员 |
| 私聊 | **仅主人**可查（需配置 `Owner`） |
| 主人画像 | 仅主人本人可查 |

## 配置（可选）

零配置即可运行；如需调节，编辑 `plugins/config.toml` 的 `[profile.config]` 段（首次运行由宿主自动生成默认值）：

```toml
[profile.config]
chunk_token_budget = 4000       # 每块 token 估算上限；中文模型可调大到 6000
max_single_msg_chars = 2000     # 单条消息硬截断长度
cold_start_max_messages = 2000  # 冷启动最多处理的消息条数
cold_start_max_chunks = 30      # 冷启动最多分块数（超出丢弃最早的）
render_image = true             # 画像是否渲染成图片发送；false 发纯文本
```

留空（或值为 0）回退默认值。`render_image` 默认 `true`。

## 工作原理

```
收到「人物画像」消息事件
   → 解析触发语 + @ 的 wxid → 权限/范围判定
   → 调 statistics.query_messages 取历史发言（JSON）
   → 按 token 切块 → 每块调 ai.chat 产出「局部观察」
   → 合并(已有画像 + 观察 + 量化指标) → 一次 ai.chat 产出完整画像
   → 存 profile.db 的 profiles 表（更新水位线）→ 经 text.to.image 渲染图片发送
```

- **异步生成**：冷启动耗时长，触发后立即消费事件、后台 goroutine 生成，避免事件分发超时（1 分钟）导致 ai 重复回复
- **增量水位线**：`profiles.last_msg_id`，只拉取 `id > last_msg_id` 的新发言
- **无新发言**：直接返回已有画像，不空跑 AI
- **消息过多**：查询层取最近 `cold_start_max_messages` 条；切块层超 `cold_start_max_chunks` 丢弃最早块

## ⚠️ Token 用量提醒

冷启动最费 token：最多 30 块 × 每块 ~4000 token + 一次合并 ≈ **12 万+ input tokens / 人**。
建议冷启动用便宜模型（DeepSeek/Qwen，靠 `ai` 插件配置），冷启动后日常增量更新（通常 1 次调用），
不要频繁 `--rebuild`。多人/多群会线性放大成本。

## 数据存储

- `profile.db`：画像库（`profiles` 表），与插件 exe 同目录（`plugins/`）
  ```sql
  CREATE TABLE profiles (
      chatroom  TEXT NOT NULL DEFAULT '',   -- 群 wxid；全局画像为空
      member    TEXT NOT NULL,              -- 成员 wxid
      profile   TEXT,                       -- 画像文本
      last_msg_id INTEGER DEFAULT 0,        -- 增量水位线
      updated_at DATETIME,                   -- 本地时间
      PRIMARY KEY (chatroom, member)
  );
  ```
- 历史发言数据**不在本插件**，由 `statistics` 插件管理（经能力查询）

清空画像：删 `profile.db` 即可，下次启动自动重建。

## 开发信息

- **插件名称**：profile
- **版本**：1.0.0
- **作者**：ovo
- **触发**：消息事件（`message::text`），Priority 0，命中即终止事件链（`Next=false`）
- **依赖**：`statistics` / `ai` / `gg` 插件

## 扩展阅读

- [Statistics 插件 readme](../statistics/readme.md)（提供 `statistics.query_messages` 能力）
- [Golem 插件开发指南](../../readme.md)
