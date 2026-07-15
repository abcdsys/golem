package main

import (
	"encoding/xml"
	"strings"

	"github.com/sbgayhub/golem/sdk/chatroom"
	"github.com/sbgayhub/golem/sdk/contact"
	"github.com/sbgayhub/golem/sdk/message"
	"github.com/sbgayhub/golem/sdk/plugin"
)

func userInfoFromMember(m *chatroom.Member) *userInfo {
	if m == nil {
		return nil
	}
	name := m.GetNickname()
	if m.GetDisplayName() != "" {
		name = m.GetDisplayName()
	}
	return &userInfo{Avatar: m.GetAvatar(), Nickname: name}
}

func userInfoFromContact(c *contact.Contact) *userInfo {
	if c == nil {
		return nil
	}
	name := c.GetNickname()
	if c.GetRemark() != "" {
		name = c.GetRemark()
	}
	return &userInfo{Avatar: c.GetAvatar(), Nickname: name}
}

func (m *MemePlugin) collectUsers(event *plugin.Event, msg *message.Message) (target *userInfo, sender *userInfo) {
	isGroup := msg.GetSender() != nil && msg.GetSender().GetType() == contact.ContactType_CONTACT_TYPE_CHATROOM

	// 1. sender: prefer member (with DisplayName) in groups
	if isGroup && msg.GetMember() != nil {
		sender = userInfoFromMember(msg.GetMember())
	} else {
		sender = userInfoFromContact(msg.GetSender())
	}

	// 2. quote → parse XML for chatusr
	if msg.GetApp() != nil {
		if quote := parseQuoteXML(msg.GetApp().GetXml()); quote.chatUser != "" {
			target = m.resolveUser(quote.chatUser, isGroup, event.GetSender())
			if target != nil {
				return target, sender
			}
		}
	}

	// 3. @message → resolve first @-mentioned user from Reminds
	if isGroup {
		if text := msg.GetText(); text != nil {
			reminds := text.GetReminds()
			if len(reminds) > 0 {
				wxid := strings.TrimSpace(reminds[0])
				if wxid != "" {
					member := m.chatroom.GetMember(event.GetSender(), wxid)
					if member != nil {
						target = userInfoFromMember(member)
						return target, sender
					}
				}
			}
		}
	}

	return nil, sender
}

type quoteRefer struct {
	ChatUser    string `xml:"chatusr"`
	FromUser    string `xml:"fromusr"`
	DisplayName string `xml:"displayname"`
}

type quoteData struct {
	chatUser string
}

func parseQuoteXML(raw string) quoteData {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return quoteData{}
	}

	var parsed struct {
		AppMsg struct {
			Refer quoteRefer `xml:"refermsg"`
		} `xml:"appmsg"`
		Refer quoteRefer `xml:"refermsg"`
	}
	if err := xml.Unmarshal([]byte(raw), &parsed); err != nil {
		return quoteData{}
	}

	refer := parsed.AppMsg.Refer
	if refer.ChatUser == "" {
		refer = parsed.Refer
	}
	return quoteData{chatUser: strings.TrimSpace(refer.ChatUser)}
}

func (m *MemePlugin) resolveUser(wxid string, isGroup bool, chatroomWxid string) *userInfo {
	if wxid == "" {
		return nil
	}
	if isGroup {
		member := m.chatroom.GetMember(chatroomWxid, wxid)
		if member != nil {
			return userInfoFromMember(member)
		}
	}
	c := m.contact.Get(wxid)
	return userInfoFromContact(c)
}
