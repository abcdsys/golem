# AI 插件

基于 OpenAI 兼容接口的智能对话插件，支持在微信群聊和私聊中进行 AI 对话。

## 功能特性

- **OpenAI 兼容接口**：支持所有兼容 OpenAI API 格式的大模型服务（OpenAI、DeepSeek、通义千问、Kimi 等）
- **多提示词管理**：支持创建和切换多个提示词配置，满足不同场景需求
- **智能上下文**：自动维护对话上下文，支持连续对话
- **灵活回复策略**：
  - 私聊：默认自动回复所有消息
  - 群聊：支持 @提及、引用回复、概率回复
- **完善的配置管理**：通过命令行实时调整配置，无需重启

## 快速开始

### 1. 配置 AI 服务

使用 `/ai set` 命令配置 AI 接口：

```bash
# 配置 DeepSeek
/ai set -u https://api.deepseek.com/v1 -k sk-xxx -m deepseek-chat

# 配置 OpenAI
/ai set -u https://api.openai.com/v1 -k sk-xxx -m gpt-4

# 配置通义千问
/ai set -u https://dashscope.aliyuncs.com/compatible-mode/v1 -k sk-xxx -m qwen-plus
```

### 2. 设置提示词（可选）

```bash
# 使用默认提示词
/ai prompt default

# 创建自定义提示词
/ai prompt roleplay -p "你是一个幽默风趣的聊天机器人"

# 创建专业助手提示词
/ai prompt assistant -p "你是一个专业的技术助手，擅长解答编程问题"
```

### 3. 开始对话

- **私聊**：直接发送消息即可
- **群聊**：@机器人或引用机器人的消息

## 命令说明

### `/ai get`

查看当前 AI 配置。

**示例输出**：
```
AI 配置：
url=https://api.deepseek.com/v1
api_key=sk-ab****wxyz
model=deepseek-chat
active_prompt=default
prompts=assistant,default,roleplay
reply_rate=0.1
max_context=200
timeout=60
prompt=你是一个友好的助手
```

### `/ai set`

更新 AI 配置。

**参数**：
- `-u, --url`：OpenAI 兼容接口地址
- `-k, --api-key`：API 密钥
- `-m, --model`：模型名称
- `-r, --reply-rate`：群聊普通消息回复概率（0~1）
- `--max-context`：每个会话保留的最大上下文消息数
- `--timeout`：请求超时时间（秒）

**示例**：
```bash
# 完整配置
/ai set -u https://api.deepseek.com/v1 -k sk-xxx -m deepseek-chat

# 仅更新模型
/ai set -m gpt-4-turbo

# 调整群聊回复概率为 20%
/ai set -r 0.2

# 增加上下文长度
/ai set --max-context 300

# 设置超时时间为 120 秒
/ai set --timeout 120
```

### `/ai prompt`

切换或设置提示词。

**用法**：
```bash
/ai prompt <name> [-p prompt]
```

**参数**：
- `name`：提示词名称（必需）
- `-p, --prompt`：提示词内容（可选，提供则创建/更新提示词）

**示例**：
```bash
# 切换到已存在的提示词
/ai prompt default

# 创建新提示词
/ai prompt tech -p "你是一个技术专家"

# 更新现有提示词
/ai prompt tech -p "你是一个全栈开发专家，精通前后端技术"
```

### `/ai clear-context`

清理对话上下文。

**用法**：
```bash
/ai clear-context [-t target]
```

**参数**：
- `-t, --target`：会话标识（不传则清理当前会话）

**示例**：
```bash
# 清理当前会话上下文
/ai clear-context

# 清理指定群聊上下文
/ai clear-context -t chatroom::123456@chatroom

# 清理指定私聊上下文
/ai clear-context -t private::wxid_abc123
```

## 配置详解

### 配置文件

配置文件位于 `~/.golem/plugins/ai.toml`，首次启动时自动生成。

**完整配置示例**：
```toml
# OpenAI 兼容接口地址
base_url = "https://api.deepseek.com/v1"

# API 密钥
api_key = "sk-xxx"

# 模型名称
model = "deepseek-chat"

# 当前使用的提示词名称
active_prompt = "default"

# 群聊普通消息回复概率（0~1）
reply_rate = 0.1

# 每个会话最多保留的上下文消息数
max_context_messages = 200

# 请求超时时间（秒）
http_timeout_seconds = 60

# 提示词映射
[prompts]
default = "你是一个友好的助手"
roleplay = "你是一个幽默风趣的聊天机器人"
assistant = "你是一个专业的技术助手"
```

### 回复策略

#### 私聊模式
- 自动回复所有消息
- 不受 `reply_rate` 参数影响

#### 群聊模式
机器人在以下情况下会回复：

1. **被 @提及**：消息中 @了机器人（必定回复）
2. **被引用**：引用了机器人的消息（必定回复）
3. **普通消息**：根据 `reply_rate` 概率回复

**reply_rate 说明**：
- `0.0`：不回复群聊普通消息（仅响应 @和引用）
- `0.1`：10% 概率回复群聊普通消息（默认值）
- `1.0`：回复所有群聊消息
- 建议值：`0.05` ~ `0.2`，避免过于频繁打扰群聊

