# GG 插件

基于 gogpu/gg 库的文字转图片插件，提供两个能力：

- **text.to.image**：纯文本转 PNG（不解析 markdown，原样绘制），支持 Emoji
- **markdown.to.image**：markdown 转 PNG（goldmark 解析，支持标题 / 加粗 / 斜体 / 列表 / 引用 / 代码块 / 分隔线 / 链接）

完美支持彩色 Emoji 渲染，使用 MapleMono Nerd Font（Regular / Bold / Italic 三种字重）。

## 功能特性

- **双能力**：纯文本（text.to.image）与 markdown（markdown.to.image）两种渲染模式
- **Emoji 支持**：完美渲染彩色 Emoji 表情符号
- **中文字体**：MapleMono Nerd Font，Regular / Bold / Italic 三种字重，支持中文和特殊符号
- **自定义颜色**：支持自定义字体颜色和背景颜色
- **自动换行**：按字体宽度智能换行
- **不透明输出**：默认浅灰背景 `#F7F7F7`，输出无 alpha 的 RGB PNG（color type 2），兼容微信等客户端缩略图

---

## 能力一：text.to.image

将**纯文本原样**渲染成 PNG。不解析任何 markdown 符号，`#`、`**`、`-` 等按字面字符输出。

### 参数

| 参数 | 类型 | 必需 | 默认值 | 说明 |
|------|------|------|--------|------|
| `context` | string | 是 | - | 要转换的纯文本内容 |
| `font_color` | string | 否 | black | 字体颜色 |
| `bg_color` | string | 否 | #F7F7F7 | 背景颜色 |

### 调用示例

```json
{
  "capability": "text.to.image",
  "args": {
    "context": "你好，世界！👋",
    "font_color": "#FF0000",
    "bg_color": "#FFFFFF"
  }
}
```

---

## 能力二：markdown.to.image

将 markdown 文本**解析后按结构**渲染成 PNG（goldmark 引擎，CommonMark 子集）。

### 支持的 markdown 元素

| 元素 | markdown 语法 | 渲染效果 |
|------|--------------|----------|
| 标题 | `# H1` ~ `###### H6` | 分级字号 + 加粗（H1=38px ~ H6=24px） |
| 加粗 | `**文字**` | Bold 字重 |
| 斜体 | `*文字*` 或 `_文字_` | Italic 字重 |
| 行内代码 | `` `代码` `` | 等宽字体 + 红字 + 浅灰背景 |
| 无序列表 | `* ` / `- ` / `+ ` | `•` 项目符号，支持嵌套 |
| 有序列表 | `1. ` | 数字序号，支持嵌套 |
| 引用 | `> 文字` | 左侧灰色竖线 + 缩进 |
| 代码块 | ` ```lang ` 围栏或 4 空格缩进 | 等宽字体 + 浅灰背景 |
| 分隔线 | `---` / `***` | 水平横线 |
| 链接 | `[文字](url)` | 仅渲染链接文字 |

> **列表说明**：tight list（项间无空行）与 loose list（项间空行）均支持；嵌套列表按缩进递增。

### 参数

| 参数 | 类型 | 必需 | 默认值 | 说明 |
|------|------|------|--------|------|
| `context` | string | 是 | - | markdown 文本 |
| `font_color` | string | 否 | black | 字体颜色（标题 / 正文 / 列表统一） |
| `bg_color` | string | 否 | #F7F7F7 | 背景颜色 |

### 调用示例

```json
{
  "capability": "markdown.to.image",
  "args": {
    "context": "## 人物画像\n\n### 性格\n* **加粗项**：内容\n* 列表项\n  * 嵌套项\n\n> 引用文字\n\n行内 `代码` 演示"
  }
}
```

---

## 颜色格式

两个能力共用，支持：

**预定义名**：`black` / `white` / `red` / `green` / `blue`

**十六进制**：

- `#RGB` - 3 位，如 `#F00`
- `#RGBA` - 4 位，如 `#F00F`
- `#RRGGBB` - 6 位，如 `#FF0000`
- `#RRGGBBAA` - 8 位，如 `#FF0000FF`
- `0xRRGGBBAA` - 前缀格式

`font_color` 留空默认黑色（alpha=FF）；`bg_color` 留空默认 `#F7F7F7`（不透明浅灰）。
显式传 `#RRGGBB00` 可得透明背景，但**微信等客户端缩略图会把透明当黑底，不建议**。

## 样式配置

内置样式参数（不可经调用参数修改）：

