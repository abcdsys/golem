package plugin

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/sbgayhub/golem/sdk/contact"
	sdk "github.com/sbgayhub/golem/sdk/plugin"
)

// IPluginConfig 插件配置接口（宿主侧 wrapper 实现，插件作者无需关心）
type IPluginConfig interface {
	GetDefaultConfig() ([]byte, error) // 获取插件默认配置
	SetConfig(data []byte) error       // 注入插件配置
}

// --- HostServiceServer 实现 ---

type hostService struct {
	sdk.UnimplementedHostServiceServer
}

func (h *hostService) SessionHold(_ context.Context, req *sdk.SessionHold_Request) (*sdk.SessionHold_Response, error) {
	duration := time.Duration(req.Duration) * time.Second

	sessionMu.Lock()
	defer sessionMu.Unlock()

	if s, ok := sessions[req.Sender]; ok {
		s.Timer.Stop()
		slog.Info("释放已有会话", "plugin", s.PluginName, "sender", req.Sender)
	}

	s := &session{
		PluginName:    req.PluginId,
		Sender:        req.Sender,
		SenderContact: &contact.Contact{Username: req.Sender},
		Duration:      duration,
		ExpireAt:      time.Now().Add(duration),
	}
	s.Timer = newSessionTimer(s)
	sessions[req.Sender] = s

	slog.Info("插件劫持会话", "plugin", req.PluginId, "sender", req.Sender, "duration", duration)
	return &sdk.SessionHold_Response{}, nil
}

func (h *hostService) SessionRelease(_ context.Context, req *sdk.SessionRelease_Request) (*sdk.SessionRelease_Response, error) {
	sessionRelease(req.Sender)
	return &sdk.SessionRelease_Response{}, nil
}

func (h *hostService) CallPlugin(_ context.Context, req *sdk.CallPlugin_Request) (*sdk.CallPlugin_Response, error) {
	mu.Lock()
	target := findWrapper(req.PluginId)
	mu.Unlock()

	if target == nil {
		return &sdk.CallPlugin_Response{Value: "插件不存在: " + req.PluginId}, nil
	}

	if target.commandPlugin == nil {
		return &sdk.CallPlugin_Response{Value: "插件不支持命令"}, nil
	}

	// 反序列化 args (bytes → map[string]string)
	var args map[string]string
	if len(req.Args) > 0 {
		_ = json.Unmarshal(req.Args, &args)
	}

	result, err := (*target.commandPlugin).OnCommand(req.Method, args)
	if err != nil {
		return &sdk.CallPlugin_Response{Value: err.Error()}, nil
	}
	return &sdk.CallPlugin_Response{Value: result}, nil
}
