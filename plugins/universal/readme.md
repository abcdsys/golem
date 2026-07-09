# Universal 插件

规则驱动的通用 API 请求与消息发送插件，通过配置关键词规则，实现灵活的消息触发、HTTP 请求和结果转发。

## 📋 功能特性

- **关键词触发**：根据消息中的关键词自动匹配规则并执行
- **模板变量系统**：支持丰富的变量替换（参数、引用消息、自定义变量等）
- **HTTP 请求**：支持 GET/POST/PUT/DELETE 等各种请求方法
- **结果提取**：使用 gjson 路径语法精确提取 JSON 响应内容
- **二进制媒体支持**：响应体本身即为图片/视频等二进制数据时，可直接发送无需二次请求；`result_path` 非空时自动从提取的 URL 进行二次 GET 下载媒体
- **多种消息类型**：支持发送文本、图片、视频、表情等多种消息
- **智能 @ 提及**：可自动 @ 引用消息的用户或参数中的用户
- **规则管理**：提供完整的规则增删改查、启用禁用命令

## 🚀 快速开始

### 基础示例

创建一个简单的随机图片规则：

```bash
/universal add random-pic -k 来张图,随机图片 -u "https://api.example.com/random/image" -t image -p data.url
```

使用方式：
```
来张图
```

### 带参数的示例

创建一个支持参数的 CP 配对规则：

```bash
/universal add cp -k cp配对,cp -u "https://api.example.com/cp?a={{arg_1}}&b={{arg_2}}" -t text
```

使用方式：
```
cp配对 张三 李四
```

### 带命名参数的示例

创建一个图片尺寸控制规则：

```bash
/universal add pixiv -k pixiv,p站 -u "https://api.example.com/pixiv?width={{width}}&height={{height}}" -t image -p data.url
```

使用方式：
```
pixiv width=800 height=600
```

## 🎯 工作原理

### 消息匹配流程

1. **接收消息**：插件接收文本消息或引用消息
2. **解析消息**：将消息按第一个空格分割为 `keyword` 和 `param`
3. **匹配规则**：使用 `keyword` 进行全文匹配查找对应规则
4. **执行请求**：使用模板变量替换 URL、请求头、请求体后发送 HTTP 请求
5. **结果处理**：根据 `send_type` 与 `result_path` 的组合决定如何处理响应（见下文）
6. **发送消息**：根据规则配置的消息类型发送结果

### send_type × result_path 双控语义

请求得到响应体 `body` 后，按 `send_type` 与 `result_path` 是否为空分四种情况：

| send_type | result_path | 行为 |
|-----------|-------------|------|
| `text` | 空 | 将 `body` 原文作为文本发送 |
| `text` | 非空 | 用 gjson 从 `body` 提取字段，作为文本发送 |
| `image`/`video`/`emoji` | 空 | `body` 本身即二进制媒体，直接发送（不再二次请求） |
| `image`/`video`/`emoji` | 非空 | 用 gjson 从 `body` 提取 URL，复用首次请求的头部进行二次 GET，第二个 `body` 即待发送的二进制媒体 |

> 该设计修复了 historic bug：当 API 直接返回二进制（如 `http://api.yujn.cn/api/haibian.php` 返回视频二进制）时，旧逻辑会把二进制当作 URL 再请求一次。现在只要 `send_type` 为媒体且 `result_path` 为空，即把首次响应体直接作为媒体发送。

### 参数解析规则

消息格式：`<keyword> [params...]`

参数分为两种：

1. **命名参数**：`name=value` 格式，生成 `{{name}}` 变量
2. **位置参数**：普通文本，按顺序生成 `{{arg_1}}`、`{{arg_2}}` 等变量

特殊处理：
- 参数前的 `@` 符号会被自动移除（用于 @ 用户）
- 存在引用消息时，被引用者自动成为 `{{arg_1}}`，其他参数后移

### 引用消息处理

当消息引用了其他消息时：
- `{{quoter}}`：被引用者的昵称
- `{{quote}}`：被引用的消息内容
- 被引用者自动成为 `{{arg_1}}`，原参数顺序后移

## 📝 模板变量

### 内置变量

| 变量 | 说明 | 示例 |
|------|------|------|
| `{{text}}` | 完整消息内容 | `pixiv width=800 height=600` |
| `{{keyword}}` | 匹配的关键词 | `pixiv` |
| `{{param}}` | 参数部分（不含关键词） | `width=800 height=600` |
| `{{quote}}` | 引用的消息内容 | `这是被引用的消息` |
| `{{quoter}}` | 被引用者昵称 | `张三` |
| `{{arg_1}}`, `{{arg_2}}`, ... | 位置参数 | 第 1、2... 个参数 |

### 自定义变量

通过 `name=value` 格式创建：
```
pixiv width=800 height=600
```
生成变量：`{{width}}=800`、`{{height}}=600`

### 变量使用场景

- **URL 模板**：`https://api.example.com/image?size={{arg_1}}&format={{format}}`
- **请求头模板**：`Authorization=Bearer {{token}};Content-Type=application/json`
- **请求体模板**：`{"user": "{{arg_1}}", "message": "{{param}}"}`

