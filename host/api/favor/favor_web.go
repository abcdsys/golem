//go:build !lib

// Package favorapi 提供收藏服务的 web 实现（通过 HTTP 调用远程服务）。
package favorapi

import (
	"encoding/base64"
	"fmt"
	"sync"

	"github.com/sbgayhub/golem/host/api"
)

// web 收藏服务 web 实现（通过 HTTP 调用远程服务）
type web struct{}

// Get 获取 FavorService 单例（web 模式）
var Get = sync.OnceValue(func() FavorService {
	return &web{}
})

// GetInfo 获取收藏容量信息
func (w web) GetInfo() (*GetInfoResponse, error) {
	var resp GetInfoResponse
	if err := api.GetHttp().Get("/api/favor/info").DoProto(&resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetItem 获取收藏项详情
func (w web) GetItem(favId int32) (*GetFavItemResponse, error) {
	var resp GetFavItemResponse
	if err := api.GetHttp().Get(fmt.Sprintf("/api/favor/item/%d", favId)).DoProto(&resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// BatchGetItems 批量获取收藏项（web 模式逐个查询聚合）
func (w web) BatchGetItems(favIds []int32) (*BatchGetFavItemsResponse, error) {
	items := make([]*FavorItem, 0, len(favIds))
	for _, favId := range favIds {
		resp, err := w.GetItem(favId)
		if err != nil {
			return nil, err
		}
		items = append(items, resp.Items...)
	}
	return &BatchGetFavItemsResponse{Objects: items}, nil
}

// Delete 删除收藏项
func (w web) Delete(favId int32) (*DeleteFavItemResponse, error) {
	var resp DeleteFavItemResponse
	if err := api.GetHttp().Delete(fmt.Sprintf("/api/favor/item/%d", favId)).DoProto(&resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// BatchDelete 批量删除收藏项（web 模式逐个删除聚合）
func (w web) BatchDelete(favIds []int32) (*BatchDeleteFavItemsResponse, error) {
	results := make([]*DeleteResult, 0, len(favIds))
	for _, favId := range favIds {
		resp, err := w.Delete(favId)
		if err != nil {
			return nil, err
		}
		results = append(results, resp.Results...)
	}
	count := uint32(len(results))
	return &BatchDeleteFavItemsResponse{Count: &count, Results: results}, nil
}

// Sync 同步收藏列表
func (w web) Sync(key []byte) (*SyncFavorResponse, error) {
	req := api.GetHttp().Post("/api/favor/sync")
	if len(key) > 0 {
		req = req.Query("key", base64.StdEncoding.EncodeToString(key))
	}
	var resp SyncFavorResponse
	if err := req.DoProto(&resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
