package main

// historyMsg 从 statistics.query_messages 能力返回的 JSON 反序列化得到的历史发言。
// 字段须与 statistics 插件 historyMsg 的 json tag 一致。
type historyMsg struct {
	ID        int64  `json:"id"`
	Content   string `json:"content"`
	Timestamp string `json:"timestamp"`
}

// profileRecord 画像记录（持久化于 profile.db 的 profiles 表）
type profileRecord struct {
	Chatroom  string // 群 wxid；全局画像为空字符串
	Member    string // 成员 wxid
	Profile   string // 画像文本
	LastMsgID int64  // 已处理到的 statistics.id 水位线
	UpdatedAt string
}

// chatMessage 发给 ai.chat 能力的消息结构（与 ai 插件的 openAIMessage 对应）
type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// aiChatPayload 与 ai 插件 ai.chat 能力约定的请求结构
type aiChatPayload struct {
	System   string        `json:"system"`
	Messages []chatMessage `json:"messages"`
}

// named 同时适配 *contact.Contact 与 *chatroom.Member 的展示名读取
type named interface {
	GetRemark() string
	GetNickname() string
	GetAlias() string
	GetUsername() string
}