## ⚙️ 配置说明

### 规则配置项

| 配置项 | 类型 | 必填 | 默认值 | 说明 |
|--------|------|------|--------|------|
| `id` | string | ✅ | - | 规则唯一标识 |
| `keywords` | []string | ✅ | - | 触发关键词列表 |
| `url` | string | ✅ | - | 请求地址模板 |
| `method` | string | ❌ | `GET` | 请求方法 |
| `headers` | string | ❌ | - | 请求头，格式：`A=B;C=D` |
| `body` | string | ❌ | - | 请求体模板 |
| `send_type` | string | ❌ | `text` | 发送类型：`text`\|`emoji`\|`image`\|`video` |
| `result_path` | string | ❌ | - | gjson 路径，空则使用响应原文（媒体类型空路径时直接发送响应体二进制） |
| `at` | bool | ❌ | `false` | 是否在文本回复中 @ 用户 |
| `enabled` | *bool | ❌ | `true` | 是否启用规则 |

### 配置文件示例

```toml
# HTTP 请求超时时间（秒）
http_timeout_seconds = 15

[[rules]]
id = "random-image"
keywords = ["来张图", "随机图片"]
url = "https://api.example.com/random/image"
method = "GET"
send_type = "image"
result_path = "data.url"
at = false
enabled = true

[[rules]]
id = "cp-match"
keywords = ["cp配对", "cp"]
url = "https://api.example.com/cp?a={{arg_1}}&b={{arg_2}}"
method = "GET"
send_type = "text"
result_path = "result"
at = true
enabled = true
```

## 🔧 命令使用

### 查看规则

```bash
# 查看所有规则
/universal list

# 查看指定规则详情
/universal get <id>
```

### 新增规则

```bash
/universal add <id> -k <关键词列表> -u <URL模板> [选项]
```

**选项：**
- `-k, --keywords`：关键词列表，逗号分隔（必填）
- `-u, --url`：请求地址模板（必填）
- `-m, --method`：请求方法，默认 `GET`
- `-H, --headers`：请求头，格式：`A=B;C=D`
- `-b, --body`：请求体模板
- `-t, --send-type`：发送类型，默认 `text`
- `-p, --result-path`：gjson 结果路径
- `--at`：是否 @ 用户，默认 `false`

**示例：**

```bash
# 简单图片规则
/universal add pixiv -k 来张图 -u "https://api.example.com/image" -t image -p data.url

# 带参数的 POST 请求
/universal add search -k 搜索 -u "https://api.example.com/search" -m POST -b '{"query":"{{arg_1}}"}' -p results.0.title

# 带自定义请求头
/universal add api -k 查询 -u "https://api.example.com/data?id={{arg_1}}" -H "Authorization=Bearer TOKEN;Accept=application/json" -p data.content

# 启用 @ 功能
/universal add cp -k cp -u "https://api.example.com/cp?a={{arg_1}}&b={{arg_2}}" -t text --at true

# 响应体即为二进制视频（无需 result_path，直接发送）
/universal add haibian -k 看风景,海边 -u "http://api.yujn.cn/api/haibian.php" -t video
```

### 更新规则

```bash
/universal update <id> [选项]
```

**选项：**
- 所有 `add` 命令的选项
- `--clear-headers`：清空请求头
- `--clear-body`：清空请求体
- `--clear-result-path`：清空结果路径

**示例：**

```bash
# 更新 URL
/universal update pixiv -u "https://new-api.example.com/image"

# 更新发送类型和结果路径
/universal update search -t text -p results.0.description

# 清空请求头
/universal update api --clear-headers

# 启用 @ 功能
/universal update cp --at true
```

### 启用/禁用规则

```bash
# 启用规则
/universal enable <id>

# 禁用规则
/universal disable <id>
```

### 删除规则

```bash
/universal delete <id>
```

### 查看帮助

```bash
/universal help
```

## 💡 使用场景

### 1. 随机内容获取

**场景**：获取随机图片、名言、笑话等

```bash
# 随机图片
/universal add random-pic -k 来张图,随机图 -u "https://api.example.com/random/image" -t image -p url

# 随机笑话
/universal add joke -k 讲个笑话,来个笑话 -u "https://api.example.com/joke" -t text -p content
```

### 2. 查询服务

**场景**：天气查询、词典、快递查询等

```bash
# 天气查询
/universal add weather -k 天气 -u "https://api.example.com/weather?city={{arg_1}}" -t text -p data.description

# 快递查询
/universal add express -k 快递 -u "https://api.example.com/express?no={{arg_1}}" -t text -p data.status
```

### 3. CP 配对游戏

**场景**：群聊娱乐，随机配对

```bash
/universal add cp -k cp,配对 -u "https://api.example.com/cp?a={{arg_1}}&b={{arg_2}}" -t text -p result --at true
```

使用方式（支持引用消息）：
```
# 直接输入
cp 张三 李四

# 引用消息（被引用者自动成为第一个参数）
[引用张三的消息]
cp 李四
```

