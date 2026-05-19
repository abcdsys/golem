package plugin

import (
	"fmt"
	"log/slog"
	"slices"

	"github.com/sbgayhub/golem/sdk/command"
)

// dispatcher 事件分发循环
func dispatcher() {
	for e := range events {
		slog.Debug("消费事件", "topic", e.Topic)
		sender := ""
		if e.Sender != nil {
			sender = e.Sender.GetUsername()
		}

		for _, p := range plugins {
			// 跳过禁用的插件
			if p.Config != nil && !p.Config.Enable {
				continue
			}

			// 检查事件主题是否匹配插件订阅
			if !matchesTopic(e.Topic, p.subscriptions) {
				continue
			}

			// 检查发送者是否被允许
			if sender != "" && !isAllowed(sender, p) {
				continue
			}

			// 会话劫持检查：只有劫持插件和 AlwaysRun 插件允许接收
			if sender != "" && !isSessionAllowed(sender, p.Metadata) {
				continue
			}

			// 安全调用插件 OnEvent，捕获 panic
			func() {
				defer func() {
					if r := recover(); r != nil {
						slog.Error("插件处理事件时发生崩溃", "plugin", p.Name, "error", r)
					}
				}()

				if res, err := (*p.eventPlugin).OnEvent(e); err != nil {
					slog.Error("插件处理事件失败", "plugin", p.Name, "res", res, "err", err)
				}

				// 事件分发成功，刷新会话时间
				if sender != "" && isSessionActive(sender) && p.Name == getSessionPlugin(sender) {
					refreshSession(sender)
				}
			}()
		}
	}
}

// DispatchCommand 分发命令给插件
func DispatchCommand(cmd *command.Command, plugins []wrapper) {
	for _, p := range plugins {
		if p.Config != nil && !p.Config.Enable {
			continue
		}

		if !slices.Contains(p.commands, cmd.Cmd) {
			continue
		}

		sender := ""
		if cmd.Sender != nil {
			sender = cmd.Sender.GetUsername()
		}
		if sender != "" && !isAllowed(sender, &p) {
			continue
		}

		func() {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("插件处理命令时发生崩溃", "plugin", p.Name, "error", r)
				}
			}()

			if _, err := (*p.commandPlugin).OnCommand(cmd.Cmd, cmd.Args); err != nil {
				errMsg := fmt.Sprintf("插件[%s]处理命令失败: %v", p.Name, err)
				slog.Error(errMsg)
			}
		}()
	}
}
