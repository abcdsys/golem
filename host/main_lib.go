//go:build lib

package main

import (
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"golem"
	gc "golem/config"
	"golem/pkg/login"

	"github.com/phsym/console-slog"
	"github.com/sbgayhub/golem/host/ability"
	chatroomability "github.com/sbgayhub/golem/host/ability/chatroom"
	contactability "github.com/sbgayhub/golem/host/ability/contact"
	messageability "github.com/sbgayhub/golem/host/ability/message"
	"github.com/sbgayhub/golem/host/ability/sync"
	hc "github.com/sbgayhub/golem/host/config"
	"github.com/sbgayhub/golem/host/plugin"
	sdkcontact "github.com/sbgayhub/golem/sdk/contact"
)

// lib模式
func main() {
	// 加载配置
	cfg := hc.Get()
	// 初始化日志
	initialLog(*cfg)
	messageability.SetOutboundReady(false)

	// 初始化协议层
	if err := golem.Initial(golem.WithConfig(buildGolemConfig(*cfg)), golem.WithSyncCallback(sync.CallBack)); err != nil {
		slog.Warn("初始化协议层出错", "err", err)
		return
	}

	// 检查登录状态
	user, err := login.Check()
	if err != nil {
		slog.Error("登录失败", "err", err)
		return
	}
	contactability.SetSelf(&sdkcontact.SelfInfo{
		Username: user.Username,
		Nickname: user.Nickname,
		Alias:    user.Alias,
		Avatar:   user.Avatar,
		Uin:      user.UIN,
		Email:    user.Email,
		Mobile:   user.Mobile,
	})

	// 初始化联系人能力，加载联系人缓存
	contactability.Initial()
	// 初始化群组能力，加载群组缓存
	chatroomability.Initial()
	messageability.SetOutboundReady(true)

	// 初始化插件管理器
	if err := plugin.Initial(); err != nil {
		slog.Error("插件管理器初始化失败", "err", err)
		return
	}

	// 等待中断信号优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	slog.Info("正在关闭...")
	ability.Destroy()
	plugin.Destroy()
	golem.Stop()
}

func buildGolemConfig(cfg hc.HostConfig) *gc.Config {
	return &gc.Config{
		Core:    cfg.Core,
		Server:  cfg.Server,
		Push:    gc.PushConfig{},
		Storage: cfg.Storage,
		Device:  cfg.Device,
		Log: gc.LogConfig{
			Level:      "info",
			Output:     "console",
			AddSource:  false,
			Ansi:       true,
			MaxSize:    0,
			MaxAge:     0,
			MaxBackups: 0,
			Compress:   false,
		},
		SaveFunc: hc.Save,
	}
}

func initialLog(cfg hc.HostConfig) {
	var level slog.Level
	switch strings.ToLower(cfg.Log.Level) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}
	handler := console.NewHandler(os.Stderr, &console.HandlerOptions{
		Level:      level,
		AddSource:  cfg.Log.AddSource,
		TimeFormat: "2006-01-02 15:04:05",
	})

	slog.SetDefault(slog.New(handler))
}
