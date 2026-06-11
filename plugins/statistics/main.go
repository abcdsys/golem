package main

import (
	"errors"
	"log/slog"
	"sync"

	"github.com/sbgayhub/golem/sdk/chatroom"
	"github.com/sbgayhub/golem/sdk/contact"
	"github.com/sbgayhub/golem/sdk/message"
	"github.com/sbgayhub/golem/sdk/plugin"
)

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
		Version:     "1.0.0",
		Description: "消息统计插件，记录消息并提供群发言排行和详情",
		Priority:    -1 << 31,
		Next:        true,
		AlwaysRun:   true,
	}
}

func (p *StatisticsPlugin) GetSubscriptions() []string {
	return []string{"message"}
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
