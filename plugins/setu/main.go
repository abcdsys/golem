package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/sbgayhub/golem/sdk/cdn"
	"github.com/sbgayhub/golem/sdk/contact"
	"github.com/sbgayhub/golem/sdk/message"
	"github.com/sbgayhub/golem/sdk/plugin"
)

// Config 插件配置
type Config struct {
	ImgURL        string `toml:"img_url" comment:"美女图片API"`
	BoyURL        string `toml:"boy_url" comment:"帅哥图片API"`
	HeisiURL      string `toml:"heisi_url" comment:"黑丝图片API"`
	BaisiURL      string `toml:"baisi_url" comment:"白丝图片API"`
	HeisiVideoURL string `toml:"heisi_video_url" comment:"黑丝视频API"`
	BaisiVideoURL string `toml:"baisi_video_url" comment:"白丝视频API"`
	// 搜图方式多种，可以百度图片网页，这里用的是 https://www.apihz.cn/api/apihzbqbbaidu.html 提供的
	SearchURL     string `toml:"search_url" comment:"搜索图片API（默认使用 apihz.cn 的百度表情搜图）"`
	VideoRate     int    `toml:"video_rate" comment:"视频触发概率(0-100)"`
}

// SetuPlugin 色图插件
type SetuPlugin struct {
	plugin.ConfigAbility[Config]
	message message.Ability
	contact contact.Ability
	cdn     cdn.Ability
	client  *http.Client
}

// newHTTPClient 创建带自定义重定向处理的 HTTP 客户端
func newHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("过多重定向")
			}
			// 处理 Location 头中可能存在的引号
			loc := req.Response.Header.Get("Location")
			if loc != "" {
				// 去除引号
				loc = strings.Trim(loc, "'\"")
				if parsed, err := url.Parse(loc); err == nil {
					req.URL = req.URL.ResolveReference(parsed)
				}
			}
			return nil
		},
	}
}

// GetMetadata 返回插件元数据
func (p *SetuPlugin) GetMetadata() *plugin.Metadata {
	return &plugin.Metadata{
		Name:        "setu",
		Author:      "Golem Team",
		Version:     "1.0.0",
		Description: "色图插件 - 提供各种图片和视频",
		Priority:    -100,
	}
}

func (p *SetuPlugin) OnLoad() error {
	p.ensureDefaults()
	slog.Info("[setu] 色图插件加载成功",
		"img_url", p.Config.ImgURL,
		"video_rate", p.Config.VideoRate,
	)
	return nil
}

func (p *SetuPlugin) OnUnload() error {
	slog.Info("[setu] 色图插件已卸载")
	return nil
}

// GetSubscriptions 订阅事件
func (p *SetuPlugin) GetSubscriptions() []string {
	return []string{
		message.TypeText.Topic,
	}
}

// OnEvent 处理事件
func (p *SetuPlugin) OnEvent(e *plugin.Event) (bool, error) {
	p.ensureDefaults()

	msg := e.Payload.(*plugin.Event_Message).Message
	if msg == nil {
		return false, nil
	}

	text := strings.TrimSpace(msg.GetContent())
	if text == "" {
		return false, nil
	}

	// 获取接收者
	receiver := p.contact.Get(e.GetSender())
	if receiver == nil {
		slog.Warn("[setu] 未找到接收者", "sender", e.GetSender())
		return false, nil
	}

	// 匹配关键词
	switch text {
	case "plmm", "漂亮妹妹", "来点美女":
		return p.handlePlmm(receiver)
	case "来点黑丝":
		return p.handleHeisi(receiver)
	case "来点白丝":
		return p.handleBaisi(receiver)
	case "看看腿":
		return p.handleKkt(receiver)
	case "来点帅哥":
		return p.handleBoy(receiver)
	}

	// 前缀匹配：来点XX（搜索）
	if strings.HasPrefix(text, "来点") && len([]rune(text)) > 2 {
		keyword := string([]rune(text)[2:])
		return p.handleSearch(receiver, keyword)
	}

	return false, nil
}