### 4. 图片处理

**场景**：图片美化、格式转换等

```bash
/universal add beautify -k 美化图片 -u "https://api.example.com/beautify?style={{style}}&intensity={{intensity}}" -m POST -b "{{quote}}" -t image -p result_url
```

使用方式（引用图片消息）：
```
[引用图片消息]
美化图片 style=anime intensity=high
```

### 5. 二次请求场景

**场景**：API 返回的是资源 URL，需要再次请求获取实际媒体内容

当 `send_type` 为 `image`/`video`/`emoji` 且 `result_path` 非空时，插件会先用 gjson 从首次响应提取 URL，再复用首次请求的头部对该 URL 进行第二次 GET，最终发送第二次响应体（即二进制媒体）。

```bash
/universal add douyin -k 抖音解析 -u "https://api.example.com/douyin/parse?url={{arg_1}}" -p video_url -t video
```

流程：
1. 第一次请求：解析抖音链接，响应 JSON 中含 `video_url`
2. 第二次请求：GET `video_url`（复用首次请求头），下载视频二进制
3. 发送视频消息

> 若 API 直接返回二进制媒体（响应体本身就是图片/视频），则将 `result_path` 留空即可，无需二次请求。

### 6. 自定义 API 调用

**场景**：调用内部 API、第三方服务等

```bash
# POST JSON 请求
/universal add create-task -k 创建任务 -u "https://internal-api.example.com/tasks" -m POST -H "Authorization=Bearer TOKEN;Content-Type=application/json" -b '{"title":"{{arg_1}}","assignee":"{{arg_2}}"}' -p task.id

# 带认证的 GET 请求
/universal add query-user -k 查询用户 -u "https://api.example.com/users/{{arg_1}}" -H "X-API-Key=YOUR_KEY" -p user.name
```

## 📌 高级技巧

### 1. gjson 路径语法

gjson 支持强大的 JSON 查询语法：

```bash
# 简单路径
data.url

# 数组索引
results.0.title

# 数组遍历
results.#.title

# 条件查询
results.#(status=="success").url

# 嵌套路径
data.images.0.urls.original
```

详见：[gjson 语法文档](https://github.com/tidwall/gjson/blob/master/SYNTAX.md)

### 2. 请求头模板

支持在请求头中使用模板变量：

```bash
/universal add auth-api -k 查询 -u "https://api.example.com/data" -H "Authorization=Bearer {{token}};X-User-ID={{arg_1}}"
```

使用时：
```
查询 token=abc123 user_id=12345
```

### 3. 动态请求体

POST/PUT 请求支持模板化的请求体：

```bash
/universal add webhook -k 通知 -u "https://webhook.example.com" -m POST -b '{"text":"{{arg_1}}","user":"{{arg_2}}","channel":"{{channel}}"}'
```

### 4. 组合使用引用和参数

```bash
/universal add comment -k 评论 -u "https://api.example.com/comment" -m POST -b '{"content":"{{arg_1}}","reply_to":"{{quoter}}","original":"{{quote}}"}' --at true
```

使用方式：
```
[引用某条消息]
评论 我也这么觉得
```

效果：
- `{{arg_1}}`：`我也这么觉得`
- `{{quoter}}`：被引用消息的发送者昵称
- `{{quote}}`：被引用的消息内容
- 回复时会 @ 被引用者

## ⚠️ 注意事项

1. **关键词唯一性**：同一关键词不能被多个规则使用
2. **关键词格式**：关键词不能包含空白字符（空格、Tab、换行等）
3. **规则 ID 唯一性**：每个规则的 ID 必须唯一
4. **HTTP 超时**：默认超时 15 秒，可通过 `http_timeout_seconds` 配置
5. **变量命名**：自定义变量名必须符合 `[A-Za-z_][A-Za-z0-9_]*` 格式
6. **请求头格式**：多个请求头用分号分隔，格式为 `Key=Value;Key2=Value2`
7. **媒体二次请求**：`send_type` 为媒体且 `result_path` 非空时，提取出的字段必须是合法 URL，否则报错；`result_path` 为空时响应体必须是非空二进制

## 🔍 故障排查

### 规则不触发

1. 检查关键词是否匹配（大小写敏感）
2. 确认规则已启用：`/universal get <id>`
3. 查看规则列表：`/universal list`

### HTTP 请求失败

1. 检查 URL 模板是否正确
2. 确认模板变量是否正确替换
3. 检查请求头和请求体格式
4. 增加超时时间：修改配置文件中的 `http_timeout_seconds`

### 结果提取失败

1. 确认 API 返回的是 JSON 格式
2. 检查 `result_path` 路径是否正确
3. 使用空路径获取完整响应：`--clear-result-path`
4. 参考 gjson 语法文档调整路径

### @ 功能不生效

1. 确认规则已启用 `at` 选项
2. 检查是否使用了引用消息或参数中包含 @ 用户
3. 确认消息类型为 `text`（其他类型不支持 @）

## 📄 许可证

遵循主项目许可证。

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

---

**作者**：ovo  
**版本**：1.0.0
