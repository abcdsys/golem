package main

import (
	"log/slog"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/sbgayhub/golem/sdk/cdn"
	"github.com/sbgayhub/golem/sdk/chatroom"
	"github.com/sbgayhub/golem/sdk/contact"
	"github.com/sbgayhub/golem/sdk/message"
	"github.com/sbgayhub/golem/sdk/plugin"
)

// FarmPlugin 农场游戏插件
type FarmPlugin struct {
	plugin.ConfigAbility[Config]
	message  message.Ability
	contact  contact.Ability
	chatroom chatroom.Ability
	cdn      cdn.Ability

	mu        sync.RWMutex
	groupData map[string]map[string]*FarmPlayer
	random    *rand.Rand
}

// GetMetadata 返回插件元数据
func (p *FarmPlugin) GetMetadata() *plugin.Metadata {
	return &plugin.Metadata{
		Name:        "farm",
		Author:      "Golem Team",
		Version:     "1.0.0",
		Description: "农场小游戏插件",
		Priority:    0,
	}
}

// OnLoad 插件加载
func (p *FarmPlugin) OnLoad() error {
	p.resolvePaths()
	p.groupData = make(map[string]map[string]*FarmPlayer)
	p.random = rand.New(rand.NewSource(time.Now().UnixNano()))
	p.loadData()
	slog.Info("[farm] 农场插件加载成功", "data_file", p.Config.DataFile, "image_dir", p.Config.ImageDir)
	return nil
}

// OnUnload 插件卸载
func (p *FarmPlugin) OnUnload() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	_ = p.saveDataLocked()
	slog.Info("[farm] 农场插件已卸载")
	return nil
}

// OnEnable 插件启用
func (p *FarmPlugin) OnEnable() error {
	return nil
}

// OnDisable 插件禁用
func (p *FarmPlugin) OnDisable() error {
	return nil
}

// GetSubscriptions 订阅文本消息
func (p *FarmPlugin) GetSubscriptions() []string {
	return []string{message.TypeText.Topic}
}

// resolvePaths 将相对路径解析为基于插件可执行文件目录的绝对路径
func (p *FarmPlugin) resolvePaths() {
	exe, err := os.Executable()
	if err != nil {
		slog.Warn("[farm] 无法获取可执行文件路径，使用相对路径", "err", err)
		return
	}
	exeDir := filepath.Dir(exe)

	if !filepath.IsAbs(p.Config.DataFile) {
		p.Config.DataFile = filepath.Join(exeDir, p.Config.DataFile)
	}
	if !filepath.IsAbs(p.Config.ImageDir) {
		p.Config.ImageDir = filepath.Join(exeDir, p.Config.ImageDir)
	}
}
