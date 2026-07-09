package main

import (
	"regexp"
)

const (
	defaultHTTPTimeoutSeconds = 15
	defaultSendType           = "text"
)

var templateNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

type Rule struct {
	ID                 string   `toml:"id" comment:"规则 ID"`
	Keywords           []string `toml:"keywords" comment:"关键词列表，消息第一段全文匹配"`
	URL                string   `toml:"url" comment:"请求地址模板"`
	Method             string   `toml:"method" comment:"请求方法"`
	Headers            string   `toml:"headers" comment:"请求头，格式：A=B;C=D，支持模板变量"`
	Body               string   `toml:"body" comment:"请求体模板"`
	SendType           string   `toml:"send_type" comment:"发送类型：text|emoji|image|video"`
	ResultPath         string   `toml:"result_path" comment:"gjson 路径，空则使用响应 body 原文"`
	At                 bool     `toml:"at" comment:"是否在文本回复中 @ 参数或引用对应用户"`
	Enabled            *bool    `toml:"enabled,omitempty" comment:"是否启用，空值按启用处理"`
}

type incomingText struct {
	Text    string
	Keyword string
	Param   string
}

type quoteInfo struct {
	Username string
	Quoter   string
	Quote    string
}

type quoteRefer struct {
	DisplayName string `xml:"displayname"`
	FromUser    string `xml:"fromusr"`
	ChatUser    string `xml:"chatusr"`
	Content     string `xml:"content"`
}

type mentionTarget struct {
	Username    string
	DisplayName string
}
