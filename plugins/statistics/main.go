package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"github.com/sbgayhub/golem/sdk/chatroom"
	"github.com/sbgayhub/golem/sdk/contact"
	"github.com/sbgayhub/golem/sdk/message"
	"github.com/sbgayhub/golem/sdk/plugin"
)

const capabilityQueryMessages = "statistics.query_messages"

// timeLayout statistics 表 timestamp 列的本地时间格式，也是 since 参数的格式
const timeLayout = "2006-01-02 15:04:05"

func main() {
	plugin.Start(&StatisticsPlugin{})
}

type StatisticsPlugin struct {
	message  message.Ability
	chatroom chatroom.Ability

	mu    sync.Mutex
	dbDir string
	store *store
}

func (p *StatisticsPlugin) GetMetadata() *plugin.Metadata {
	return &plugin.Metadata{
		Name:        "statistics",
		Author:      "ovo",
		Version:     "1.3.0",
		Description: "消息统计插件，记录消息并提供群发言排行/详情；暴露 statistics.query_messages 能力供其它插件查询历史发言",
		Priority:    -1 << 31,
		Next:        true,
		AlwaysRun:   true,
	}
}

func (p *StatisticsPlugin) GetSubscriptions() []string {
	return []string{"message"}
}

// GetCapabilities 声明本插件可被其它插件调用的能力
func (p *StatisticsPlugin) GetCapabilities() []string {
	return []string{capabilityQueryMessages}
}

// OnCall 处理其它插件的调用请求
func (p *StatisticsPlugin) OnCall(capability string, args map[string]string) (string, []byte, error) {
	switch capability {
	case capabilityQueryMessages:
		return p.handleQueryMessages(args)
	default:
		return "", nil, errors.New("unsupported capability: " + capability)
	}
}

// handleQueryMessages 查询历史发言，返回 JSON 数组（时间正序）。
// 入参：member（可选，成员 wxid）、chatroom（可选，群 wxid），两者至少给一个：
// 都给=群内成员；仅 chatroom=整群；仅 member=跨群全局。
// since（可选，本地时间下限，格式 2006-01-02 15:04:05，含边界）、
// since_id（可选，默认0）、limit（可选，默认0=不限）。
func (p *StatisticsPlugin) handleQueryMessages(args map[string]string) (string, []byte, error) {
	member := args["member"]
	chatroom := args["chatroom"]
	if member == "" && chatroom == "" {
		return "", nil, errors.New("member or chatroom is required")
	}
	since := args["since"]
	if since != "" {
		if _, err := time.ParseInLocation(timeLayout, since, time.Local); err != nil {
			return "", nil, fmt.Errorf("invalid since %q (want %s): %w", since, timeLayout, err)
		}
	}
	sinceID, _ := strconv.ParseInt(args["since_id"], 10, 64)
	limit, _ := strconv.Atoi(args["limit"])

	p.mu.Lock()
	store := p.store
	p.mu.Unlock()
	if store == nil {
		return "", nil, errors.New("store is not initialized")
	}

	msgs, err := store.queryMessages(chatroom, member, since, sinceID, limit)
	if err != nil {
		return "", nil, err
	}
	if msgs == nil {
		msgs = []historyMsg{} // 保证返回 [] 而非 null
	}
	data, err := json.Marshal(msgs)
	if err != nil {
		return "", nil, err
	}
	return "json", data, nil
}

func (p *StatisticsPlugin) OnLoad() error {
	return p.ensureStore()
}

func (p *StatisticsPlugin) OnUnload() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.store == nil {
		return nil
	}
	err := p.store.Close()
	p.store = nil
	return err
}

func (p *StatisticsPlugin) OnEnable() error {
	return p.ensureStore()
}

func (p *StatisticsPlugin) OnDisable() error {
	return nil
}

func (p *StatisticsPlugin) OnEvent(event *plugin.Event) (bool, error) {
	msg := event.GetPayload().(*plugin.Event_Message).Message

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.store == nil {
		return false, errors.New("store is not initialized")
	}

	if isRankingKeyword(msg) {
		return p.handleRanking(msg)
	}

	recorded, err := p.store.record(msg)
	if err != nil {
		slog.Warn("[statistics] 记录消息失败", "err", err)
		return false, nil
	}
	return recorded, nil
}

func (p *StatisticsPlugin) ensureStore() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.store != nil {
		return nil
	}

	st, err := openStore(p.dbDir)
	if err != nil {
		return err
	}
	p.store = st
	return nil
}

func (p *StatisticsPlugin) sendText(receiver *contact.Contact, content string, reminds []string) error {
	if p.message == nil {
		return errors.New("message ability is not injected")
	}
	if receiver == nil || receiver.GetUsername() == "" {
		return errors.New("message receiver is empty")
	}

	_, err := p.message.Send(&message.Message{
		Type:     message.TypeText,
		Receiver: receiver,
		Content:  content,
		Data: &message.Message_Text{Text: &message.TextData{
			Content: content,
			Reminds: reminds,
		}},
	})
	return err
}