### 上下文管理

- 每个会话（群聊或私聊）独立维护上下文
- 默认保留最近 200 条消息（可通过 `max_context_messages` 调整）
- 超出限制时自动删除最早的消息
- 使用 `/ai clear-context` 可手动清理上下文

### 提示词系统

提示词由两部分组成：

1. **预制提示词**（自动添加）：
   - 基本约束（中文回复、消息格式等）
   - 主人信息（创建者的 username 和 nickname）
   - 微信聊天特定规则

2. **用户提示词**（通过 `prompts` 配置）：
   - 可定义多个提示词
   - 通过 `active_prompt` 指定当前使用的提示词
   - 通过 `/ai prompt` 命令动态切换

## 常见使用场景

### 场景 1：个人助手

```bash
# 配置个人助手提示词
/ai prompt assistant -p "你是我的个人助手，帮我处理日常事务、回答问题、提供建议"
```

### 场景 2：技术支持

```bash
# 配置技术支持提示词
/ai prompt tech -p "你是一个技术支持专家，擅长解答编程、系统配置、网络等技术问题。回答要准确、详细，并提供代码示例"
```

### 场景 3：创意写作

```bash
# 配置创意写作提示词
/ai prompt creative -p "你是一个富有创造力的写作助手，擅长构思故事、撰写文案、创作诗歌。风格活泼有趣，用词生动"
```

### 场景 4：角色扮演

```bash
# 配置角色扮演提示词
/ai prompt cat -p "你是一只可爱的小猫，说话带有喵喵声，性格调皮但温柔。你喜欢撒娇，偶尔会用爪子挠人"
```

### 场景 5：群聊娱乐

```bash
# 配置娱乐机器人
/ai prompt fun -p "你是一个活跃的群聊成员，幽默风趣，喜欢开玩笑。但要注意分寸，不要过于打扰他人"

# 设置较低的回复概率，避免刷屏
/ai set -r 0.05
```

## 支持的大模型服务

### DeepSeek（推荐）

```bash
/ai set -u https://api.deepseek.com/v1 -k sk-xxx -m deepseek-chat
```

### OpenAI

```bash
/ai set -u https://api.openai.com/v1 -k sk-xxx -m gpt-4
```

### 通义千问

```bash
/ai set -u https://dashscope.aliyuncs.com/compatible-mode/v1 -k sk-xxx -m qwen-plus
```

### Kimi

```bash
/ai set -u https://api.moonshot.cn/v1 -k sk-xxx -m moonshot-v1-8k
```

### 智谱 AI

```bash
/ai set -u https://open.bigmodel.cn/api/paas/v4 -k xxx -m glm-4
```

### 本地模型（Ollama）

```bash
/ai set -u http://localhost:11434/v1 -k ollama -m llama2
```

## 注意事项

1. **API 密钥安全**：
   - 配置文件包含敏感信息，注意保护
   - 使用 `/ai get` 查看时，密钥会自动脱敏显示

2. **成本控制**：
   - 注意 API 调用费用
   - 合理设置 `reply_rate` 避免不必要的调用
   - 适当限制 `max_context_messages` 减少 token 消耗

3. **响应时间**：
   - 默认超时 60 秒，可根据模型响应速度调整
   - 网络不稳定时可适当增加超时时间

4. **上下文管理**：
   - 上下文过长会增加 token 消耗和响应时间
   - 定期清理不需要的会话上下文
   - 切换提示词时会自动清理当前会话上下文

5. **群聊礼仪**：
   - 避免设置过高的 `reply_rate`，以免打扰群聊
   - 建议群聊 `reply_rate` 设置为 0.05 ~ 0.2
   - 私聊则无此限制

## 故障排查

### 1. 机器人不回复

**检查清单**：
- 确认 API 配置正确（`/ai get`）
- 检查 API 密钥是否有效
- 确认网络连接正常
- 查看日志是否有错误信息

**群聊不回复**：
- 确认消息中是否 @了机器人
- 检查 `reply_rate` 是否过低
- 尝试引用机器人的消息

### 2. 返回错误信息

**常见错误**：
- `AI base_url 未配置`：使用 `/ai set -u` 设置接口地址
- `AI api_key 未配置`：使用 `/ai set -k` 设置密钥
- `AI model 未配置`：使用 `/ai set -m` 设置模型
- `请求 AI 接口失败`：检查网络和接口地址
- `AI 接口返回错误`：检查 API 密钥和配额

### 3. 回复内容不符合预期

**解决方法**：
- 优化提示词内容
- 清理上下文后重试（`/ai clear-context`）
- 尝试切换不同的模型

### 4. 响应速度慢

**优化建议**：
- 减少 `max_context_messages` 数量
- 选择响应更快的模型
- 增加 `timeout` 避免超时

## 开发信息

- **插件名称**：ai
- **版本**：1.0.0
- **作者**：ovo
- **SDK 版本**：golem/sdk v0.1.1

## 更新日志

### v1.0.0
- 初始版本发布
- 支持 OpenAI 兼容接口
- 支持多提示词管理
- 支持上下文维护
- 支持群聊和私聊
- 支持命令行配置管理