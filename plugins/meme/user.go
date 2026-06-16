package main

import (
	"log/slog"

	"github.com/sbgayhub/golem/sdk/contact"
	"github.com/sbgayhub/golem/sdk/message"
	"github.com/sbgayhub/golem/sdk/plugin"
)

func userInfoFromContact(c *contact.Contact) *userInfo {
	return &userInfo{Avatar: c.Avatar, Nickname: c.Nickname}
}

func (m *MemePlugin) collectUsers(event *plugin.Event, msg *message.Message) (target *userInfo, sender *userInfo) {
	senderContact := m.contact.Get(event.GetSender())
	if senderContact == nil {
		slog.Warn("未找到发送者联系人", "sender", event.GetSender())
		return nil, &userInfo{Avatar: "", Nickname: "未知"}
	}

	if msg.Member != nil {
		sender = &userInfo{Avatar: msg.Member.Avatar, Nickname: msg.Member.Nickname}
	} else {
		sender = &userInfo{Avatar: msg.Sender.Avatar, Nickname: msg.Sender.Nickname}
	}

	return nil, sender
}