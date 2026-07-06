package main

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
)

func (p *FarmPlugin) loadData() {
	p.mu.Lock()
	defer p.mu.Unlock()

	path := p.Config.DataFile
	if _, err := os.Stat(path); os.IsNotExist(err) {
		slog.Info("[farm] 无历史数据，开始新游戏")
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		slog.Warn("[farm] 读取数据失败", "err", err)
		return
	}
	if len(data) == 0 {
		return
	}
	var loaded map[string]map[string]*FarmPlayer
	if err := json.Unmarshal(data, &loaded); err != nil {
		slog.Warn("[farm] 解析数据失败", "err", err)
		return
	}
	if loaded != nil {
		p.groupData = loaded
	}
	slog.Info("[farm] 加载数据成功", "groups", len(p.groupData))
}

func (p *FarmPlugin) saveData() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.saveDataLocked()
}

func (p *FarmPlugin) saveDataLocked() error {
	path := p.Config.DataFile
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	data, err := json.MarshalIndent(p.groupData, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func (p *FarmPlugin) getOrCreatePlayer(chatroomID, userID string) *FarmPlayer {
	p.mu.Lock()
	defer p.mu.Unlock()

	group, ok := p.groupData[chatroomID]
	if !ok {
		group = make(map[string]*FarmPlayer)
		p.groupData[chatroomID] = group
	}
	player, ok := group[userID]
	if !ok {
		player = &FarmPlayer{
			UserID:      userID,
			DisplayName: p.getDisplayName(chatroomID, userID),
			Coins:       p.Config.InitialCoins,
			Fields:      p.Config.InitialFields,
			Ponds:       1,
			CropCount:   make(map[string]int),
			LandFields:  make(map[string]*Field),
			Pets:        []int{},
		}
		group[userID] = player
		_ = p.saveDataLocked()
	}
	if player.CropCount == nil {
		player.CropCount = make(map[string]int)
	}
	if player.LandFields == nil {
		player.LandFields = make(map[string]*Field)
	}
	if player.Pets == nil {
		player.Pets = []int{}
	}
	return player
}
