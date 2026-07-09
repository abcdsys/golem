package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const (
	defaultVideoDurationSeconds = 10
	defaultThumbWidth           = 100
	defaultThumbHeight          = 100
	probeTimeout                = 10 * time.Second
)

// extractVideoMeta 从视频字节中提取时长（秒）与首帧缩略图。
// 依赖 ffmpeg / ffprobe；缺失或失败时降级为默认黑图 + 10 秒时长，永不返回错误。
// 唯一例外：默认黑图编码失败（标准库 jpeg encoder 在 image.RGBA 上不会失败）。
func extractVideoMeta(data []byte) (duration uint32, thumb []byte, err error) {
	thumb, err = defaultBlackThumb()
	if err != nil {
		return 0, nil, fmt.Errorf("生成默认黑图失败: %w", err)
	}
	duration = defaultVideoDurationSeconds

	ffprobePath, errProbe := exec.LookPath("ffprobe")
	ffmpegPath, errFF := exec.LookPath("ffmpeg")
	if errProbe != nil || errFF != nil {
		slog.Error("ffmpeg 未找到")
		return duration, thumb, nil
	}

	tmpPath, cleanup, writeErr := writeTempVideo(data)
	if writeErr != nil {
		return duration, thumb, nil
	}
	defer cleanup()

	if d, ok := probeVideoDuration(ffprobePath, tmpPath); ok && d > 0 {
		duration = d
	}
	if t, ok := extractVideoThumb(ffmpegPath, tmpPath); ok && len(t) > 0 {
		thumb = t
	}
	slog.Debug("ffmpeg 获取视频信息成功", "thumb_len", len(thumb), "duration", duration)
	return duration, thumb, nil
}

// defaultBlackThumb 用标准库生成一张全黑 JPEG（image.RGBA 零值即纯黑）。
func defaultBlackThumb() ([]byte, error) {
	img := image.NewRGBA(image.Rect(0, 0, defaultThumbWidth, defaultThumbHeight))
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// writeTempVideo 把视频字节写入临时文件，返回路径与 cleanup。
// 用临时文件而非 stdin pipe：mp4 等格式的 moov atom 可能位于文件末尾，pipe 无法 seek。
func writeTempVideo(data []byte) (path string, cleanup func(), err error) {
	f, err := os.CreateTemp("", "universal-video-*.bin")
	if err != nil {
		return "", nil, err
	}
	cleanup = func() { _ = os.Remove(f.Name()) }
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		cleanup()
		return "", nil, err
	}
	_ = f.Close()
	return f.Name(), cleanup, nil
}

// probeVideoDuration 调用 ffprobe 读取视频时长（秒），失败返回 false。
func probeVideoDuration(ffprobe, path string) (uint32, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), probeTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, ffprobe,
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		path,
	).Output()
	if err != nil {
		slog.Error("ff 获取视频时长失败", "err", err)
		return 0, false
	}
	s := strings.TrimSpace(string(out))
	if s == "" || s == "N/A" {
		return 0, false
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil || f <= 0 {
		return 0, false
	}
	return uint32(f), true
}

// extractVideoThumb 调用 ffmpeg 抽取视频首帧为 MJPEG 字节，失败返回 false。
func extractVideoThumb(ffmpeg, path string) ([]byte, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), probeTimeout)
	defer cancel()
	var stdout bytes.Buffer
	cmd := exec.CommandContext(ctx, ffmpeg,
		"-i", path,
		"-frames:v", "1",
		"-f", "mjpeg",
		"pipe:1",
	)
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return nil, false
	}
	if stdout.Len() == 0 {
		return nil, false
	}
	return stdout.Bytes(), true
}
