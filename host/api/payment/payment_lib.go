//go:build lib

// Package paymentapi 提供支付服务的 lib 实现（直接调用底层实现）。
package paymentapi

import (
	"errors"
	"fmt"
	"reflect"
	"sync"

	pkgpayment "golem/pkg/payment"

	"github.com/sbgayhub/golem/host/api"
	"google.golang.org/protobuf/proto"
)

// lib 支付服务 lib 实现（直接调用底层实现）。
type lib struct{}

// Get 获取 PaymentService 单例（lib 模式）。
var Get = sync.OnceValue(func() PaymentService {
	return &lib{}
})

// GeneratePayQCode 生成支付收款二维码
func (l lib) GeneratePayQCode() (*F2FQrcodeResponse, error) {
	resp, err := pkgpayment.GeneratePayQCode()
	if resp == nil || err != nil {
		return nil, err
	}
	var result F2FQrcodeResponse
	if err := api.TransformProto(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetBandCardList 获取已绑定银行卡及余额
func (l lib) GetBandCardList() (*TenPayResponse, error) {
	resp, err := pkgpayment.GetBandCardList()
	if resp == nil || err != nil {
		return nil, err
	}
	var result TenPayResponse
	if err := api.TransformProto(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateHongBao 创建红包
func (l lib) CreateHongBao(params CreateHongBaoParams) (*HongBaoResponse, error) {
	resp, err := callPayment(pkgpayment.CreateHongBao, func(arg reflect.Value) error {
		return setFields(arg, map[string]any{
			"HBType":   params.HBType,
			"Username": params.Username,
			"InWay":    params.InWay,
			"Count":    params.Count,
			"Amount":   params.Amount,
			"Wishing":  params.Wishing,
		})
	})
	return transformHongBao(resp, err)
}

// ReceiveHongBao 接收红包
func (l lib) ReceiveHongBao(params ReceiveHongBaoParams) (*HongBaoResponse, error) {
	resp, err := callPayment(pkgpayment.ReceiveHongBao, func(arg reflect.Value) error {
		return setFields(arg, map[string]any{
			"NativeURL": params.NativeURL,
			"InWay":     params.InWay,
		})
	})
	return transformHongBao(resp, err)
}

// OpenHongBao 打开红包
func (l lib) OpenHongBao(params OpenHongBaoParams) (*HongBaoResponse, error) {
	resp, err := callPayment(pkgpayment.OpenHongBao, func(arg reflect.Value) error {
		return setFields(arg, map[string]any{
			"NativeURL":        params.NativeURL,
			"TimingIdentifier": params.TimingIdentifier,
			"SendUserName":     params.SendUserName,
		})
	})
	return transformHongBao(resp, err)
}

// GrabHongBao 抢红包
func (l lib) GrabHongBao(params GrabHongBaoParams) (*HongBaoResponse, error) {
	resp, err := callPayment(pkgpayment.GrabHongBao, func(arg reflect.Value) error {
		return setFields(arg, map[string]any{
			"NativeURL": params.NativeURL,
			"InWay":     params.InWay,
		})
	})
	return transformHongBao(resp, err)
}

// QueryHongBaoDetail 查询红包领取详情
func (l lib) QueryHongBaoDetail(params QueryHongBaoDetailParams) (*HongBaoResponse, error) {
	resp, err := callPayment(pkgpayment.QueryHongBaoDetail, func(arg reflect.Value) error {
		return setFields(arg, map[string]any{
			"NativeURL":    params.NativeURL,
			"SendUserName": params.SendUserName,
		})
	})
	return transformHongBao(resp, err)
}

// QueryHongBaoList 查询红包领取列表
func (l lib) QueryHongBaoList(params QueryHongBaoListParams) (*HongBaoResponse, error) {
	resp, err := callPayment(pkgpayment.QueryHongBaoList, func(arg reflect.Value) error {
		return setFields(arg, map[string]any{
			"NativeURL":    params.NativeURL,
			"SendUserName": params.SendUserName,
			"Offset":       params.Offset,
			"Limit":        params.Limit,
		})
	})
	return transformHongBao(resp, err)
}

// CreatePreTransfer 创建转账
func (l lib) CreatePreTransfer(params CreatePreTransferParams) (*TenPayResponse, error) {
	resp, err := callPayment(pkgpayment.CreatePreTransfer, func(arg reflect.Value) error {
		return setFields(arg, map[string]any{
			"ToUserName":  params.ToUserName,
			"Fee":         params.Fee,
			"Description": params.Description,
		})
	})
	return transformTenPay(resp, err)
}

// ConfirmPreTransfer 确认转账
func (l lib) ConfirmPreTransfer(params ConfirmPreTransferParams) (*TenPayResponse, error) {
	resp, err := callPayment(pkgpayment.ConfirmPreTransfer, func(arg reflect.Value) error {
		return setFields(arg, map[string]any{
			"BankType":    params.BankType,
			"BankSerial":  params.BankSerial,
			"ReqKey":      params.ReqKey,
			"PayPassword": params.PayPassword,
		})
	})
	return transformTenPay(resp, err)
}

// CollectMoney 确认收款
func (l lib) CollectMoney(params CollectMoneyParams) (*TenPayResponse, error) {
	resp, err := callPayment(pkgpayment.CollectMoney, func(arg reflect.Value) error {
		return setFields(arg, map[string]any{
			"InvalidTime":   params.InvalidTime,
			"TransferID":    params.TransferID,
			"TransactionID": params.TransactionID,
			"ToUserName":    params.ToUserName,
		})
	})
	return transformTenPay(resp, err)
}

func callPayment(function any, fill func(reflect.Value) error) (proto.Message, error) {
	fn := reflect.ValueOf(function)
	if fn.Kind() != reflect.Func || fn.Type().NumIn() != 1 || fn.Type().NumOut() != 2 {
		return nil, errors.New("invalid payment function signature")
	}
	arg := reflect.New(fn.Type().In(0)).Elem()
	if err := fill(arg); err != nil {
		return nil, err
	}
	out := fn.Call([]reflect.Value{arg})
	if !out[1].IsNil() {
		err, ok := out[1].Interface().(error)
		if !ok {
			return nil, errors.New("payment function returned non-error error value")
		}
		return nil, err
	}
	if out[0].IsNil() {
		return nil, nil
	}
	msg, ok := out[0].Interface().(proto.Message)
	if !ok {
		return nil, errors.New("payment function returned non-proto response")
	}
	return msg, nil
}

func setFields(arg reflect.Value, fields map[string]any) error {
	for name, value := range fields {
		if err := setField(arg, name, value); err != nil {
			return err
		}
	}
	return nil
}

func setField(arg reflect.Value, name string, value any) error {
	field := arg.FieldByName(name)
	if !field.IsValid() {
		return fmt.Errorf("payment params field %s not found", name)
	}
	if !field.CanSet() {
		return fmt.Errorf("payment params field %s cannot be set", name)
	}
	source := reflect.ValueOf(value)
	if source.Type().AssignableTo(field.Type()) {
		field.Set(source)
		return nil
	}
	if source.Type().ConvertibleTo(field.Type()) {
		field.Set(source.Convert(field.Type()))
		return nil
	}
	return fmt.Errorf("payment params field %s cannot use %s as %s", name, source.Type(), field.Type())
}

func transformHongBao(resp proto.Message, err error) (*HongBaoResponse, error) {
	if resp == nil || err != nil {
		return nil, err
	}
	var result HongBaoResponse
	if err := api.TransformProto(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func transformTenPay(resp proto.Message, err error) (*TenPayResponse, error) {
	if resp == nil || err != nil {
		return nil, err
	}
	var result TenPayResponse
	if err := api.TransformProto(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