// ensureDefaults 确保配置不为空
func (p *SetuPlugin) ensureDefaults() {
	if p.Config.ImgURL == "" {
		p.Config.ImgURL = "https://api.52vmy.cn/api/img/tu/girl?type=text"
	}
	if p.Config.BoyURL == "" {
		p.Config.BoyURL = "https://api.52vmy.cn/api/img/tu/boy?type=text"
	}
	if p.Config.HeisiURL == "" {
		p.Config.HeisiURL = "http://api.yujn.cn/api/heisi.php?"
	}
	if p.Config.BaisiURL == "" {
		p.Config.BaisiURL = "http://api.yujn.cn/api/baisi.php?"
	}
	if p.Config.HeisiVideoURL == "" {
		p.Config.HeisiVideoURL = "http://api.yujn.cn/api/heisis.php?type=video"
	}
	if p.Config.BaisiVideoURL == "" {
		p.Config.BaisiVideoURL = "http://api.yujn.cn/api/baisis.php?type=video"
	}
	if p.Config.SearchURL == "" {
		// 搜图方式多种，可以百度图片网页，这里用的是 https://www.apihz.cn/api/apihzbqbbaidu.html 提供的
		p.Config.SearchURL = "https://cn.apihz.cn/api/img/apihzbqbbaidu.php?id=88888888&key=88888888&limit=10&page=1&words="
	}
	if p.Config.VideoRate == 0 {
		p.Config.VideoRate = 50
	}
}

// handlePlmm 处理漂亮妹妹
func (p *SetuPlugin) handlePlmm(receiver *contact.Contact) (bool, error) {
	slog.Info("[setu] 处理漂亮妹妹请求", "api", p.Config.ImgURL)
	
	imgURL, err := p.httpGet(p.Config.ImgURL)
	if err != nil {
		slog.Error("[setu] 获取漂亮妹妹图片失败", "err", err)
		p.sendText(receiver, "获取漂亮妹妹图片失败: "+err.Error())
		return true, nil
	}
	
	if imgURL == "" {
		slog.Warn("[setu] 漂亮妹妹 API 返回空内容")
		p.sendText(receiver, "获取漂亮妹妹图片失败: API 返回空内容")
		return true, nil
	}

	slog.Info("[setu] 获取到图片 URL", "url", imgURL)

	if err := p.sendImage(receiver, imgURL); err != nil {
		slog.Error("[setu] 发送漂亮妹妹图片失败", "err", err)
		p.sendText(receiver, "发送图片失败: "+err.Error())
		return true, nil
	}

	slog.Info("[setu] 发送漂亮妹妹图片成功")
	return true, nil
}

// handleHeisi 处理来点黑丝（30%视频，70%图片）
func (p *SetuPlugin) handleHeisi(receiver *contact.Contact) (bool, error) {
	if rand.Intn(100) < p.Config.VideoRate {
		// 尝试发送视频
		err := p.sendVideo(receiver, p.Config.HeisiVideoURL)
		if err == nil {
			slog.Info("[setu] 发送黑丝视频（概率触发）")
			return true, nil
		}
		// 视频失败，降级发送图片
		slog.Warn("[setu] 黑丝视频失败，降级发送图片", "err", err)
	}
	
	if err := p.sendImage(receiver, p.Config.HeisiURL); err != nil {
		slog.Error("[setu] 发送黑丝图片失败", "err", err)
		p.sendText(receiver, "发送图片失败")
		return true, nil
	}
	slog.Info("[setu] 发送黑丝图片")
	return true, nil
}

// handleBaisi 处理来点白丝（30%视频，70%图片）
func (p *SetuPlugin) handleBaisi(receiver *contact.Contact) (bool, error) {
	if rand.Intn(100) < p.Config.VideoRate {
		// 尝试发送视频
		err := p.sendVideo(receiver, p.Config.BaisiVideoURL)
		if err == nil {
			slog.Info("[setu] 发送白丝视频（概率触发）")
			return true, nil
		}
		// 视频失败，降级发送图片
		slog.Warn("[setu] 白丝视频失败，降级发送图片", "err", err)
	}
	
	if err := p.sendImage(receiver, p.Config.BaisiURL); err != nil {
		slog.Error("[setu] 发送白丝图片失败", "err", err)
		p.sendText(receiver, "发送图片失败")
		return true, nil
	}
	slog.Info("[setu] 发送白丝图片")
	return true, nil
}

