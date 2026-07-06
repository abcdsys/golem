package main

// Profession 职业定义
type Profession struct {
	Name          string  // 职业名称
	BonusStrength int     // 武力加成
	BonusMoney    int     // 金钱加成
	SkillBonus    float64 // 技能加成（成功率等）
	Description   string  // 技能描述
}

// professions 职业静态表
var professions = map[string]Profession{
	"盗贼": {Name: "盗贼", BonusStrength: 20, BonusMoney: 10, SkillBonus: 0.15, Description: "背水一战：打劫成功率 +15%"},
	"战士": {Name: "战士", BonusStrength: 30, BonusMoney: 5, SkillBonus: 0.0, Description: "铁壁：初始武力 +30"},
	"商人": {Name: "商人", BonusStrength: 10, BonusMoney: 20, SkillBonus: 0.05, Description: "经商：初始金钱 +200，打劫收益 +20%"},
	"刺客": {Name: "刺客", BonusStrength: 25, BonusMoney: 15, SkillBonus: 0.10, Description: "暗杀：暴击率 +10%，暴击时伤害翻倍"},
	"牧师": {Name: "牧师", BonusStrength: 15, BonusMoney: 12, SkillBonus: 0.08, Description: "治愈：每小时恢复 5 点金钱"},
}

// Equipment 装备定义
type Equipment struct {
	Name          string // 装备名称
	Price         int    // 价格
	StrengthBonus int    // 武力加成
	Description   string // 描述
}

// equipmentStore 装备商店静态表
var equipmentStore = map[string]Equipment{
	"木棍": {Name: "木棍", Price: 50, StrengthBonus: 5, Description: "简陋的武器"},
	"铁剑": {Name: "铁剑", Price: 150, StrengthBonus: 12, Description: "锋利的铁剑"},
	"钢刀": {Name: "钢刀", Price: 400, StrengthBonus: 20, Description: "精钢打造的刀"},
	"匕首": {Name: "匕首", Price: 300, StrengthBonus: 15, Description: "刺客的最爱"},
	"护甲": {Name: "护甲", Price: 500, StrengthBonus: 0, Description: "减少被抢劫损失 30%"},
	"头盔": {Name: "头盔", Price: 200, StrengthBonus: 0, Description: "提供额外保护"},
}

// PlayerData 玩家数据
type PlayerData struct {
	UserID                 string   `json:"user_id"`                  // 用户 wxid
	DisplayName            string   `json:"display_name"`             // 显示名称（从群成员信息获取）
	Money                  int      `json:"money"`                    // 金钱
	Strength               int      `json:"strength"`                 // 武力
	CrimeCount             int      `json:"crime_count"`              // 犯罪次数
	InJail                 bool     `json:"in_jail"`                  // 是否被监禁
	JailTurns              int      `json:"jail_turns"`               // 剩余监禁回合
	Profession             string   `json:"profession"`               // 职业
	LastJobTime            int64    `json:"last_job_time"`            // 最后完成任务时间（Unix 毫秒）
	LastWelfareTime        int64    `json:"last_welfare_time"`        // 最后领取救济时间（Unix 毫秒）
	Equipment              []string `json:"equipment"`                // 拥有的装备
	TotalRobberyCount      int      `json:"total_robbery_count"`      // 总打劫次数
	SuccessfulRobberyCount int      `json:"successful_robbery_count"` // 成功打劫次数
}

// GainMoney 增加金钱
func (p *PlayerData) GainMoney(amount int) { p.Money += amount }

// LoseMoney 减少金钱（不低于 0）
func (p *PlayerData) LoseMoney(amount int) {
	p.Money -= amount
	if p.Money < 0 {
		p.Money = 0
	}
}

// AddStrength 增加武力
func (p *PlayerData) AddStrength(amount int) { p.Strength += amount }

// IncreaseCrime 犯罪次数 +1
func (p *PlayerData) IncreaseCrime() { p.CrimeCount++ }

// ResetCrime 重置犯罪次数
func (p *PlayerData) ResetCrime() { p.CrimeCount = 0 }

// ReduceCrime 减少犯罪次数（不低于 0）
func (p *PlayerData) ReduceCrime(amount int) {
	p.CrimeCount -= amount
	if p.CrimeCount < 0 {
		p.CrimeCount = 0
	}
}

// DecreaseJailTurns 监禁回合 -1
func (p *PlayerData) DecreaseJailTurns() { p.JailTurns-- }

// HasEquipment 是否拥有指定装备
func (p *PlayerData) HasEquipment(name string) bool {
	for _, e := range p.Equipment {
		if e == name {
			return true
		}
	}
	return false
}

// AddEquipment 添加装备
func (p *PlayerData) AddEquipment(name string) {
	if !p.HasEquipment(name) {
		p.Equipment = append(p.Equipment, name)
	}
}

// IncreaseTotalRobbery 总打劫次数 +1
func (p *PlayerData) IncreaseTotalRobbery() { p.TotalRobberyCount++ }

// IncreaseSuccessfulRobbery 成功打劫次数 +1
func (p *PlayerData) IncreaseSuccessfulRobbery() { p.SuccessfulRobberyCount++ }
