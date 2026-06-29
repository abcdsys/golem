package main

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"slices"
	"sync"
	"time"

	"github.com/sbgayhub/golem/sdk/contact"
	"github.com/sbgayhub/golem/sdk/message"
	"github.com/sbgayhub/golem/sdk/plugin"
)

var keywords = []string{"今日新闻", "今日图卦"}

type CacheEntry struct {
	Data []byte
	Date string
}

type NewsCache struct {
	mu      sync.RWMutex
	news    *CacheEntry
	diagram [2]*CacheEntry
}

type NewsPlugin struct {
	message message.Ability
	contact contact.Ability
	cache   *NewsCache
}

func (n *NewsPlugin) GetMetadata() *plugin.Metadata {
	return &plugin.Metadata{
		Name:        "news",
		Author:      "ovo",
		Version:     "v0.0.1",
		Description: "新闻插件，根据“今日新闻”、“今日图卦”关键词返回新闻图片",
		Priority:    0,
		Next:        false,
		AlwaysRun:   false,
	}
}

func (n *NewsPlugin) GetSubscriptions() []string {
	return []string{message.TypeText.Topic}
}

func (n *NewsPlugin) GetCapabilities() []string {
	return []string{"news.today", "news.diagram"}
}

func (n *NewsPlugin) OnCall(capability string, args map[string]string) (string, []byte, error) {
	receiver, ex := args["receiver"]
	if !ex || receiver == "" {
		return "", nil, errors.New("receiver 不可为空")
	}
	c := n.contact.Get(receiver)
	if c == nil {
		return "", nil, errors.New("未找到联系人：" + receiver)
	}
	switch capability {
	case "news.today":
		if _, err := n.news(c); err != nil {
			return "", nil, err
		}
		return "none", nil, nil
	case "news.diagram":
		if _, err := n.diagram(c); err != nil {
			return "", nil, err
		}
		return "none", nil, nil
	default:
		return "", nil, errors.New("不支持：" + capability)
	}
}

func (n *NewsPlugin) OnEvent(event *plugin.Event) (bool, error) {
	msg := event.Payload.(*plugin.Event_Message).Message
	if slices.Contains(keywords, msg.Content) {
		switch msg.Content {
		case "今日新闻":
			return n.news(msg.Sender)
		case "今日图卦":
			return n.diagram(msg.Sender)
		default:
			slog.Warn("暂不支持：" + msg.Content)
		}
	}
	return false, nil
}

func (n *NewsPlugin) news(receiver *contact.Contact) (bool, error) {
	today := time.Now().Format("2006-01-02")

	// 尝试从缓存读取
	n.cache.mu.RLock()
	if n.cache.news != nil && n.cache.news.Date == today {
		imageData := n.cache.news.Data
		n.cache.mu.RUnlock()
		slog.Info("[今日新闻] 使用缓存数据", "date", today)
		return n.sendImage(receiver, "今日新闻", imageData)
	}
	n.cache.mu.RUnlock()

	// 缓存未命中或已过期，重新请求
	slog.Info("[今日新闻] 缓存未命中，开始请求", "date", today)
	imageData, err := n.fetchNews(today)
	if err != nil {
		return false, err
	}

	// 更新缓存
	n.cache.mu.Lock()
	n.cache.news = &CacheEntry{
		Data: imageData,
		Date: today,
	}
	n.cache.mu.Unlock()

	return n.sendImage(receiver, "今日新闻", imageData)
}

// fetchNews 获取今日新闻图片数据
func (n *NewsPlugin) fetchNews(date string) ([]byte, error) {
	url := fmt.Sprintf("https://cdn.jsdmirror.com/gh/vikiboss/60s-static-host@main/static/images/%s.png", date)
	resp, err := http.DefaultClient.Get(url)
	if err != nil {
		return nil, errors.New("[今日新闻] 请求失败")
	}
	defer func() { _ = resp.Body.Close() }()

	imageData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("[今日新闻] 读取响应失败：%w", err)
	}

	return imageData, nil
}

func (n *NewsPlugin) diagram(receiver *contact.Contact) (bool, error) {
	// 获取一天前的日期
	t := time.Now().AddDate(0, 0, -1)
	today := t.Format("2006-01-02")
	month := t.Format("200601")
	date := t.Format("20060102")

	// 检查缓存
	n.cache.mu.RLock()
	allCached := true
	for i := range 2 {
		if n.cache.diagram[i] == nil || n.cache.diagram[i].Date != today {
			allCached = false
			break
		}
	}
	if allCached {
		// 使用缓存数据
		slog.Info("[今日图卦] 使用缓存数据", "date", today)
		for i := range 2 {
			imageData := n.cache.diagram[i].Data
			n.cache.mu.RUnlock()
			if i > 0 {
				time.Sleep(1 * time.Second)
			}
			if _, err := n.sendImage(receiver, "今日图卦", imageData); err != nil {
				return false, err
			}
			n.cache.mu.RLock()
		}
		n.cache.mu.RUnlock()
		return true, nil
	}
	n.cache.mu.RUnlock()

	// 缓存未命中或已过期，重新请求
	slog.Info("[今日图卦] 缓存未命中，开始请求", "date", today)
	template := "https://penti.5aihj.com/%s/tugua/%s%d.jpg"
	var imageDataList [2][]byte

	for i := range 2 {
		if i > 0 {
			time.Sleep(1 * time.Second)
		}

		url := fmt.Sprintf(template, month, date, i+1)
		imageData, err := n.fetchDiagram(url)
		if err != nil {
			return false, err
		}

		imageDataList[i] = imageData

		// 发送图片
		if _, err := n.sendImage(receiver, "今日图卦", imageData); err != nil {
			return false, err
		}
	}

	// 更新缓存
	n.cache.mu.Lock()
	for i := range 2 {
		n.cache.diagram[i] = &CacheEntry{
			Data: imageDataList[i],
			Date: today,
		}
	}
	n.cache.mu.Unlock()

	return true, nil
}

// fetchDiagram 获取图卦图片数据
func (n *NewsPlugin) fetchDiagram(url string) ([]byte, error) {
	resp, err := http.DefaultClient.Get(url)
	if err != nil {
		return nil, errors.New("[今日图卦] 请求失败")
	}
	defer func() { _ = resp.Body.Close() }()

	imageData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("[今日图卦] 读取响应失败：%w", err)
	}

	return imageData, nil
}

// sendImage 发送图片消息
func (n *NewsPlugin) sendImage(receiver *contact.Contact, content string, imageData []byte) (bool, error) {
	data := &message.Message{
		Content:  content,
		Receiver: receiver,
		Type:     message.TypeImage,
		Data:     &message.Message_Image{Image: &message.ImageData{Media: &message.Media{Data: imageData}}},
	}
	if _, err := n.message.Send(data); err != nil {
		return false, fmt.Errorf("[%s] 发送消息失败：%w", content, err)
	}
	return true, nil
}

func main() {
	plugin.Start(&NewsPlugin{
		cache: &NewsCache{},
	})
}
