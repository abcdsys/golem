//go:build !lib

// Package reportapi 提供状态通知服务的 web 实现（通过 HTTP 调用远程服务）。
package reportapi

import (
	"errors"
	"sync"

	"github.com/sbgayhub/golem/host/api"
)

// web 状态通知服务 web 实现（通过 HTTP 调用远程服务）。
type web struct{}

// Get 获取 ReportService 单例（web 模式）。
var Get = sync.OnceValue(func() ReportService {
	return &web{}
})

// StartTyping 通知对方正在输入
func (w web) StartTyping(receiver string) error {
	_, err := api.GetHttp().Post("/api/message/start").Query("receiver", receiver).Do()
	return err
}

// StopTyping 通知对方停止输入
func (w web) StopTyping(receiver string) error {
	_, err := api.GetHttp().Post("/api/message/stop").Query("receiver", receiver).Do()
	return err
}

// ReadMessage 通知对方消息已读
func (w web) ReadMessage(receiver string) error {
	return errors.New("ReadMessage not supported in web mode: swagger does not define an HTTP route")
}
