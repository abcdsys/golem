//go:build !lib

// Package loginapi 提供登录服务的 web 实现（通过 HTTP 调用远程服务）。
package loginapi

import (
	"sync"

	"github.com/sbgayhub/golem/host/api/util"
)

// web 登录服务 web 实现
type web struct{}

// Get 获取 LoginService 单例（web 模式）
var Get = sync.OnceValue(func() LoginService {
	return &web{}
})

// Login 执行扫码登录
func (w web) Login() (*QRCodeResult, error) {
	data, err := util.GetHttp().Get("/login/login")
	if err != nil {
		return nil, err
	}
	var resp QRCodeResult
	if err := util.ParseProtoResponse(data, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Init 首次登录后初始化
func (w web) Init() (*InitResponse, error) {
	data, err := util.GetHttp().Get("/login/init")
	if err != nil {
		return nil, err
	}
	var resp InitResponse
	if err := util.ParseProtoResponse(data, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Refresh 刷新登录状态
func (w web) Refresh() (*OperateResponse, error) {
	data, err := util.GetHttp().Get("/login/refresh")
	if err != nil {
		return nil, err
	}
	var resp OperateResponse
	if err := util.ParseProtoResponse(data, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Wakeup 唤醒登录
func (w web) Wakeup() (*OperateResponse, error) {
	data, err := util.GetHttp().Get("/login/wakeup")
	if err != nil {
		return nil, err
	}
	var resp OperateResponse
	if err := util.ParseProtoResponse(data, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Logout 登出
func (w web) Logout() (*OperateResponse, error) {
	data, err := util.GetHttp().Get("/login/logout")
	if err != nil {
		return nil, err
	}
	var resp OperateResponse
	if err := util.ParseProtoResponse(data, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// PasswordLogin 使用账号密码登录
func (w web) PasswordLogin(req *PasswordLoginRequest) (*PasswordLoginResult, error) {
	data, err := util.GetHttp().Post("/login/password", req)
	if err != nil {
		return nil, err
	}
	var resp PasswordLoginResult
	if err := util.ParseProtoResponse(data, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
