// Package loginability 提供登录能力的实现。
package loginability

import (
	sdk "github.com/sbgayhub/golem/sdk/login"

	api "github.com/sbgayhub/golem/host/api/login"
	"github.com/sbgayhub/golem/host/api/util"
)

// ability 登录能力实现（直连型）
type ability struct {
	api api.LoginService
}

func init() {
	sdk.Instance = &ability{api: api.Get()}
}

// Login 执行扫码登录
func (a ability) Login() (*sdk.QRCodeResult, error) {
	resp, err := a.api.Login()
	if resp == nil || err != nil {
		return nil, err
	}
	var result sdk.QRCodeResult
	if err := util.TransformProto(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Init 首次登录后初始化
func (a ability) Init() (*sdk.InitResponse, error) {
	resp, err := a.api.Init()
	if resp == nil || err != nil {
		return nil, err
	}
	var result sdk.InitResponse
	if err := util.TransformProto(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Refresh 刷新登录状态
func (a ability) Refresh() (*sdk.OperateResponse, error) {
	resp, err := a.api.Refresh()
	if resp == nil || err != nil {
		return nil, err
	}
	var result sdk.OperateResponse
	if err := util.TransformProto(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Wakeup 唤醒登录
func (a ability) Wakeup() (*sdk.OperateResponse, error) {
	resp, err := a.api.Wakeup()
	if resp == nil || err != nil {
		return nil, err
	}
	var result sdk.OperateResponse
	if err := util.TransformProto(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Logout 登出
func (a ability) Logout() (*sdk.OperateResponse, error) {
	resp, err := a.api.Logout()
	if resp == nil || err != nil {
		return nil, err
	}
	var result sdk.OperateResponse
	if err := util.TransformProto(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// PasswordLogin 使用账号密码登录
func (a ability) PasswordLogin(req *sdk.PasswordLoginRequest) (*sdk.PasswordLoginResult, error) {
	var apiReq api.PasswordLoginRequest
	if err := util.TransformProto(req, &apiReq); err != nil {
		return nil, err
	}
	resp, err := a.api.PasswordLogin(&apiReq)
	if resp == nil || err != nil {
		return nil, err
	}
	var result sdk.PasswordLoginResult
	if err := util.TransformProto(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
