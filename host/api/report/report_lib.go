//go:build lib

// Package reportapi 提供状态通知服务的 lib 实现（直接调用底层实现）。
package reportapi

import (
	"sync"

	"golem/pkg/report"
)

// lib 状态通知服务 lib 实现（直接调用底层实现）。
type lib struct{}

// Get 获取 ReportService 单例（lib 模式）。
var Get = sync.OnceValue(func() ReportService {
	return &lib{}
})

// StartTyping 通知对方正在输入
func (l lib) StartTyping(receiver string) error {
	report.StartTyping(receiver)
	return nil
}

// StopTyping 通知对方停止输入
func (l lib) StopTyping(receiver string) error {
	report.StopTyping(receiver)
	return nil
}

// ReadMessage 通知对方消息已读
func (l lib) ReadMessage(receiver string) error {
	report.ReadMessage(receiver)
	return nil
}
