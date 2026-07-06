package main

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/sbgayhub/golem/sdk/contact"
	"github.com/sbgayhub/golem/sdk/message"
)

// httpGet 发送 HTTP GET 请求
func (p *SetuPlugin) httpGet(urlStr string) (string, error) {
	slog.Debug("[setu] HTTP GET 请求", "url", urlStr)

	resp, err := p.client.Get(urlStr)
	if err != nil {
		slog.Error("[setu] HTTP 请求失败", "url", urlStr, "err", err)
		return "", err
	}
	defer resp.Body.Close()

	slog.Debug("[setu] HTTP 响应", "status", resp.StatusCode, "content_type", resp.Header.Get("Content-Type"))

	if resp.StatusCode != http.StatusOK {
		slog.Error("[setu] HTTP 状态码错误", "status", resp.StatusCode)
		return "", fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("[setu] 读取响应失败", "err", err)
		return "", err
	}

	result := strings.TrimSpace(string(body))
	slog.Debug("[setu] HTTP 响应内容", "length", len(result), "preview", result[:min(100, len(result))])

	return result, nil
}

// downloadMedia 下载媒体资源
func (p *SetuPlugin) downloadMedia(urlStr string) ([]byte, error) {
	resp, err := p.client.Get(urlStr)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// sendText 发送文本消息
func (p *SetuPlugin) sendText(receiver *contact.Contact, text string) {
	msg := &message.Message{
		Type:     message.TypeText,
		Receiver: receiver,
		Content:  text,
		Data:     &message.Message_Text{Text: &message.TextData{Content: text}},
	}
	if _, err := p.message.Send(msg); err != nil {
		slog.Error("[setu] 发送文本失败", "err", err)
	}
}

// sendImage 发送图片消息
func (p *SetuPlugin) sendImage(receiver *contact.Contact, imageURL string) error {
	data, err := p.downloadMedia(imageURL)
	if err != nil {
		return err
	}

	// 使用 CDN 流式上传，不受 gRPC 4MB 限制
	_, err = p.cdn.UploadImage(
		receiver.GetUsername(),
		bytes.NewReader(data),
	)
	if err != nil {
		slog.Error("[setu] CDN 上传图片失败", "err", err)
		// 降级：发送 URL 文本
		p.sendText(receiver, "图片发送失败，直接看链接吧："+imageURL)
		return nil
	}

	slog.Debug("[setu] 发送图片成功", "url", imageURL)
	return nil
}

// sendVideo 发送视频消息（使用 CDN 上传）
func (p *SetuPlugin) sendVideo(receiver *contact.Contact, videoURL string) error {
	// 下载视频到临时文件
	tmpVideo, err := os.CreateTemp("", "setu-video-*.mp4")
	if err != nil {
		return fmt.Errorf("创建临时文件失败: %w", err)
	}
	defer os.Remove(tmpVideo.Name())
	defer tmpVideo.Close()

	// 下载视频数据
	resp, err := p.client.Get(videoURL)
	if err != nil {
		return fmt.Errorf("下载视频失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	if _, err := io.Copy(tmpVideo, resp.Body); err != nil {
		return fmt.Errorf("保存视频失败: %w", err)
	}
	tmpVideo.Close()

	// 使用 ffprobe 获取视频时长
	duration, err := p.getVideoDuration(tmpVideo.Name())
	if err != nil {
		slog.Warn("[setu] 获取视频时长失败，使用默认值", "err", err)
		duration = 10 // 默认 10 秒
	}

	// 使用 ffmpeg 提取缩略图
	thumbData, err := p.extractThumbnail(tmpVideo.Name())
	if err != nil {
		slog.Warn("[setu] 提取缩略图失败，使用空缩略图", "err", err)
		thumbData = []byte{} // 空缩略图
	}

	// 打开视频文件用于上传
	videoFile, err := os.Open(tmpVideo.Name())
	if err != nil {
		return fmt.Errorf("打开视频文件失败: %w", err)
	}
	defer videoFile.Close()

	// 使用 CDN 流式上传视频
	_, err = p.cdn.UploadVideo(
		receiver.GetUsername(),
		thumbData,
		videoFile,
		uint32(duration),
	)
	if err != nil {
		slog.Error("[setu] CDN 上传视频失败", "err", err)
		return err
	}

	slog.Debug("[setu] 发送视频成功", "url", videoURL, "duration", duration, "thumb_size", len(thumbData))
	return nil
}

// getVideoDuration 使用 ffprobe 获取视频时长（秒）
func (p *SetuPlugin) getVideoDuration(videoPath string) (int, error) {
	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		videoPath,
	)

	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("ffprobe 执行失败: %w", err)
	}

	durationStr := strings.TrimSpace(string(output))
	duration, err := strconv.ParseFloat(durationStr, 64)
	if err != nil {
		return 0, fmt.Errorf("解析时长失败: %w", err)
	}

	return int(duration), nil
}

// extractThumbnail 使用 ffmpeg 提取视频缩略图
func (p *SetuPlugin) extractThumbnail(videoPath string) ([]byte, error) {
	// 创建临时缩略图文件
	tmpThumb, err := os.CreateTemp("", "setu-thumb-*.jpg")
	if err != nil {
		return nil, fmt.Errorf("创建缩略图临时文件失败: %w", err)
	}
	defer os.Remove(tmpThumb.Name())
	tmpThumb.Close()

	// 使用 ffmpeg 提取第 1 秒的帧作为缩略图
	cmd := exec.Command("ffmpeg",
		"-i", videoPath,
		"-ss", "00:00:01",
		"-vframes", "1",
		"-f", "image2",
		"-y",
		tmpThumb.Name(),
	)

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg 执行失败: %w", err)
	}

	// 读取缩略图数据
	thumbData, err := os.ReadFile(tmpThumb.Name())
	if err != nil {
		return nil, fmt.Errorf("读取缩略图失败: %w", err)
	}

	return thumbData, nil
}
