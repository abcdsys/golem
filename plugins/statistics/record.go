package main

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/sbgayhub/golem/sdk/contact"
	"github.com/sbgayhub/golem/sdk/message"
)

type recordItem struct {
	Name    string
	Avatar  string
	Content string
	Time    string
}

func (p *StatisticsPlugin) sendRecord(receiver *contact.Contact, title, desc string, records []recordItem) error {
	if p.message == nil {
		return errors.New("message ability is not injected")
	}
	if receiver == nil || receiver.GetUsername() == "" {
		return errors.New("message receiver is empty")
	}

	xmlContent := buildAppMessageXML(title, desc, records)
	_, err := p.message.Send(&message.Message{
		Type:     message.TypeApplication,
		Receiver: receiver,
		Content:  title,
		Data: &message.Message_App{App: &message.AppData{
			SubType: 19,
			Title:   title,
			Desc:    desc,
			Xml:     xmlContent,
		}},
	})
	return err
}

func buildAppMessageXML(title, desc string, records []recordItem) string {
	return fmt.Sprintf(`<appmsg>
	<title>%s</title>
	<des>%s</des>
	<action>view</action>
	<type>19</type>
	<url>https://support.weixin.qq.com/cgi-bin/mmsupport-bin/readtemplate?t=page/favorite_record__w_unsupport&amp;from=singlemessage&amp;isappinstalled=0</url>
	<recorditem>%s</recorditem>
</appmsg>`, escapeXML(title), escapeXML(desc), buildRecordItemXML(title, desc, records))
}

func buildRecordItemXML(title, desc string, records []recordItem) string {
	var builder strings.Builder
	builder.WriteString("<![CDATA[<recordinfo>\n")
	builder.WriteString(fmt.Sprintf("<title>%s</title>\n", escapeXML(title)))
	builder.WriteString(fmt.Sprintf("<desc>%s</desc>\n", escapeXML(desc)))
	builder.WriteString(fmt.Sprintf("<datalist count=\"%d\">\n", len(records)))

	for _, record := range records {
		now := time.Now()
		builder.WriteString(fmt.Sprintf(`<dataitem datatype="1">
	<datadesc>%s</datadesc>
	<sourcename>%s</sourcename>
	<sourceheadurl>%s</sourceheadurl>
	<sourcetime>%s</sourcetime>
	<srcMsgCreateTime>%d</srcMsgCreateTime>
	<fromnewmsgid>%d</fromnewmsgid>
</dataitem>`, escapeXML(record.Content), escapeXML(record.Name), escapeXML(record.Avatar), escapeXML(record.Time), now.Unix(), now.UnixNano()))
		builder.WriteString("\n")
	}

	builder.WriteString("</datalist></recordinfo>]]>")
	return builder.String()
}

func escapeXML(value string) string {
	var buffer bytes.Buffer
	if err := xml.EscapeText(&buffer, []byte(value)); err != nil {
		return value
	}
	return buffer.String()
}
