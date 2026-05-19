package plugin

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"
)

// HostServiceImpl 宿主侧 HostService 实现，由宿主在初始化时设置
var HostServiceImpl HostServiceServer

// --- SessionClient 会话能力 gRPC 客户端包装（注入到插件） ---

type SessionClient struct {
	Client HostServiceClient
}

func (c SessionClient) Hold(name string, sender string, duration time.Duration) {
	if _, err := c.Client.SessionHold(context.Background(), &SessionHold_Request{
		PluginId: name,
		Sender:   sender,
		Duration: uint32(duration.Seconds()),
	}); err != nil {
		slog.Error("[session] 劫持会话失败", "plugin", name, "sender", sender, "err", err)
	}
}

func (c SessionClient) Release(sender string) {
	if _, err := c.Client.SessionRelease(context.Background(), &SessionRelease_Request{
		Sender: sender,
	}); err != nil {
		slog.Error("[session] 释放会话失败", "sender", sender, "err", err)
	}
}

// --- CallerClient 插件调用能力 gRPC 客户端包装（注入到插件） ---

type CallerClient struct {
	Client HostServiceClient
}

func (c CallerClient) CallPlugin(pluginId string, method string, args map[string]string) (string, error) {
	data, _ := json.Marshal(args)
	resp, err := c.Client.CallPlugin(context.Background(), &CallPlugin_Request{
		PluginId: pluginId,
		Method:   method,
		Args:     data,
	})
	if err != nil {
		return "", err
	}
	return resp.Value, nil
}
