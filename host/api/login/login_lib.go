//go:build lib

// Package loginapi 提供登录服务的 lib 实现（直接调用底层实现）。
package loginapi

import (
	"sync"

	"golem/pkg/login"
)

// lib 登录服务 lib 实现
type lib struct{}

// Get 获取 LoginService 单例（lib 模式）
var Get = sync.OnceValue(func() LoginService {
	return &lib{}
})

// Login 执行扫码登录
func (l lib) Login() (*QRCodeResult, error) {
	resp, err := login.Login()
	if resp == nil || err != nil {
		return nil, err
	}
	return &QRCodeResult{
		Uuid:        resp.UUID,
		Data:        resp.Data,
		Url:         resp.Url,
		CheckTime:   resp.CheckTime,
		ExpiredTime: resp.ExpiredTime,
	}, nil
}

// Init 首次登录后初始化
func (l lib) Init() (*InitResponse, error) {
	_, err := login.Init()
	if err != nil {
		return nil, err
	}
	return &InitResponse{Code: 0, Message: "ok"}, nil
}

// Refresh 刷新登录状态
func (l lib) Refresh() (*OperateResponse, error) {
	err := login.Refresh()
	if err != nil {
		return &OperateResponse{Code: -1, Message: err.Error()}, nil
	}
	return &OperateResponse{Code: 0, Message: "ok"}, nil
}

// Wakeup 唤醒登录
func (l lib) Wakeup() (*OperateResponse, error) {
	err := login.Wakeup()
	if err != nil {
		return &OperateResponse{Code: -1, Message: err.Error()}, nil
	}
	return &OperateResponse{Code: 0, Message: "ok"}, nil
}

// Logout 登出
func (l lib) Logout() (*OperateResponse, error) {
	err := login.Logout()
	if err != nil {
		return &OperateResponse{Code: -1, Message: err.Error()}, nil
	}
	return &OperateResponse{Code: 0, Message: "ok"}, nil
}

// PasswordLogin 使用账号密码登录
func (l lib) PasswordLogin(req *PasswordLoginRequest) (*PasswordLoginResult, error) {
	loginReq := login.PasswordLoginRequest{
		Username:   req.Username,
		Password:   req.Password,
		DeviceType: req.DeviceType,
		DeviceName: req.DeviceName,
		DeviceID:   req.DeviceId,
	}
	resp, err := login.PasswordLogin(&loginReq)
	if resp == nil || err != nil {
		return nil, err
	}
	return &PasswordLoginResult{
		Uin:      resp.UIN,
		Username: resp.Username,
		Nickname: resp.Nickname,
		Alias:    resp.Alias,
		Email:    resp.Email,
		Mobile:   resp.Mobile,
	}, nil
}
