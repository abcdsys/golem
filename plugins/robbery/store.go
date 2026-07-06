package main

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/sbgayhub/golem/sdk/message"
)

// loadData 加载持久化数据
func (p *RobberyPlugin) loadData() {
	data, err := os.ReadFile(p.Config.DataFile)
	if err != nil {
		if !os.IsNotExist(err) {
			slog.Warn("[robbery] 读取数据文件失败，将使用空数据", "err", err)
		} else {
			slog.Info("[robbery] 未找到历史数据文件，从头开始")
		}
		return
	}

	if len(data) == 0 {
		return
	}

	var loaded map[string]map[string]*PlayerData
	if err := json.Unmarshal(data, &loaded); err != nil {
		slog.Warn("[robbery] 解析数据文件失败，将使用空数据", "err", err)
		return
	}

	p.data = loaded
	slog.Info("[robbery] 已加载历史数据", "groups", len(loaded))
}

// saveData 保存数据到文件
func (p *RobberyPlugin) saveData() error {
	p.mu.RLock()
	// 深拷贝一份用于序列化，避免序列化期间数据被修改
	data := make(map[string]map[string]*PlayerData, len(p.data))
	for k, v := range p.data {
		players := make(map[string]*PlayerData, len(v))
		for uid, pd := range v {
			players[uid] = pd
		}
		data[k] = players
	}
	p.mu.RUnlock()

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	// 确保目录存在
	dir := filepath.Dir(p.Config.DataFile)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	return os.WriteFile(p.Config.DataFile, jsonData, 0644)
}

// getOrCreatePlayer 获取或创建玩家数据
func (p *RobberyPlugin) getOrCreatePlayer(chatroomID, userID string, msg *message.Message) *PlayerData {
	p.mu.Lock()
	defer p.mu.Unlock()

	playersInGroup, ok := p.data[chatroomID]
	if !ok {
		playersInGroup = make(map[string]*PlayerData)
		p.data[chatroomID] = playersInGroup
	}

	if pd, exists := playersInGroup[userID]; exists {
		return pd
	}

	displayName := p.getDisplayName(chatroomID, userID, msg)
	pd := &PlayerData{
		UserID:      userID,
		DisplayName: displayName,
		Money:       p.Config.InitialMoney,
		Strength:    p.Config.InitialStrength,
	}
	playersInGroup[userID] = pd
	return pd
}

// initializeGroupMembers 初始化群聊成员（如果该群还没有玩家数据）
func (p *RobberyPlugin) initializeGroupMembers(chatroomID string) {
	p.mu.Lock()
	playersInGroup, ok := p.data[chatroomID]
	hasPlayers := ok && len(playersInGroup) > 0
	p.mu.Unlock()

	if hasPlayers {
		return
	}

	if p.chatroom == nil {
		slog.Warn("[robbery] chatroom 能力未注入，跳过群成员初始化", "chatroom", chatroomID)
		return
	}

	members := p.chatroom.ListMembers(chatroomID)
	if len(members) == 0 {
		slog.Warn("[robbery] 无法获取群成员列表", "chatroom", chatroomID)
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if playersInGroup == nil {
		playersInGroup = make(map[string]*PlayerData)
		p.data[chatroomID] = playersInGroup
	}

	initializedCount := 0
	for _, member := range members {
		uid := member.GetUsername()
		if uid == "" {
			continue
		}
		if _, exists := playersInGroup[uid]; exists {
			continue
		}

		displayName := member.GetDisplayName()
		if displayName == "" {
			displayName = member.GetNickname()
		}
		if displayName == "" {
			displayName = uid
		}

		playersInGroup[uid] = &PlayerData{
			UserID:      uid,
			DisplayName: displayName,
			Money:       p.Config.InitialMoney,
			Strength:    p.Config.InitialStrength,
		}
		initializedCount++
	}

	slog.Info("[robbery] 群聊成员初始化完成",
		"chatroom", chatroomID, "initialized", initializedCount)

	// 保存由调用方在后续命令处理中触发，这里不主动保存以避免锁重入
}

// getDisplayName 获取用户在群聊中的显示名称
func (p *RobberyPlugin) getDisplayName(chatroomID, userID string, msg *message.Message) string {
	// 优先从消息中的 Member 字段获取（群消息中 SDK 注入）
	if msg != nil && msg.Member != nil && msg.Member.GetUsername() == userID {
		if name := msg.Member.GetDisplayName(); name != "" {
			return name
		}
		if name := msg.Member.GetNickname(); name != "" {
			return name
		}
	}

	// 通过 chatroom 能力查询
	if p.chatroom != nil {
		if member := p.chatroom.GetMember(chatroomID, userID); member != nil {
			if name := member.GetDisplayName(); name != "" {
				return name
			}
			if name := member.GetNickname(); name != "" {
				return name
			}
		}
	}

	// 通过 contact 能力查询
	if p.contact != nil {
		if c := p.contact.Get(userID); c != nil {
			if name := c.GetNickname(); name != "" {
				return name
			}
		}
	}

	return userID
}

// getEligibleTargets 获取可被打劫的目标（排除自己、被监禁、金钱<=0 的玩家）
func (p *RobberyPlugin) getEligibleTargets(chatroomID, excludeUserID string) []*PlayerData {
	p.mu.RLock()
	defer p.mu.RUnlock()

	playersInGroup, ok := p.data[chatroomID]
	if !ok {
		return nil
	}

	var eligible []*PlayerData
	for _, pd := range playersInGroup {
		if pd.UserID == excludeUserID {
			continue
		}
		if pd.InJail {
			continue
		}
		if pd.Money <= 0 {
			continue
		}
		eligible = append(eligible, pd)
	}
	return eligible
}
