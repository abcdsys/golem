//go:build !lib

// Package cdnapi 提供 CDN 文件上传/下载服务的 web 实现（通过 HTTP 调用远程服务）。
package cdnapi

import (
	"bytes"
	"fmt"
	"io"
	"sync"

	"github.com/sbgayhub/golem/host/api"
)

// web CDN 服务 web 实现（通过 HTTP 调用远程服务）
type web struct{}

// Get 获取 CDNService 单例（web 模式）
var Get = sync.OnceValue(func() CDNService {
	return &web{}
})

// UploadImage CDN 上传聊天图片
func (w web) UploadImage(receiver string, reader io.Reader) (*UploadImageResult, error) {
	imageData, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	var resp UploadImageResult
	if err := api.GetHttp().Post("/api/cdn/upload/image").Multipart(
		map[string][]byte{"image": imageData},
		map[string]string{"receiver": receiver},
	).DoProto(&resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// UploadMomentsImage CDN 上传朋友圈图片
func (w web) UploadMomentsImage(imageData []byte) (*UploadSnsImageResult, error) {
	var resp UploadSnsImageResult
	if err := api.GetHttp().Post("/api/cdn/upload/moments/image").Multipart(
		map[string][]byte{"image": imageData},
		nil,
	).DoProto(&resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// UploadVideo CDN 上传聊天视频
func (w web) UploadVideo(receiver string, thumb []byte, reader io.Reader, duration uint32) (*UploadVideoResult, error) {
	videoData, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	var resp UploadVideoResult
	if err := api.GetHttp().Post("/api/cdn/upload/video").Multipart(
		map[string][]byte{"video": videoData, "thumb": thumb},
		map[string]string{"receiver": receiver, "duration": fmt.Sprintf("%d", duration)},
	).DoProto(&resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// UploadMomentsVideo CDN 上传朋友圈视频
func (w web) UploadMomentsVideo(videoData, thumbData []byte) (*UploadSnsVideoResult, error) {
	var resp UploadSnsVideoResult
	if err := api.GetHttp().Post("/api/cdn/upload/moments/video").Multipart(
		map[string][]byte{"video": videoData, "thumb": thumbData},
		nil,
	).DoProto(&resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// DownloadImage CDN 下载高清图片（返回 ReadCloser 流）
func (w web) DownloadImage(fileID, fileAesKey string) (io.ReadCloser, error) {
	raw, err := api.GetHttp().Get("/api/cdn/download/image").Query("id", fileID, "key", fileAesKey).Do()
	if err != nil {
		return nil, err
	}
	return io.NopCloser(bytes.NewReader(raw)), nil
}

// DownloadVideoCover CDN 下载视频封面
func (w web) DownloadVideoCover(fileID, fileAesKey string) ([]byte, error) {
	return api.GetHttp().Get("/api/cdn/download/video/cover").Query("id", fileID, "key", fileAesKey).Do()
}

// DownloadVideo CDN 下载聊天视频（返回 ReadCloser 流）
func (w web) DownloadVideo(fileID, fileAesKey string) (io.ReadCloser, error) {
	raw, err := api.GetHttp().Get("/api/cdn/download/video").Query("id", fileID, "key", fileAesKey).Do()
	if err != nil {
		return nil, err
	}
	return io.NopCloser(bytes.NewReader(raw)), nil
}

// DownloadSnsVideo CDN 下载朋友圈视频
func (w web) DownloadSnsVideo(videoURL string, encKey uint64) ([]byte, error) {
	return api.GetHttp().Get("/api/cdn/download/moments/video").Query("url", videoURL, "key", encKey).Do()
}
