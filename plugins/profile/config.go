package main

// 人物画像相关参数的默认值（也可经 plugins/config.toml 的 [profile.config] 覆盖）
const (
	defaultChunkTokenBudget     = 4000 // 每块消息 token 估算上限；中文模型可上调到 6000
	defaultMaxSingleMsgChars    = 2000 // 单条消息硬截断长度
	defaultColdStartMaxMessages = 2000 // 冷启动最多处理的消息条数（安全天花板）
	defaultColdStartMaxChunks   = 30   // 冷启动最多分块数（安全天花板）
)

// Config 插件配置（经 plugins/config.toml 的 [profile.config] 注入）
type Config struct {
	// ChunkTokenBudget 每块消息的 token 估算上限，按内容量切块而非条数。默认 4000；中文模型可调大到 6000
	ChunkTokenBudget int `toml:"chunk_token_budget" comment:"每块消息的 token 估算上限，按内容量切块而非条数。默认 4000；中文模型可调大到 6000"`
	// MaxSingleMsgChars 单条消息硬截断长度，防止超长消息撑爆请求。默认 2000
	MaxSingleMsgChars int `toml:"max_single_msg_chars" comment:"单条消息硬截断长度，防止超长消息撑爆请求。默认 2000"`
	// ColdStartMaxMessages 冷启动最多处理的消息条数（安全天花板），超出只取最近的部分。默认 2000
	ColdStartMaxMessages int `toml:"cold_start_max_messages" comment:"冷启动最多处理的消息条数（安全天花板），超出只取最近的部分。默认 2000"`
	// ColdStartMaxChunks 冷启动最多分块数（安全天花板），超出丢弃最早的。默认 30
	ColdStartMaxChunks int `toml:"cold_start_max_chunks" comment:"冷启动最多分块数（安全天花板），超出丢弃最早的。默认 30"`
	// RenderImage 人物画像是否渲染成图片发送（经 gg 插件 text.to.image 能力）。默认 true；
	// 设为 false 则发纯文本。gg 未启用或渲染失败时仍会自动回退为文本。
	RenderImage bool `toml:"render_image" comment:"人物画像是否渲染成图片发送（经 gg 插件 text.to.image）。默认 true；设为 false 发纯文本。gg 未启用或渲染失败时自动回退文本"`
}

func defaultConfig() Config {
	return Config{
		ChunkTokenBudget:     defaultChunkTokenBudget,
		MaxSingleMsgChars:    defaultMaxSingleMsgChars,
		ColdStartMaxMessages: defaultColdStartMaxMessages,
		ColdStartMaxChunks:   defaultColdStartMaxChunks,
		RenderImage:          true,
	}
}

// normalizeConfig 补默认值（用户在 config.toml 里留空时兜底）
func normalizeConfig(c Config) Config {
	if c.ChunkTokenBudget <= 0 {
		c.ChunkTokenBudget = defaultChunkTokenBudget
	}
	if c.MaxSingleMsgChars <= 0 {
		c.MaxSingleMsgChars = defaultMaxSingleMsgChars
	}
	if c.ColdStartMaxMessages <= 0 {
		c.ColdStartMaxMessages = defaultColdStartMaxMessages
	}
	if c.ColdStartMaxChunks <= 0 {
		c.ColdStartMaxChunks = defaultColdStartMaxChunks
	}
	// RenderImage 为 bool，留空时由 toml 合并机制保留 defaultConfig 的 true，这里不动
	return c
}
