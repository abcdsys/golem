package main

import (
	"log/slog"

	"github.com/sbgayhub/golem/sdk/chatroom"
	"github.com/sbgayhub/golem/sdk/contact"
	"github.com/sbgayhub/golem/sdk/message"
	"github.com/sbgayhub/golem/sdk/plugin"
)

func main() {
	p := &FakeForwardPlugin{}

	if err := plugin.RegisterCommand(p.handle); err != nil {
		slog.Error("[fake_forward] 注册 chat 命令失败", "err", err)
		return
	}

	plugin.Start(p)
}

// FakeForwardPlugin 伪转发插件。
// 使用 /fake chat 名字1:内容1|名字2:内容2 创建假的聊天记录。
type FakeForwardPlugin struct {
	contact  contact.Ability
	chatroom chatroom.Ability
	message  message.Ability
}

// ============================================================================
// 元数据与命令接口
// ============================================================================

func (p *FakeForwardPlugin) GetMetadata() *plugin.Metadata {
	return &plugin.Metadata{
		Name:        "fake_forward",
		Author:      "golem",
		Version:     "2.0.0",
		Description: "伪转发插件，支持手动编辑假聊天记录",
		Next:        false,
		Priority:    0,
		AlwaysRun:   false,
	}
}

func (p *FakeForwardPlugin) GetCommands() []string {
	return plugin.CommandCommands()
}

func (p *FakeForwardPlugin) OnCommand(cmd *plugin.Command) (string, error) {
	return plugin.DispatchCommand(cmd)
}
