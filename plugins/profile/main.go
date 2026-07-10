package main

import (
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/sbgayhub/golem/sdk/chatroom"
	"github.com/sbgayhub/golem/sdk/contact"
	"github.com/sbgayhub/golem/sdk/message"
	"github.com/sbgayhub/golem/sdk/plugin"
)

// ProfilePlugin 人物画像插件（独立）
type ProfilePlugin struct {
	plugin.ConfigAbility[Config]

	contact  contact.Ability
	chatroom chatroom.Ability
	caller   plugin.CallerAbility
	message  message.Ability

	mu    sync.RWMutex
	store *store
}

func newProfilePlugin() *ProfilePlugin {
	return &ProfilePlugin{
		ConfigAbility: plugin.ConfigAbility[Config]{
			Config: defaultConfig(),
		},
	}
}

// GetMetadata 返回插件元数据
func (p *ProfilePlugin) GetMetadata() *plugin.Metadata {
	return &plugin.Metadata{
		Name:        "profile",
		Author:      "ovo",
		Version:     "1.1.0",
		Description: "群成员人物画像插件：基于历史发言经 AI 生成/增量更新画像并渲染图片发送。历史发言经 statistics.query_messages 能力获取，LLM 经 ai.chat；按配置 RenderImage 决定渲染图片（经 markdown.to.image，要求 LLM 输出 markdown）或发送纯文本（要求 LLM 输出无 markdown 符号的纯文本）。群聊任意成员可查本群成员；私聊人人可查自己的全局画像或自己在 #指定群 的画像，主人（Owner）还可查指定成员的全局 / #指定群画像。",
		Priority:    0,
		Next:        false,
		AlwaysRun:   false,
	}
}

// GetSubscriptions 订阅文本消息，用于「人物画像」触发
func (p *ProfilePlugin) GetSubscriptions() []string {
	return []string{message.TypeText.Topic}
}

// OnEvent 消息事件：检测「人物画像」触发语，异步生成并回复
func (p *ProfilePlugin) OnEvent(event *plugin.Event) (bool, error) {
	payload, ok := event.GetPayload().(*plugin.Event_Message)
	if !ok || payload.Message == nil {
		return false, nil
	}
	msg := payload.Message
	if msg.GetSender() == nil {
		return false, nil
	}
	if msg.Sender.GetType() == contact.ContactType_CONTACT_TYPE_SPECIAL {
		return false, nil
	}

	opts, triggered := parseTrigger(msg)
	if !triggered {
		return false, nil
	}

	return p.handleProfile(msg, opts)
}

// OnLoad 插件加载时调用
func (p *ProfilePlugin) OnLoad() error {
	return p.ensureStore()
}

// OnUnload 插件卸载时调用
func (p *ProfilePlugin) OnUnload() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.store != nil {
		err := p.store.Close()
		p.store = nil
		return err
	}
	return nil
}

// OnEnable 插件启用时调用
func (p *ProfilePlugin) OnEnable() error {
	return p.ensureStore()
}

// OnDisable 插件禁用时调用
func (p *ProfilePlugin) OnDisable() error {
	return nil
}

func (p *ProfilePlugin) ensureStore() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.store != nil {
		return nil
	}
	st, err := openStore()
	if err != nil {
		return err
	}
	p.store = st
	return nil
}

// sendText 发送文本消息
func (p *ProfilePlugin) sendText(receiver *contact.Contact, content string) error {
	if p.message == nil {
		return errors.New("message ability 未注入")
	}
	if receiver == nil || receiver.GetUsername() == "" {
		return errors.New("receiver 为空")
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

// sendProfileResult 按配置决定把画像渲染成图片（经 gg 插件 markdown.to.image）还是发纯文本。
// render_image=true 时尝试渲染图片；gg 未启用/渲染失败自动回退文本。render_image=false 直接发文本。
func (p *ProfilePlugin) sendProfileResult(receiver *contact.Contact, text string) error {
	if normalizeConfig(p.Config).RenderImage {
		if data, err := p.renderMarkdownToImage(text); err == nil && len(data) > 0 {
			if imgErr := p.sendImage(receiver, data); imgErr == nil {
				return nil
			}
		} else {
			slog.Warn("[profile] 画像渲染图片失败，回退为文本", "err", err)
		}
	}
	return p.sendText(receiver, text)
}

// renderMarkdownToImage 调用 markdown.to.image 能力把 markdown 画像渲染成 PNG
func (p *ProfilePlugin) renderMarkdownToImage(text string) ([]byte, error) {
	if p.caller == nil {
		return nil, fmt.Errorf("caller 未注入（需要 gg 插件提供 markdown.to.image）")
	}
	mime, data, err := p.caller.CallPlugin("markdown.to.image", map[string]string{"context": text})
	if err != nil {
		return nil, err
	}
	_ = mime
	if len(data) == 0 {
		return nil, fmt.Errorf("markdown.to.image 返回空数据")
	}
	return data, nil
}

// sendImage 发送图片消息（PNG 字节）
func (p *ProfilePlugin) sendImage(receiver *contact.Contact, pngData []byte) error {
	if p.message == nil {
		return errors.New("message ability 未注入")
	}
	if receiver == nil || receiver.GetUsername() == "" {
		return errors.New("receiver 为空")
	}
	_, err := p.message.Send(&message.Message{
		Type:     message.TypeImage,
		Receiver: receiver,
		Content:  "人物画像",
		Data: &message.Message_Image{Image: &message.ImageData{
			Media: &message.Media{Data: pngData},
		}},
	})
	return err
}

func main() {
	plugin.Start(newProfilePlugin())
}
