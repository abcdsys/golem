package main

import (
	"log/slog"
	"strings"
	"time"

	"github.com/sbgayhub/golem/sdk/contact"
	"github.com/sbgayhub/golem/sdk/message"
	"github.com/sbgayhub/golem/sdk/plugin"
)

type Config struct {
	Name    string `toml:"name" comment:"姓名"`
	Age     int32  `toml:"age" comment:"年龄"`
	Address string `toml:"address" comment:"地址"`
}

type ExamplePlugin struct {
	plugin.ConfigAbility[Config]
	timer   *time.Timer
	message message.Ability
	session plugin.SessionAbility
	//commands *plugin.CommandRegistry
}

type ExampleEcho struct {
	_      struct{} `cmd:"example echo" help:"回显文本" usage:"/example echo <text> [--prefix 前缀]" example:"/example echo hello --prefix 测试"`
	Text   string   `arg:"text" help:"回显内容" required:"true" variadic:"true"`
	Prefix string   `flag:"prefix" help:"回显前缀"`
	Upper  bool     `flag:"upper" help:"转换为大写" value:"false"`
}

func (p *ExamplePlugin) GetMetadata() *plugin.Metadata {
	return &plugin.Metadata{
		Name:        "example",
		Author:      "ovo",
		Version:     "1.0.0",
		Description: "example plugin",
		Next:        false,
		Priority:    0,
		AlwaysRun:   false,
	}
}

func (p *ExamplePlugin) GetSubscriptions() []string {
	return []string{"message::text", "session::expired"}
}

func (p *ExamplePlugin) GetCommands() []string {
	return plugin.CommandCommands()
}

func (p *ExamplePlugin) OnCommand(cmd *plugin.Command) (string, error) {
	return plugin.DispatchCommand(cmd)
}

func (p *ExamplePlugin) handleEcho(echo ExampleEcho) (string, error) {
	text := echo.Text
	if echo.Upper {
		text = strings.ToUpper(text)
	}
	if echo.Prefix != "" {
		text = echo.Prefix + text
	}
	return text, nil
}

func (p *ExamplePlugin) OnEvent(e *plugin.Event) (bool, error) {
	slog.Error("接收到事件", "topic", e.Topic)
	if e.Topic == "session::expired" {
		slog.Info("接收到会话过期事件")
		p.message.Send(&message.Message{Receiver: &contact.Contact{Username: e.Sender}, Content: "会话到期咯"})
		return true, nil
	}

	msg := e.Payload.(*plugin.Event_Message)
	slog.Info("[example] 获取配置", "config", p.Config)
	p.Config.Name = msg.Message.Content

	p.session.Hold(p, e.Sender, 10*time.Second)
	p.timer.Reset(10 * time.Second)

	if err := p.SaveConfig(p); err != nil {
		slog.Error("保存配置出现错误", "err", err)
		return false, err
	}

	//// 使用新的统一 Message 接口发送消息
	//msg := &message.Message{
	//	Type:     message.MessageType_MESSAGE_TYPE_TEXT,
	//	Receiver: &contact.Contact{Username: "wxid_hello"},
	//	Data: &message.Message_Text{Text: &message.TextData{
	//		Content: "this is a test message",
	//	}},
	//}
	//
	//if _, err := p.message.Send(msg); err != nil {
	//	slog.Error("[example] 发送消息失败", "err", err)
	//	return false, err
	//} else {
	//	slog.Error("[example] 发送消息成功")
	//}

	return true, nil
}

func main() {
	p := ExamplePlugin{
		timer: time.NewTimer(10 * time.Second),
	}
	if err := plugin.RegisterCommand(p.handleEcho); err != nil {
		slog.Error("注册命令失败", "err", err)
		return
	}
	go func() {
		slog.Info("插件已启动，等待会话到期")
		<-p.timer.C
		slog.Info("会话到期")
		p.message.Send(&message.Message{Receiver: &contact.Contact{Username: "ovo"}, Content: "会话到期咯"})
	}()

	plugin.Start(&p)
}
