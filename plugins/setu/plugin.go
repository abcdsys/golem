package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/sbgayhub/golem/sdk/cdn"
	"github.com/sbgayhub/golem/sdk/contact"
	"github.com/sbgayhub/golem/sdk/message"
	"github.com/sbgayhub/golem/sdk/plugin"
)

// SetuPlugin 色图插件
type SetuPlugin struct {
	plugin.ConfigAbility[Config]
	message message.Ability
	contact contact.Ability
	cdn     cdn.Ability
	client  *http.Client
}

// newHTTPClient 创建带自定义重定向处理的 HTTP 客户端
func newHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("过多重定向")
			}
			// 处理 Location 头中可能存在的引号
			loc := req.Response.Header.Get("Location")
			if loc != "" {
				// 去除引号
				loc = strings.Trim(loc, "'\"")
				if parsed, err := url.Parse(loc); err == nil {
					req.URL = req.URL.ResolveReference(parsed)
				}
			}
			return nil
		},
	}
}

// GetMetadata 返回插件元数据
func (p *SetuPlugin) GetMetadata() *plugin.Metadata {
	return &plugin.Metadata{
		Name:        "setu",
		Author:      "Golem Team",
		Version:     "1.0.1",
		Description: "色图插件 - 提供各种图片和视频",
		Priority:    -100,
	}
}

func (p *SetuPlugin) OnLoad() error {
	slog.Info("[setu] 色图插件加载成功",
		"img_url", p.Config.ImgURL,
		"video_rate", p.Config.VideoRate,
	)
	return nil
}

func (p *SetuPlugin) OnUnload() error {
	slog.Info("[setu] 色图插件已卸载")
	return nil
}

// GetSubscriptions 订阅事件
func (p *SetuPlugin) GetSubscriptions() []string {
	return []string{
		message.TypeText.Topic,
	}
}

// OnEvent 处理事件
func (p *SetuPlugin) OnEvent(e *plugin.Event) (bool, error) {
	msg := e.Payload.(*plugin.Event_Message).Message
	if msg == nil {
		return false, nil
	}

	text := strings.TrimSpace(msg.GetContent())
	if text == "" {
		return false, nil
	}

	// 获取接收者
	receiver := p.contact.Get(e.GetSender())
	if receiver == nil {
		slog.Warn("[setu] 未找到接收者", "sender", e.GetSender())
		return false, nil
	}

	// 匹配关键词
	switch text {
	case "setu帮助", "色图帮助":
		return p.handleHelp(receiver)
	case "plmm", "漂亮妹妹", "来点美女":
		return p.handlePlmm(receiver)
	case "来点黑丝":
		return p.handleSiImage(receiver, "黑丝", p.Config.HeisiVideoURL, p.Config.HeisiURL)
	case "来点白丝":
		return p.handleSiImage(receiver, "白丝", p.Config.BaisiVideoURL, p.Config.BaisiURL)
	case "看看腿":
		return p.handleKkt(receiver)
	case "来点帅哥":
		return p.handleBoy(receiver)
	}

	// 前缀匹配：来点XX（搜索）
	if strings.HasPrefix(text, "来点") && len([]rune(text)) > 2 {
		keyword := string([]rune(text)[2:])
		return p.handleSearch(receiver, keyword)
	}

	return false, nil
}