// handleKkt 处理看看腿（50%黑丝，50%白丝）
func (p *SetuPlugin) handleKkt(receiver *contact.Contact) (bool, error) {
	if time.Now().Unix()%2 == 0 {
		return p.handleHeisi(receiver)
	}
	return p.handleBaisi(receiver)
}

// handleBoy 处理来点帅哥
func (p *SetuPlugin) handleBoy(receiver *contact.Contact) (bool, error) {
	imgURL, err := p.httpGet(p.Config.BoyURL)
	if err != nil || imgURL == "" {
		p.sendText(receiver, "获取帅哥图片失败")
		return true, nil
	}

	if err := p.sendImage(receiver, imgURL); err != nil {
		slog.Error("[setu] 发送帅哥图片失败", "err", err)
		p.sendText(receiver, "发送图片失败")
		return true, nil
	}

	slog.Info("[setu] 发送帅哥图片")
	return true, nil
}

// handleSearch 处理来点XX（搜索图片）
func (p *SetuPlugin) handleSearch(receiver *contact.Contact, keyword string) (bool, error) {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return false, nil
	}

	slog.Info("[setu] 搜索图片关键词", "keyword", keyword)

	// URL 编码
	encodedKeyword := url.QueryEscape(keyword)
	searchURL := p.Config.SearchURL + encodedKeyword

	// 调用搜索 API
	response, err := p.httpGet(searchURL)
	if err != nil || response == "" {
		p.sendText(receiver, "搜索失败，请稍后再试")
		return true, nil
	}

	// 解析 JSON 响应
	var result struct {
		Code int      `json:"code"`
		Msg  string   `json:"msg"`
		Res  []string `json:"res"`
	}
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		p.sendText(receiver, "解析响应失败")
		return true, nil
	}

	if result.Code != 200 {
		msg := result.Msg
		if msg == "" {
			msg = "搜索失败"
		}
		p.sendText(receiver, msg)
		return true, nil
	}

	if len(result.Res) == 0 {
		p.sendText(receiver, "没有找到相关图片")
		return true, nil
	}

	// 随机选择一张图片
	index := rand.Intn(len(result.Res))
	imageURL := result.Res[index]

	if err := p.sendImage(receiver, imageURL); err != nil {
		slog.Error("[setu] 发送搜索图片失败", "err", err)
		p.sendText(receiver, "发送图片失败")
		return true, nil
	}

	slog.Info("[setu] 发送搜索图片成功", "keyword", keyword)
	return true, nil
}

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

	slog.Info("[setu] 发送图片成功", "url", imageURL)
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

	slog.Info("[setu] 发送视频成功", "url", videoURL, "duration", duration, "thumb_size", len(thumbData))
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

func main() {
	p := &SetuPlugin{
		ConfigAbility: plugin.ConfigAbility[Config]{
			Config: Config{
				ImgURL:        "https://api.52vmy.cn/api/img/tu/girl?type=text",
				BoyURL:        "https://api.52vmy.cn/api/img/tu/boy?type=text",
				HeisiURL:      "http://api.yujn.cn/api/heisi.php?",
				BaisiURL:      "http://api.yujn.cn/api/baisi.php?",
				HeisiVideoURL: "http://api.yujn.cn/api/heisis.php?type=video",
				BaisiVideoURL: "http://api.yujn.cn/api/baisis.php?type=video",
				// 搜图方式多种，可以百度图片网页，这里用的是 https://www.apihz.cn/api/apihzbqbbaidu.html 提供的
				SearchURL:     "https://cn.apihz.cn/api/img/apihzbqbbaidu.php?id=88888888&key=88888888&limit=10&page=1&words=",
				VideoRate:     50,
			},
		},
		client: newHTTPClient(),
	}
	slog.Info("[setu] 色图插件启动中...")
	plugin.Start(p)
}
