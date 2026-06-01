//go:build !lib

// Package paymentapi 提供支付服务的 web 实现（通过 HTTP 调用远程服务）。
package paymentapi

import (
	"sync"

	"github.com/sbgayhub/golem/host/api"
)

// web 支付服务 web 实现（通过 HTTP 调用远程服务）。
type web struct{}

// Get 获取 PaymentService 单例（web 模式）。
var Get = sync.OnceValue(func() PaymentService {
	return &web{}
})

// GeneratePayQCode 生成支付收款二维码
func (w web) GeneratePayQCode() (*F2FQrcodeResponse, error) {
	var resp F2FQrcodeResponse
	if err := api.GetHttp().Get("/api/payment/qrcode").DoProto(&resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetBandCardList 获取已绑定银行卡及余额
func (w web) GetBandCardList() (*TenPayResponse, error) {
	var resp TenPayResponse
	if err := api.GetHttp().Get("/api/payment/cards").DoProto(&resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// CreateHongBao 创建红包
func (w web) CreateHongBao(params CreateHongBaoParams) (*HongBaoResponse, error) {
	var resp HongBaoResponse
	if err := api.GetHttp().Post("/api/payment/hongbao/create").Body(map[string]any{
		"hb_type":  params.HBType,
		"username": params.Username,
		"in_way":   params.InWay,
		"count":    params.Count,
		"amount":   params.Amount,
		"wishing":  params.Wishing,
	}).DoProto(&resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ReceiveHongBao 接收红包
func (w web) ReceiveHongBao(params ReceiveHongBaoParams) (*HongBaoResponse, error) {
	var resp HongBaoResponse
	if err := api.GetHttp().Post("/api/payment/hongbao/receive").Body(map[string]any{
		"native_url": params.NativeURL,
		"in_way":     params.InWay,
	}).DoProto(&resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// OpenHongBao 打开红包
func (w web) OpenHongBao(params OpenHongBaoParams) (*HongBaoResponse, error) {
	var resp HongBaoResponse
	if err := api.GetHttp().Post("/api/payment/hongbao/open").Body(map[string]any{
		"native_url":        params.NativeURL,
		"timing_identifier": params.TimingIdentifier,
		"send_username":     params.SendUserName,
	}).DoProto(&resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GrabHongBao 抢红包
func (w web) GrabHongBao(params GrabHongBaoParams) (*HongBaoResponse, error) {
	var resp HongBaoResponse
	if err := api.GetHttp().Post("/api/payment/hongbao/grab").Body(map[string]any{
		"native_url": params.NativeURL,
		"in_way":     params.InWay,
	}).DoProto(&resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// QueryHongBaoDetail 查询红包领取详情
func (w web) QueryHongBaoDetail(params QueryHongBaoDetailParams) (*HongBaoResponse, error) {
	var resp HongBaoResponse
	if err := api.GetHttp().Post("/api/payment/hongbao/detail").Body(map[string]any{
		"native_url":    params.NativeURL,
		"send_username": params.SendUserName,
	}).DoProto(&resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// QueryHongBaoList 查询红包领取列表
func (w web) QueryHongBaoList(params QueryHongBaoListParams) (*HongBaoResponse, error) {
	var resp HongBaoResponse
	if err := api.GetHttp().Post("/api/payment/hongbao/list").Body(map[string]any{
		"native_url":    params.NativeURL,
		"send_username": params.SendUserName,
		"offset":        params.Offset,
		"limit":         params.Limit,
	}).DoProto(&resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// CreatePreTransfer 创建转账
func (w web) CreatePreTransfer(params CreatePreTransferParams) (*TenPayResponse, error) {
	var resp TenPayResponse
	if err := api.GetHttp().Post("/api/payment/transfer/create").Body(map[string]any{
		"to_username": params.ToUserName,
		"fee":         params.Fee,
		"description": params.Description,
	}).DoProto(&resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ConfirmPreTransfer 确认转账
func (w web) ConfirmPreTransfer(params ConfirmPreTransferParams) (*TenPayResponse, error) {
	var resp TenPayResponse
	if err := api.GetHttp().Post("/api/payment/transfer/confirm").Body(map[string]any{
		"bank_type":    params.BankType,
		"bank_serial":  params.BankSerial,
		"req_key":      params.ReqKey,
		"pay_password": params.PayPassword,
	}).DoProto(&resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// CollectMoney 确认收款
func (w web) CollectMoney(params CollectMoneyParams) (*TenPayResponse, error) {
	var resp TenPayResponse
	if err := api.GetHttp().Post("/api/payment/collect").Body(map[string]any{
		"invalid_time":   params.InvalidTime,
		"transfer_id":    params.TransferID,
		"transaction_id": params.TransactionID,
		"to_username":    params.ToUserName,
	}).DoProto(&resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
