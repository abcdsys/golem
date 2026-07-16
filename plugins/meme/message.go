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

// handleList 以聊天记录卡片形式返回中文关键词列表
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

	const batchSize = 20
	var records []record
	for i := 0; i < len(infos); i += batchSize {
		end := i + batchSize
		if end > len(infos) {
			end = len(infos)
		}
		var entries []string
		for _, item := range infos[i:end] {
			if len(item.Keywords) > 0 {
				entries = append(entries, strings.Join(item.Keywords, "/"))
			}
		}
		records = append(records, record{
			name:    fmt.Sprintf("第 %d-%d 条", i+1, end),
			content: strings.Join(entries, "、"),
			time:    fmt.Sprintf("%d-%d", i+1, end),
		})
	}

	title := fmt.Sprintf("可用表情列表 共%d个", len(infos))
	desc := fmt.Sprintf("使用方法: %s 名称", prefix)
	m.sendRecord(receiver, title, desc, records)
	return true
}

// buildRecordXml 构建聊天记录卡片的 XML 内容
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

// sendRecord 发送聊天记录卡片消息
func (m *MemePlugin) sendRecord(receiver *contact.Contact, title, desc string, records []record) {
	recordXml := buildRecordXml(title, desc, records)
	appXml := fmt.Sprintf(`<appmsg>`+
		`<title>%s</title>`+
		`<des>%s</des>`+
		`<action>view</action>`+
		`<type>19</type>`+
		`<url>https://support.weixin.qq.com/cgi-bin/mmsupport-bin/readtemplate?t=page/favorite_record__w_unsupport&amp;from=singlemessage&amp;isappinstalled=0</url>`+
		`<recorditem>%s</recorditem>`+
		`</appmsg>`, title, desc, recordXml)
	if _, err := m.message.Send(&message.Message{
		Content:  title,
		Receiver: receiver,
		Type:     message.TypeApplication,
		Data: &message.Message_App{App: &message.AppData{
			SubType: 19,
			Title:   title,
			Desc:    desc,
			Xml:     appXml,
		}},
	}); err != nil {
		slog.Warn("发送聊天记录失败", "err", err)
	}
}
