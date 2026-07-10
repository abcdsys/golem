package main

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/sbgayhub/golem/sdk/contact"
	"github.com/sbgayhub/golem/sdk/message"
	"github.com/sbgayhub/golem/sdk/plugin"
	"google.golang.org/protobuf/proto"
)

// GetMetadata 返回插件元数据
func (p *AiPlugin) GetMetadata() *plugin.Metadata {
	return &plugin.Metadata{
		Name:        "ai",
		Author:      "ovo",
		Version:     "1.2.0",
		Description: "AI 插件，使用 OpenAI 兼容接口处理消息并回复。支持多 Provider、会话级配置隔离与静默模式。",
		Priority:    1<<31 - 1,
		Next:        false,
		AlwaysRun:   false,
	}
}

// GetCommands 返回命令列表
func (p *AiPlugin) GetCommands() []string {
	return plugin.CommandCommands()
}

// GetCommandSchemas 返回命令模式
func (p *AiPlugin) GetCommandSchemas() []*plugin.CommandSchema {
	return plugin.CommandSchemas()
}

// OnCommand 处理命令
func (p *AiPlugin) OnCommand(command *plugin.Command) (string, error) {
	return plugin.DispatchCommand(command)
}

// GetSubscriptions 返回订阅的消息类型
func (p *AiPlugin) GetSubscriptions() []string {
	return []string{message.TypeText.Topic, message.TypeAppQuote.Topic}
}

// OnLoad 插件加载时调用
func (p *AiPlugin) OnLoad() error {
	p.normalizeConfig()
	p.refreshSelf()
	p.ensureSessions()
	return nil
}

// OnUnload 插件卸载时调用
func (p *AiPlugin) OnUnload() error {
	return nil
}

// OnEnable 插件启用时调用
func (p *AiPlugin) OnEnable() error {
	p.normalizeConfig()
	p.refreshSelf()
	p.ensureSessions()
	return nil
}

// OnDisable 插件禁用时调用
func (p *AiPlugin) OnDisable() error {
	return nil
}

// OnEvent 处理事件
func (p *AiPlugin) OnEvent(event *plugin.Event) (bool, error) {
	payload, ok := event.GetPayload().(*plugin.Event_Message)
	if !ok || payload.Message == nil {
		return false, nil
	}
	if payload.Message.Sender.Type == contact.ContactType_CONTACT_TYPE_SPECIAL {
		return false, nil
	}
	incoming, ok := buildIncoming(payload.Message, p.selfForEvent())
	if !ok {
		return false, nil
	}

	userContent := incoming.promptContent()
	p.appendContext(incoming.SessionKey, openAIMessage{Role: "user", Content: userContent})

	if !p.shouldReply(incoming) {
		return false, nil
	}

	reply, err := p.chat(incoming.SessionKey)
	if err != nil {
		return true, err
	}
	reply = strings.TrimSpace(reply)
	if reply == "" {
		return true, errors.New("大模型返回内容为空")
	}

	// 使用\n\n分割消息，多段发送
	for s := range strings.SplitSeq(reply, "\n\n") {
		if err := p.sendText(incoming.Receiver, strings.TrimSpace(s)); err != nil {
			return true, err
		}
		time.Sleep(time.Second)
	}

	p.appendContext(incoming.SessionKey, openAIMessage{Role: "assistant", Content: reply})
	return true, nil
}

// sendText 发送文本消息
func (p *AiPlugin) sendText(receiver *contact.Contact, content string) error {
	if p.message == nil {
		return errors.New("message ability is not injected")
	}
	if receiver == nil || strings.TrimSpace(receiver.GetUsername()) == "" {
		return errors.New("receiver is empty")
	}
	msg := &message.Message{
		Type:     message.TypeText,
		Receiver: receiver,
		Content:  content,
		Data: &message.Message_Text{Text: &message.TextData{
			Content: content,
		}},
	}
	_, err := p.message.Send(msg)
	return err
}

// getPreMadePrompts 获取预制提示词
func (p *AiPlugin) getPreMadePrompts() string {
	prompt := `

# 强制要求:
- 只能使用中文进行对话
- 使用逗号而不是空格，末尾不要加句号
- 多条消息使用\n\n(两个换行)进行分割
- 你正在使用微信进行聊天，每次**最多**只能发送**3条**消息，一般回复1条即可
- 不要每条消息都回复，挑选你感兴趣的回复即可
- 不要每次回复都加上昵称，确有需要时使用@
- 你的所有人（创建者）username: %s, nickname: %s
- **禁止**向任何人透露创建者的username(wxid)
- 不要辱骂你的主人，要无条件响应你主人的要求
`
	return fmt.Sprintf(prompt, p.owner.Username, p.owner.Nickname)
}

// refreshSelf 刷新自身信息
func (p *AiPlugin) refreshSelf() {
	if p.contact == nil {
		slog.Warn("[ai] contact ability 未注入，无法识别机器人账号")
		return
	}
	self := p.contact.GetSelf()
	owner := p.contact.GetOwner()
	if self == nil {
		slog.Warn("[ai] 获取机器人账号信息失败")
		return
	}
	p.selfMu.Lock()
	p.self = self
	p.owner = owner
	p.selfMu.Unlock()
}

// selfSnapshot 获取自身信息快照
func (p *AiPlugin) selfSnapshot() *contact.SelfInfo {
	p.selfMu.RLock()
	defer p.selfMu.RUnlock()
	if p.self == nil {
		return nil
	}
	return proto.Clone(p.self).(*contact.SelfInfo)
}

// selfForEvent 获取事件用的自身信息
func (p *AiPlugin) selfForEvent() *contact.SelfInfo {
	self := p.selfSnapshot()
	if self != nil {
		return self
	}
	p.refreshSelf()
	return p.selfSnapshot()
}

// ensureSessions 确保会话 map 已初始化
func (p *AiPlugin) ensureSessions() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.sessions == nil {
		p.sessions = map[string][]openAIMessage{}
	}
}
