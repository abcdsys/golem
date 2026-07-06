package main

import (
	"log/slog"
	"strings"

	"github.com/sbgayhub/golem/sdk/contact"
	"github.com/sbgayhub/golem/sdk/message"
	"github.com/sbgayhub/golem/sdk/plugin"
)

// OnEvent 处理事件
func (p *FarmPlugin) OnEvent(e *plugin.Event) (bool, error) {
	msg := e.Payload.(*plugin.Event_Message).Message
	if msg == nil {
		return false, nil
	}

	text := strings.TrimSpace(msg.GetContent())
	if text == "" {
		if td := msg.GetText(); td != nil {
			text = strings.TrimSpace(td.Content)
		}
	}
	if text == "" {
		return false, nil
	}

	receiverType := contact.ContactType(0)
	if msg.Receiver != nil {
		receiverType = msg.Receiver.GetType()
	}
	senderID := ""
	if msg.Sender != nil {
		senderID = msg.Sender.GetUsername()
	}
	slog.Debug("[farm] 收到消息", "text", text, "receiver_type", receiverType, "sender", senderID)

	// 非群聊只允许 "农场" 提示
	if !p.isGroupChat(msg) {
		if text == "农场" {
			p.sendText(msg.Sender, "农场功能只能在群中使用")
		}
		return false, nil
	}

	// 只处理农场相关命令
	if !p.isFarmCommand(text) {
		return false, nil
	}

	chatroomID := p.getChatroomID(msg)
	userID := p.getUserID(e, msg)
	replyTo := p.getReplyTo(msg)
	if chatroomID == "" || userID == "" || replyTo == nil {
		slog.Warn("[farm] 无法确定群聊或用户", "chatroom_id", chatroomID, "user_id", userID)
		return false, nil
	}
	slog.Debug("[farm] 处理群聊命令", "chatroom_id", chatroomID, "user_id", userID, "text", text)

	defer func() {
		if r := recover(); r != nil {
			slog.Error("[farm] 处理命令时 panic", "err", r)
			p.sendText(replyTo, "农场出错了，请稍后再试")
		}
	}()

	switch text {
	case "农场":
		p.printMenu(replyTo)
		p.sendImageIfAvailable(replyTo, menuImagePath)
	case "农场帮助":
		p.printHelp(replyTo)
	case "农场商店":
		p.printCrops(chatroomID, userID, replyTo)
	case "守卫商店":
		p.printPets(chatroomID, userID, replyTo)
	case "农场购买种子", "农场购买守卫":
		p.printHelpBuy(replyTo)
	case "查询种子", "查询守卫":
		p.printHelpSearch(replyTo)
	case "种植":
		p.printHelpPlant(replyTo)
	case "偷菜":
		p.printHelpSteal(replyTo)
	case "收菜":
		p.collect(chatroomID, userID, replyTo)
	case "浇水":
		p.water(chatroomID, userID, "", msg, replyTo)
	case "我的农场":
		p.printSelf(chatroomID, userID, replyTo)
	case "农场等级":
		p.printLevels(chatroomID, userID, replyTo)
	case "购买土地":
		p.buyField(chatroomID, userID, replyTo)
	default:
		if strings.HasPrefix(text, "查询") {
			name := strings.TrimSpace(text[6:])
			p.search(chatroomID, userID, name, replyTo)
		} else if strings.HasPrefix(text, "农场购买") || strings.HasPrefix(text, "农场买") {
			normalized := p.normalizeFarmBuyContent(text)
			req := p.parseBuyRequest(normalized)
			if req == nil {
				p.sendText(replyTo, "格式错误！请使用：农场购买+名称(+数量)")
				return true, nil
			}
			p.buy(chatroomID, userID, req, replyTo)
		} else if strings.HasPrefix(text, "种植") || strings.HasPrefix(text, "播种") || strings.HasPrefix(text, "种") {
			name := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(strings.TrimPrefix(text, "种植"), "播种"), "种"))
			if name == "" {
				p.printHelpPlant(replyTo)
			} else {
				p.plant(chatroomID, userID, name, replyTo)
			}
		} else if strings.HasPrefix(text, "偷菜") {
			name := strings.TrimSpace(text[6:])
			p.steal(chatroomID, userID, name, msg, replyTo)
		} else if strings.HasPrefix(text, "浇水") {
			name := strings.TrimSpace(text[6:])
			p.water(chatroomID, userID, name, msg, replyTo)
		}
	}

	return true, nil
}

func (p *FarmPlugin) isGroupChat(msg *message.Message) bool {
	return p.getChatroomID(msg) != ""
}

// getChatroomID 从消息中解析群聊 ID。
// SDK 中 msg.Member 仅群消息有效；同时兼容 Receiver.Type == CHATROOM
// 以及 Sender 用户名以 @chatroom 结尾的两种消息模型。
func (p *FarmPlugin) getChatroomID(msg *message.Message) string {
	if msg.Receiver != nil && msg.Receiver.Type == contact.ContactType_CONTACT_TYPE_CHATROOM {
		return msg.Receiver.GetUsername()
	}
	if msg.Sender != nil && strings.HasSuffix(msg.Sender.GetUsername(), "@chatroom") {
		return msg.Sender.GetUsername()
	}
	return ""
}

// getUserID 获取真实发送者 ID。
// 群聊场景下优先使用 msg.Member（仅群消息有效，表示群内发言成员），
// 其次使用 plugin.Event.Sender，最后回退到 msg.Sender。
func (p *FarmPlugin) getUserID(e *plugin.Event, msg *message.Message) string {
	if msg.Member != nil && msg.Member.GetUsername() != "" {
		return msg.Member.GetUsername()
	}
	if e.GetSender() != "" && !strings.HasSuffix(e.GetSender(), "@chatroom") {
		return e.GetSender()
	}
	if msg.Sender != nil && !strings.HasSuffix(msg.Sender.GetUsername(), "@chatroom") {
		return msg.Sender.GetUsername()
	}
	return ""
}

// getReplyTo 返回用于回复群消息的 contact。
func (p *FarmPlugin) getReplyTo(msg *message.Message) *contact.Contact {
	if msg.Receiver != nil && msg.Receiver.Type == contact.ContactType_CONTACT_TYPE_CHATROOM {
		return msg.Receiver
	}
	if msg.Sender != nil && strings.HasSuffix(msg.Sender.GetUsername(), "@chatroom") {
		return msg.Sender
	}
	return nil
}

func (p *FarmPlugin) isFarmCommand(content string) bool {
	commands := []string{
		"农场帮助", "农场商店", "守卫商店", "农场购买种子", "农场购买守卫",
		"查询种子", "查询守卫", "种植", "偷菜", "收菜", "浇水", "我的农场",
		"农场等级", "购买土地",
	}
	for _, cmd := range commands {
		if content == cmd {
			return true
		}
	}
	prefixes := []string{"查询", "农场购买", "农场买", "种植", "播种", "种", "偷菜", "浇水"}
	for _, prefix := range prefixes {
		if strings.HasPrefix(content, prefix) {
			return true
		}
	}
	return content == "农场"
}

func (p *FarmPlugin) normalizeFarmBuyContent(content string) string {
	content = strings.TrimPrefix(content, "农场购买")
	content = strings.TrimPrefix(content, "农场买")
	return strings.TrimSpace(content)
}
