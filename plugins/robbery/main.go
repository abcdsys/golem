package main

import (
	"log/slog"
	"math/rand"
	"sync"
	"time"

	"github.com/sbgayhub/golem/sdk/chatroom"
	"github.com/sbgayhub/golem/sdk/contact"
	"github.com/sbgayhub/golem/sdk/message"
	"github.com/sbgayhub/golem/sdk/plugin"
)

// Config 插件配置结构
type Config struct {
	DataFile        string `toml:"data_file" comment:"玩家数据存储文件路径"`
	InitialMoney    int    `toml:"initial_money" comment:"新玩家初始金钱"`
	InitialStrength int    `toml:"initial_strength" comment:"新玩家初始武力"`
	JobRefreshHours int    `toml:"job_refresh_hours" comment:"任务刷新间隔（小时）"`
	WelfareAmount   int    `toml:"welfare_amount" comment:"破产救济金额"`
	WelfareCooldown int    `toml:"welfare_cooldown" comment:"救济冷却时间（小时）"`
	ChangeProfCost  int    `toml:"change_prof_cost" comment:"转职所需金币"`
}

// RobberyPlugin 打劫游戏插件
type RobberyPlugin struct {
	plugin.ConfigAbility[Config]

	message  message.Ability  // 消息发送能力
	contact  contact.Ability  // 联系人查询能力
	chatroom chatroom.Ability // 群聊能力（查询群成员）

	mu   sync.RWMutex
	data map[string]map[string]*PlayerData // 群ID -> 用户ID -> 玩家数据
	rand *rand.Rand
}

// GetMetadata 返回插件元数据
func (p *RobberyPlugin) GetMetadata() *plugin.Metadata {
	return &plugin.Metadata{
		Name:        "robbery",
		Author:      "Golem Team",
		Version:     "1.0.0",
		Description: "打劫游戏插件 - 群成员间互相打劫、职业装备、赌博等",
		Priority:    0,
	}
}

// 游戏常量
const (
	baseWantedChance   = 0.2 // 基础通缉概率
	crimeIncrement     = 0.1 // 每次犯罪增加的通缉概率
	eventTriggerChance = 0.3 // 随机事件触发概率
)

func main() {
	p := &RobberyPlugin{
		ConfigAbility: plugin.ConfigAbility[Config]{
			Config: Config{
				DataFile:        "data/robbery_game.json",
				InitialMoney:    100,
				InitialStrength: 15,
				JobRefreshHours: 24,
				WelfareAmount:   50,
				WelfareCooldown: 24,
				ChangeProfCost:  500,
			},
		},
		rand: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	slog.Info("[robbery] 打劫游戏插件启动中...")
	plugin.Start(p)
}
