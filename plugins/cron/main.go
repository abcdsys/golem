package main

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/robfig/cron/v3"
	"github.com/sbgayhub/golem/sdk/contact"
	"github.com/sbgayhub/golem/sdk/message"
	"github.com/sbgayhub/golem/sdk/plugin"
)

type CronPlugin struct {
	plugin.ConfigAbility[CronConfig]
	cron    *cron.Cron
	entries []cron.EntryID
	caller  plugin.CallerAbility
	message message.Ability
	contact contact.Ability
}

type CronConfig struct {
	Jobs []Config `toml:"jobs"`
}

type Config struct {
	Cron       string            `toml:"cron"`
	Target     []string          `toml:"target"`
	Capability string            `toml:"capability"`
	Args       map[string]string `toml:"args"`
}

func (c *CronPlugin) GetMetadata() *plugin.Metadata {
	return &plugin.Metadata{
		Name:        "cron",
		Description: "定时任务插件，根据配置定时调用能力",
		Version:     "0.0.1",
		Author:      "golem",
	}
}

func (c *CronPlugin) OnLoad() error {
	c.ensureCron()
	c.entries = make([]cron.EntryID, len(c.Config.Jobs))
	for i, config := range c.Config.Jobs {
		if id, err := c.scheduleConfig(config); err != nil {
			slog.Error("[cron] 定时任务创建失败", "cron", config.Cron, "capability", config.Capability, "err", err)
		} else {
			c.entries[i] = id
			slog.Info("[cron] 定时任务创建成功", "id", id, "cron", config.Cron, "capability", config.Capability)
		}
	}
	c.cron.Start()
	return nil
}

func (c *CronPlugin) OnUnload() error {
	if c.cron != nil {
		c.cron.Stop()
	}
	return nil
}

func (c *CronPlugin) OnEnable() error { return nil }

func (c *CronPlugin) OnDisable() error { return nil }

func (c *CronPlugin) ensureCron() {
	if c.cron == nil {
		c.cron = cron.New()
	}
}

func (c *CronPlugin) scheduleConfig(config Config) (cron.EntryID, error) {
	c.ensureCron()
	return c.cron.AddFunc(config.Cron, c.handleCron(config))
}

func (c *CronPlugin) handleCron(config Config) func() {
	return func() {
		if c.caller == nil {
			slog.Error("[cron] 插件调用能力未注入", "cron", config.Cron, "capability", config.Capability)
			return
		}

		for _, target := range config.Target {
			target = strings.TrimSpace(target)
			if target == "" {
				continue
			}

			args := cloneArgs(config.Args)
			args["receiver"] = target

			mime, data, err := c.caller.CallPlugin(config.Capability, args)
			if err != nil {
				slog.Error("[cron] 定时任务调用能力失败", "cron", config.Cron, "capability", config.Capability, "target", target, "err", err)
				continue
			}

			mime = cleanMime(mime)
			if mime == "" || mime == "none" {
				slog.Info("[cron] 定时任务执行成功", "cron", config.Cron, "capability", config.Capability, "target", target, "mime", mime)
				continue
			}

			if err := c.sendResult(target, mime, data); err != nil {
				slog.Error("[cron] 定时任务发送结果失败", "cron", config.Cron, "capability", config.Capability, "target", target, "mime", mime, "err", err)
				continue
			}
			slog.Info("[cron] 定时任务执行并发送结果成功", "cron", config.Cron, "capability", config.Capability, "target", target, "mime", mime)
		}
	}
}

func (c *CronPlugin) sendResult(receiver, mime string, data []byte) error {
	if c.message == nil {
		return errors.New("message ability is not injected")
	}
	msg, err := buildMessage(receiver, mime, data)
	if err != nil {
		return err
	}
	_, err = c.message.Send(msg)
	return err
}

func buildMessage(receiver, mime string, data []byte) (*message.Message, error) {
	receiver = strings.TrimSpace(receiver)
	if receiver == "" {
		return nil, errors.New("receiver is empty")
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("%s data is empty", mime)
	}

	msg := &message.Message{
		Receiver: &contact.Contact{Username: receiver},
		Content:  string(data),
	}
	media := &message.Media{Data: data}
	switch cleanMime(mime) {
	case "text", "json":
		msg.Type = message.TypeText
		msg.Data = &message.Message_Text{Text: &message.TextData{Content: string(data)}}
	case "image":
		msg.Type = message.TypeImage
		msg.Content = "图片消息"
		msg.Data = &message.Message_Image{Image: &message.ImageData{Media: media}}
	case "voice":
		msg.Type = message.TypeVoice
		msg.Content = "语音消息"
		msg.Data = &message.Message_Voice{Voice: &message.VoiceData{Media: media}}
	case "video":
		msg.Type = message.TypeVideo
		msg.Content = "视频消息"
		msg.Data = &message.Message_Video{Video: &message.VideoData{Media: media}}
	default:
		return nil, fmt.Errorf("unsupported mime: %s", mime)
	}
	return msg, nil
}

func cloneArgs(args map[string]string) map[string]string {
	clone := make(map[string]string, len(args)+1)
	for key, value := range args {
		clone[key] = value
	}
	return clone
}

func cleanMime(mime string) string {
	return strings.ToLower(strings.TrimSpace(mime))
}

func main() {
	p, err := newCronPlugin()
	if err != nil {
		slog.Error("[cron] 初始化失败", "err", err)
		return
	}
	plugin.Start(p)
}
