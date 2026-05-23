//go:build !lib

// Package chatroomapi 提供群聊服务的 web 实现（通过 HTTP 调用远程服务）。
package chatroomapi

import (
	"fmt"
	"sync"

	"github.com/sbgayhub/golem/host/api"
)

// web 群聊服务 web 实现（通过 HTTP 调用远程服务）
type web struct{}

// Get 获取 ChatroomService 单例（web 模式）
var Get = sync.OnceValue(func() ChatroomService {
	return &web{}
})

// Create 创建群聊 POST /api/chatroom
func (w web) Create(members []string) (*CreateChatroomResponse, error) {
	var resp CreateChatroomResponse
	if err := api.GetHttp().Post("/api/chatroom").Body(map[string]any{"members": members}).DoProto(&resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// FacingCreate 面对面建群 POST /api/chatroom/facing
func (w web) FacingCreate(password string, latitude, longitude float32, operate uint32) (*FacingCreateChatroomResponse, error) {
	var resp FacingCreateChatroomResponse
	if err := api.GetHttp().Post("/api/chatroom/facing").Body(map[string]any{
		"password":  password,
		"latitude":  latitude,
		"longitude": longitude,
		"operate":   operate,
	}).DoProto(&resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetInfo 获取群详细信息 GET /api/chatroom/info/{chatroom}
func (w web) GetInfo(chatroomID string) (*GetChatroomInfoDetailResponse, error) {
	var resp GetChatroomInfoDetailResponse
	if err := api.GetHttp().Get(fmt.Sprintf("/api/chatroom/info/%s", chatroomID)).DoProto(&resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (w web) ListMembers(chatroomID string) (*ListMembersResponse, error) {
	var resp ListMembersResponse
	if err := api.GetHttp().Get("/api/chatroom/members/%s").Path(chatroomID).DoProto(&resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetMemberDetail 获取群成员详情 GET /api/chatroom/members/{chatroom}
func (w web) GetMemberDetail(chatroomID string, members []string) (*GetChatroomMembersResponse, error) {
	var resp GetChatroomMembersResponse
	if err := api.GetHttp().Get("/api/chatroom/members/%s").Path(chatroomID).Body(members).DoProto(&resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetQRCode 获取群二维码 GET /api/chatroom/qrcode/{chatroom}
func (w web) GetQRCode(chatroomID string) (*GetChatroomQRCodeResponse, error) {
	var resp GetChatroomQRCodeResponse
	if err := api.GetHttp().Get(fmt.Sprintf("/api/chatroom/qrcode/%s", chatroomID)).DoProto(&resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// AddMember 添加群成员 POST /api/chatroom/members/{chatroom}
func (w web) AddMember(chatroomID string, members []string) (*AddChatroomMemberResponse, error) {
	var resp AddChatroomMemberResponse
	if err := api.GetHttp().Post(fmt.Sprintf("/api/chatroom/members/%s", chatroomID)).Body(map[string]any{"members": members}).DoProto(&resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// InviteMember 邀请群成员 POST /api/chatroom/invite/{chatroom}
func (w web) InviteMember(chatroomID string, members []string) (*InviteChatroomMemberResponse, error) {
	var resp InviteChatroomMemberResponse
	if err := api.GetHttp().Post(fmt.Sprintf("/api/chatroom/invite/%s", chatroomID)).Body(map[string]any{"members": members}).DoProto(&resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// RemoveMember 移除群成员 DELETE /api/chatroom/members/{chatroom} (with body)
func (w web) RemoveMember(chatroomID string, members []string) (*RemoveChatroomMemberResponse, error) {
	var resp RemoveChatroomMemberResponse
	if err := api.GetHttp().Delete(fmt.Sprintf("/api/chatroom/members/%s", chatroomID)).Body(map[string]any{"members": members}).DoProto(&resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// SetName 设置群名称 PUT /api/chatroom/name/{chatroom}?name=xxx
func (w web) SetName(chatroomID, name string) (*OperateResponse, error) {
	var resp OperateResponse
	if err := api.GetHttp().Put(fmt.Sprintf("/api/chatroom/name/%s", chatroomID)).Query("name", name).DoProto(&resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// SetAnnouncement 设置群公告 PUT /api/chatroom/announcement/{chatroom}
func (w web) SetAnnouncement(chatroomID, content string) (*SetAnnouncementResponse, error) {
	var resp SetAnnouncementResponse
	if err := api.GetHttp().Put(fmt.Sprintf("/api/chatroom/announcement/%s", chatroomID)).Body(map[string]any{"content": content}).DoProto(&resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// SetRemark 设置群备注 PUT /api/chatroom/remark/{chatroom}?remark=xxx
func (w web) SetRemark(chatroomID, remark string) (*OperateResponse, error) {
	var resp OperateResponse
	if err := api.GetHttp().Put(fmt.Sprintf("/api/chatroom/remark/%s", chatroomID)).Query("remark", remark).DoProto(&resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// SetContactList 保存到通讯录 PUT /api/chatroom/contact-list/{chatroom}?save=xxx
func (w web) SetContactList(chatroomID string, save bool) (*OperateResponse, error) {
	var resp OperateResponse
	if err := api.GetHttp().Put(fmt.Sprintf("/api/chatroom/contact-list/%s", chatroomID)).Query("save", save).DoProto(&resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// SetAdmin 设置群管理员 POST /api/chatroom/admins/{chatroom}
func (w web) SetAdmin(chatroomID string, members []string) (*OperateResponse, error) {
	var resp OperateResponse
	if err := api.GetHttp().Post(fmt.Sprintf("/api/chatroom/admins/%s", chatroomID)).Body(map[string]any{"members": members}).DoProto(&resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// RemoveAdmin 移除群管理员 DELETE /api/chatroom/admins/{chatroom} (with body)
func (w web) RemoveAdmin(chatroomID string, members []string) (*OperateResponse, error) {
	var resp OperateResponse
	if err := api.GetHttp().Delete(fmt.Sprintf("/api/chatroom/admins/%s", chatroomID)).Body(map[string]any{"members": members}).DoProto(&resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// TransferOwner 转让群主 POST /api/chatroom/transfer/{chatroom}?new_owner=xxx
func (w web) TransferOwner(chatroomID, newOwner string) (*OperateResponse, error) {
	var resp OperateResponse
	if err := api.GetHttp().Post(fmt.Sprintf("/api/chatroom/transfer/%s", chatroomID)).Query("new_owner", newOwner).DoProto(&resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Quit 退出群聊 DELETE /api/chatroom/quit/{chatroom}
func (w web) Quit(chatroomID string) (*OperateResponse, error) {
	var resp OperateResponse
	if err := api.GetHttp().Delete(fmt.Sprintf("/api/chatroom/quit/%s", chatroomID)).DoProto(&resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ScanJoin 扫码进群 POST /api/chatroom/join/scan
func (w web) ScanJoin(qrcodeURL string) (*JoinResult, error) {
	var resp JoinResult
	if err := api.GetHttp().Post("/api/chatroom/join/scan").Body(map[string]any{"url": qrcodeURL}).DoProto(&resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ScanJoinEnterprise 企业微信扫码进群
func (w web) ScanJoinEnterprise(qrcodeURL string) (*JoinResult, error) {
	return w.ScanJoin(qrcodeURL)
}

// ConsentJoin 同意入群邀请 POST /api/chatroom/join/consent
func (w web) ConsentJoin(inviteURL string) (*JoinResult, error) {
	var resp JoinResult
	if err := api.GetHttp().Post("/api/chatroom/join/consent").Body(map[string]any{"url": inviteURL}).DoProto(&resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
