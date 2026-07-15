package main

import (
	"encoding/xml"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/sbgayhub/golem/sdk/contact"
	"github.com/sbgayhub/golem/sdk/message"
	"github.com/sbgayhub/golem/sdk/plugin"
)

func extractText(msg *message.Message) string {
	if msg.Type.Code == message.TypeText.Code {
		return msg.Content
	}
	if msg.Type.Code == message.TypeAppQuote.Code {
		val := extractQuoteContent(msg)
		if val != "" {
			return val
		}
		return msg.Content
	}
	return ""
}

func extractQuoteContent(msg *message.Message) string {
	if app := msg.GetApp(); app != nil {
		xmlStr := app.GetXml()
		// 提取 <title> 内容（引用消息的文本摘要）
		var data struct {
			Title string `xml:"title"`
		}
		if err := xml.Unmarshal([]byte(xmlStr), &data); err == nil && data.Title != "" {
			return data.Title
		}
	}
	return ""
}

func matchPrefix(text string) (string, string) {
	for _, p := range prefixes {
		if text == p {
			return p, ""
		}
		if strings.HasPrefix(text, p+" ") {
			return p, text[len(p)+1:]
		}
	}
	return "", ""
}

func (m *MemePlugin) sendText(event *plugin.Event, text string) {
	receiver := m.contact.Get(event.GetSender())
	if receiver == nil {
		slog.Warn("未找到接收者联系人")
		return
	}
	if _, err := m.message.Send(&message.Message{
		Content:  text,
		Receiver: receiver,
		Type:     message.TypeText,
		Data:     &message.Message_Text{Text: &message.TextData{Content: text}},
	}); err != nil {
		slog.Warn("发送文本消息失败", "err", err)
	}
}

func (m *MemePlugin) handleList(event *plugin.Event, prefix string) bool {
	infos := m.getInfoList()
	if len(infos) == 0 {
		m.sendText(event, "表情列表为空，请尝试 "+prefix+" reload")
		return true
	}

	receiver := m.contact.Get(event.GetSender())
	if receiver == nil {
		return true
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("可用表情列表 共%d个\n\n", len(infos)))

	for i, info := range infos {
		if len(info.Keywords) > 0 {
			sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, strings.Join(info.Keywords, "/")))
		}
		if i >= 50 {
			sb.WriteString("\n... 更多请使用 meme reload 刷新")
			break
		}
	}

	if _, err := m.message.Send(&message.Message{
		Content:  sb.String(),
		Receiver: receiver,
		Type:     message.TypeText,
		Data:     &message.Message_Text{Text: &message.TextData{Content: sb.String()}},
	}); err != nil {
		slog.Warn("发送列表失败", "err", err)
	}
	return true
}

func buildRecordXml(title, desc string, records []record) string {
	var sb strings.Builder
	sb.WriteString("<![CDATA[<recordinfo>\n")
	sb.WriteString(fmt.Sprintf("<title>%s</title>\n", title))
	sb.WriteString(fmt.Sprintf("<desc>%s</desc>\n", desc))
	sb.WriteString(fmt.Sprintf("<datalist count=\"%d\">\n", len(records)))
	for _, r := range records {
		t := time.Now()
		sb.WriteString(fmt.Sprintf(`<dataitem datatype="1">`+
			`<datadesc>%s</datadesc>`+
			`<sourcename>%s</sourcename>`+
			`<sourceheadurl>%s</sourceheadurl>`+
			`<sourcetime>%s</sourcetime>`+
			`<srcMsgCreateTime>%d</srcMsgCreateTime>`+
			`<fromnewmsgid>%d</fromnewmsgid>`+
			`</dataitem>`, r.content, r.name, r.avatar, r.time, t.Unix(), t.UnixNano()))
	}
	sb.WriteString("</datalist></recordinfo>]]>")
	return sb.String()
}

func (m *MemePlugin) sendRecord(receiver *contact.Contact, title, desc string, records []record) {
	_ = buildRecordXml(title, desc, records)
	slog.Warn("聊天记录卡片发送功能暂未实现")
}