- **正文字号**：24px
- **标题字号**：H1=38 / H2=32 / H3=28 / H4=26 / H5 / H6=24 px
- **最大宽度**：1600px
- **行间距**：0.8
- **内边距**：25px
- **字体**：MapleMono-NF-CN（Regular / Bold / Italic）+ NotoColorEmoji

## 输出规格

- **格式**：PNG
- **颜色模式**：RGB（color type 2，背景不透明时）。Go png encoder 在所有像素不透明时自动输出无 alpha 通道的 RGB，微信缩略图正常
- **尺寸**：按内容自动计算
  - text.to.image：宽 = 内边距×2 + 最长行宽；高 = 内边距×2 + 行数×行高
  - markdown.to.image：两 pass 布局（先测各块高度再绘制）

## 使用场景

- **代码片段分享**：`text.to.image` 原样保留缩进与符号
- **人物画像 / 结构化报告**：`markdown.to.image` 把 AI 输出的 markdown 渲染成带层级排版的图片（profile 插件即用此能力）
- **公告通知**：自定义前景 / 背景色制作醒目图片
- **名言海报**：纯文本居中排版

## 限制说明

1. **字体固定**：仅 MapleMono + NotoEmoji，不支持自定义字体
2. **text.to.image 不解析 markdown**：`#`、`**` 等按字面字符输出；需结构化渲染请用 markdown.to.image
3. **markdown.to.image 不支持表格 / 图片 / 原生 HTML**：基于 CommonMark 子集，暂未实现 table / img / html
4. **宽度限制**：最大 1600px，超出自动换行
5. **内嵌字体**：Regular / Bold / Italic + Emoji 合计较大，首次加载稍慢

## 常见问题

### 1. 微信缩略图黑底、点开大图正常

**原因**：背景透明导致 PNG 带 alpha 通道（color type 6），微信缩略图生成器把透明当黑底预乘；大图解码器正常故显示正常。

**解决**：使用不透明背景。当前版本 `bg_color` 默认已改为 `#F7F7F7`（不透明），无需额外配置；若自定义请避免 `#RRGGBB00`。

### 2. markdown 列表项内容缺失（旧版本）

**原因**：goldmark 对 tight list（项间无空行）的内容节点用 `TextBlock`，早期版本仅处理 `Paragraph` 导致跳过。

**解决**：已修复，tight / loose 列表均正常渲染。

### 3. Emoji 显示异常

确保使用标准 Emoji 字符；过于新的 Emoji 可能 NotoEmoji 未收录。

### 4. 中文乱码

确认 MapleMono 字体完整；输入使用 UTF-8。

## 开发信息

- **插件名称**：gg
- **版本**：0.1.0
- **作者**：ovo
- **入口结构**：`main.go`（能力分发）/ `text.go`（text.to.image）/ `markdown.go`（markdown.to.image）/ `font.go`（字体与 Emoji）/ `color.go`（颜色解析）
- **依赖库**：
  - `github.com/gogpu/gg/text` - 文本渲染
  - `github.com/gogpu/gg/text/emoji` - Emoji 分段与渲染
  - `github.com/yuin/goldmark` - markdown 解析
  - `golang.org/x/image/draw` - 图像操作
  - `github.com/sbgayhub/golem/sdk/plugin` - Golem SDK
- **字体文件**：
  - MapleMono-NF-CN-Regular.ttf (~20 MB)
  - MapleMono-NF-CN-Bold.ttf
  - MapleMono-NF-CN-Italic.ttf
  - NotoColorEmoji.ttf (~10 MB)

## 更新日志

### 0.1.0

- 新增 `markdown.to.image` 能力（goldmark 解析，支持标题 / 加粗 / 斜体 / 列表 / 引用 / 代码块 / 分隔线 / 链接）
- 新增 Bold / Italic 字重
- `bg_color` 默认值由透明改为 `#F7F7F7`（不透明，修复微信缩略图黑底）
- 修复 tight list（TextBlock 节点）内容缺失
- 输出 PNG 在不透明时为 RGB（color type 2），客户端兼容性更好

### 0.0.0

- 初始版本：`text.to.image` 纯文本转图片 + Emoji 渲染 + 自定义颜色 + 自动换行

## 参考资料

- [gogpu/gg](https://github.com/gogpu/gg) - 图形库
- [goldmark](https://github.com/yuin/goldmark) - CommonMark markdown 解析器
- [MapleMono Font](https://github.com/subframe7536/maple-font) - 字体项目
- [Noto Color Emoji](https://github.com/googlefonts/noto-emoji) - Emoji 字体
