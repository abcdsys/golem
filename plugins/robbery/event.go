package main

import (
	"log/slog"
	"strings"

	"github.com/sbgayhub/golem/sdk/contact"
	"github.com/sbgayhub/golem/sdk/message"
	"github.com/sbgayhub/golem/sdk/plugin"
)

// GetSubscriptions 返回订阅的事件列表
func (p *RobberyPlugin) GetSubscriptions() []string {
	return []string{
		message.TypeText.Topic, // 订阅文本消息
	}
}

// OnEvent 事件处理入口
func (p *RobberyPlugin) OnEvent(e *plugin.Event) (bool, error) {
	em, ok := e.Payload.(*plugin.Event_Message)
	if !ok || em.Message == nil {
		return false, nil
	}

	msg := em.Message
	content := strings.TrimSpace(msg.GetContent())
	if content == "" {
		return false, nil
	}

	// 只处理群聊消息
	if !p.isGroupChat(msg) {
		return false, nil
	}

	// 命令匹配
	if !p.isGameCommand(content) {
		return false, nil
	}

	chatroomID := p.getChatroomID(msg)
	userID := p.getUserID(e, msg)
	if chatroomID == "" || userID == "" {
		slog.Warn("[robbery] 无法获取群ID或用户ID",
			"chatroom_id", chatroomID, "user_id", userID)
		return false, nil
	}

	// 回复目标（群聊 Contact）
	replyTo := p.getReplyTo(msg)
	if replyTo == nil {
		slog.Warn("[robbery] 无法确定回复目标", "chatroom_id", chatroomID)
		return false, nil
	}

	// 异常兜底
	defer func() {
		if r := recover(); r != nil {
			slog.Error("[robbery] 游戏处理 panic",
				"user_id", userID, "command", content, "err", r)
			p.sendText(replyTo, "游戏出错了，请稍后再试~ 😅")
		}
	}()

	// 这些命令不需要初始化群成员，直接处理
	switch content {
	case "我的资产":
		player := p.getOrCreatePlayer(chatroomID, userID, msg)
		p.showAssets(player, replyTo)
		return true, nil
	case "排行榜":
		p.showLeaderboard(chatroomID, replyTo)
		return true, nil
	case "职业列表":
		p.showProfessions(replyTo)
		return true, nil
	case "商店":
		p.showShop(replyTo)
		return true, nil
	case "任务":
		player := p.getOrCreatePlayer(chatroomID, userID, msg)
		p.showJobs(player, replyTo)
		return true, nil
	case "救济":
		player := p.getOrCreatePlayer(chatroomID, userID, msg)
		p.claimWelfare(player, replyTo)
		return true, nil
	case "打劫帮助":
		p.showHelp(replyTo)
		return true, nil
	}

	// 其他命令需要先初始化群聊成员
	p.initializeGroupMembers(chatroomID)
	player := p.getOrCreatePlayer(chatroomID, userID, msg)

	// 检查是否被监禁
	if player.InJail {
		p.handleJailStatus(player, replyTo)
		return true, nil
	}

	switch {
	case content == "打劫":
		p.performRobbery(player, chatroomID, replyTo)
	case content == "技能":
		p.useSkill(player, replyTo)
	case strings.HasPrefix(content, "转职"):
		p.changeProfession(player, content, replyTo)
	case strings.HasPrefix(content, "购买"):
		p.buyEquipment(player, content, replyTo)
	case strings.HasPrefix(content, "赌博"):
		p.gamble(player, content, replyTo)
	default:
		return false, nil
	}

	return true, nil
}

// isGameCommand 判断是否为游戏命令
func (p *RobberyPlugin) isGameCommand(content string) bool {
	switch content {
	case "打劫", "我的资产", "排行榜", "职业列表", "技能", "商店", "任务", "救济", "打劫帮助":
		return true
	}
	return strings.HasPrefix(content, "转职") ||
		strings.HasPrefix(content, "购买") ||
		strings.HasPrefix(content, "赌博")
}

// isGroupChat 判断是否为群聊消息
func (p *RobberyPlugin) isGroupChat(msg *message.Message) bool {
	if msg.Receiver != nil && msg.Receiver.Type == contact.ContactType_CONTACT_TYPE_CHATROOM {
		return true
	}
	if msg.Sender != nil && strings.HasSuffix(msg.Sender.GetUsername(), "@chatroom") {
		return true
	}
	if msg.Member != nil {
		return true
	}
	return false
}

// getChatroomID 获取群聊 ID
func (p *RobberyPlugin) getChatroomID(msg *message.Message) string {
	if msg.Receiver != nil && msg.Receiver.Type == contact.ContactType_CONTACT_TYPE_CHATROOM {
		return msg.Receiver.GetUsername()
	}
	if msg.Sender != nil && strings.HasSuffix(msg.Sender.GetUsername(), "@chatroom") {
		return msg.Sender.GetUsername()
	}
	if msg.Member != nil && msg.Sender != nil {
		return msg.Sender.GetUsername()
	}
	return ""
}

// getUserID 获取真实用户 ID（群消息中 Sender 可能是群账号）
func (p *RobberyPlugin) getUserID(e *plugin.Event, msg *message.Message) string {
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

// getReplyTo 获取回复目标（群聊 Contact）
func (p *RobberyPlugin) getReplyTo(msg *message.Message) *contact.Contact {
	if msg.Receiver != nil && msg.Receiver.Type == contact.ContactType_CONTACT_TYPE_CHATROOM {
		return msg.Receiver
	}
	if msg.Sender != nil && strings.HasSuffix(msg.Sender.GetUsername(), "@chatroom") {
		return msg.Sender
	}
	return nil
}

// sendText 发送文本消息到指定接收者
func (p *RobberyPlugin) sendText(receiver *contact.Contact, text string) {
	msg := &message.Message{
		Type:     message.TypeText,
		Receiver: receiver,
		Content:  text,
		Data:     &message.Message_Text{Text: &message.TextData{Content: text}},
	}
	if _, err := p.message.Send(msg); err != nil {
		slog.Error("[robbery] 发送文本失败", "err", err)
	}
}
