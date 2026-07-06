package main

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/sbgayhub/golem/sdk/contact"
)

// showAssets 显示玩家资产
func (p *RobberyPlugin) showAssets(player *PlayerData, replyTo *contact.Contact) {
	var response strings.Builder
	response.WriteString(fmt.Sprintf("📊 %s 的资产信息\n", player.DisplayName))
	response.WriteString(fmt.Sprintf("💰 金钱：%d 金币\n", player.Money))
	response.WriteString(fmt.Sprintf("💪 武力：%d\n", p.getEffectiveStrength(player)))
	response.WriteString(fmt.Sprintf("📝 犯罪记录：%d 次\n", player.CrimeCount))

	if player.Profession != "" {
		response.WriteString(fmt.Sprintf("🎭 职业：%s\n", player.Profession))
	}

	response.WriteString(fmt.Sprintf("⚔️ 总打劫次数：%d\n", player.TotalRobberyCount))
	response.WriteString(fmt.Sprintf("✅ 成功次数：%d\n", player.SuccessfulRobberyCount))

	if len(player.Equipment) > 0 {
		response.WriteString(fmt.Sprintf("🎒 装备：%s\n", strings.Join(player.Equipment, ", ")))
	}

	if player.InJail {
		response.WriteString(fmt.Sprintf("🔒 状态：监禁中（剩余 %d 回合）", player.JailTurns))
	} else {
		response.WriteString("😊 状态：自由身")
	}

	p.sendText(replyTo, response.String())
}

// showLeaderboard 显示财富排行榜
func (p *RobberyPlugin) showLeaderboard(chatroomID string, replyTo *contact.Contact) {
	p.mu.RLock()
	playersInGroup, ok := p.data[chatroomID]
	p.mu.RUnlock()

	if !ok || len(playersInGroup) == 0 {
		p.sendText(replyTo, "还没有玩家数据呢~")
		return
	}

	// 复制一份用于排序
	players := make([]*PlayerData, 0, len(playersInGroup))
	for _, pd := range playersInGroup {
		players = append(players, pd)
	}

	// 按金钱排序（降序）
	for i := 0; i < len(players); i++ {
		for j := i + 1; j < len(players); j++ {
			if players[j].Money > players[i].Money {
				players[i], players[j] = players[j], players[i]
			}
		}
	}

	var response strings.Builder
	response.WriteString("🏆 财富排行榜 🏆\n\n")

	rank := 1
	for _, player := range players {
		if rank > 10 {
			break
		}
		response.WriteString(fmt.Sprintf("%d. %s - %d 金币\n",
			rank, player.DisplayName, player.Money))
		rank++
	}

	p.sendText(replyTo, response.String())
}

// showProfessions 显示职业列表
func (p *RobberyPlugin) showProfessions(replyTo *contact.Contact) {
	var response strings.Builder
	response.WriteString("🎭 可选职业列表\n\n")

	for _, prof := range professions {
		response.WriteString(fmt.Sprintf("【%s】\n", prof.Name))
		response.WriteString(fmt.Sprintf("  武力加成：+%d\n", prof.BonusStrength))
		response.WriteString(fmt.Sprintf("  金钱加成：+%d\n", prof.BonusMoney))
		response.WriteString(fmt.Sprintf("  技能：%s\n", prof.Description))
		response.WriteString("\n")
	}

	response.WriteString(fmt.Sprintf("\n使用 \"转职 + 职业名\" 进行转职（需 %d 金币）", p.Config.ChangeProfCost))
	p.sendText(replyTo, response.String())
}

// showShop 显示装备商店
func (p *RobberyPlugin) showShop(replyTo *contact.Contact) {
	var response strings.Builder
	response.WriteString("🛒 装备商店\n\n")

	for _, equip := range equipmentStore {
		response.WriteString(fmt.Sprintf("【%s】-%d金币\n", equip.Name, equip.Price))
		response.WriteString(fmt.Sprintf("  武力 +%d | %s\n", equip.StrengthBonus, equip.Description))
		response.WriteString("\n")
	}

	response.WriteString("\n使用 \"购买 + 装备名\" 购买装备")
	p.sendText(replyTo, response.String())
}

// buyEquipment 购买装备
func (p *RobberyPlugin) buyEquipment(player *PlayerData, content string, replyTo *contact.Contact) {
	parts := strings.Fields(content)
	if len(parts) < 2 {
		p.sendText(replyTo, "格式错误！请使用：购买 + 装备名（如：购买 铁剑）")
		return
	}

	equipName := parts[1]
	equip, ok := equipmentStore[equipName]
	if !ok {
		p.sendText(replyTo, "没有这个装备哦！请使用\"商店\"查看")
		return
	}

	if player.Money < equip.Price {
		p.sendText(replyTo, "你的钱不够买这个装备呢~")
		return
	}

	if player.HasEquipment(equipName) {
		p.sendText(replyTo, "你已经拥有这个装备了！")
		return
	}

	player.LoseMoney(equip.Price)
	player.AddEquipment(equipName)

	p.sendText(replyTo,
		fmt.Sprintf("✅ %s 购买了 %s！\n当前金钱：%d | 武力：%d",
			player.DisplayName, equipName, player.Money, p.getEffectiveStrength(player)))

	if err := p.saveData(); err != nil {
		slog.Error("[robbery] 保存数据失败", "err", err)
	}
}

// changeProfession 转换职业
func (p *RobberyPlugin) changeProfession(player *PlayerData, content string, replyTo *contact.Contact) {
	parts := strings.Fields(content)
	if len(parts) < 2 {
		p.sendText(replyTo, "格式错误！请使用：转职 + 职业名（如：转职 盗贼）")
		return
	}

	professionName := parts[1]
	prof, ok := professions[professionName]
	if !ok {
		p.sendText(replyTo, "没有这个职业哦！请使用\"职业列表\"查看")
		return
	}

	if player.Money < p.Config.ChangeProfCost {
		p.sendText(replyTo, fmt.Sprintf("转职需要 %d 金币，你的钱不够呢~", p.Config.ChangeProfCost))
		return
	}

	player.LoseMoney(p.Config.ChangeProfCost)
	player.Profession = professionName

	player.AddStrength(prof.BonusStrength)
	player.GainMoney(prof.BonusMoney)

	p.sendText(replyTo,
		fmt.Sprintf("✅ %s 成功转职为 %s！\n当前金钱：%d | 武力：%d",
			player.DisplayName, professionName, player.Money, p.getEffectiveStrength(player)))

	if err := p.saveData(); err != nil {
		slog.Error("[robbery] 保存数据失败", "err", err)
	}
}

// useSkill 使用/查看技能
func (p *RobberyPlugin) useSkill(player *PlayerData, replyTo *contact.Contact) {
	if player.Profession == "" {
		p.sendText(replyTo, "你还没有职业呢！请先使用\"职业列表\"查看并转职")
		return
	}

	prof, ok := professions[player.Profession]
	if !ok {
		p.sendText(replyTo, "你的职业数据异常，请重新转职")
		return
	}

	p.sendText(replyTo,
		fmt.Sprintf("✨ %s 的职业技能：%s\n成功率加成：%.0f%%",
			player.DisplayName, prof.Description, prof.SkillBonus*100))
}

// showJobs 显示每日任务
func (p *RobberyPlugin) showJobs(player *PlayerData, replyTo *contact.Contact) {
	now := time.Now().UnixMilli()
	hoursSinceLastJob := (now - player.LastJobTime) / (1000 * 60 * 60)

	if hoursSinceLastJob < int64(p.Config.JobRefreshHours) {
		hoursLeft := int64(p.Config.JobRefreshHours) - hoursSinceLastJob
		p.sendText(replyTo,
			fmt.Sprintf("📋 任务尚未刷新，还需等待 %d 小时", hoursLeft))
		return
	}

	// 生成随机任务
	tasks := []string{
		"打劫成功 3 次",
		"积累 1000 金币",
		"武力达到 50",
		"帮助群友（发送消息 10 条）",
	}
	task := tasks[p.rand.Intn(len(tasks))]
	reward := 100 + p.rand.Intn(201) // 100-300 金币

	p.sendText(replyTo,
		fmt.Sprintf("📋 今日任务：%s\n💰 奖励：%d 金币\n\n明天再来完成任务吧！", task, reward))

	player.LastJobTime = now
	if err := p.saveData(); err != nil {
		slog.Error("[robbery] 保存数据失败", "err", err)
	}
}

// claimWelfare 领取救济
func (p *RobberyPlugin) claimWelfare(player *PlayerData, replyTo *contact.Contact) {
	if player.Money >= 100 {
		p.sendText(replyTo, "你的钱还够花，不需要领取救济哦~")
		return
	}

	now := time.Now().UnixMilli()
	hoursSinceLastWelfare := (now - player.LastWelfareTime) / (1000 * 60 * 60)

	if hoursSinceLastWelfare < int64(p.Config.WelfareCooldown) {
		hoursLeft := int64(p.Config.WelfareCooldown) - hoursSinceLastWelfare
		p.sendText(replyTo,
			fmt.Sprintf("🕐 救济金还在冷却中，还需等待 %d 小时", hoursLeft))
		return
	}

	player.GainMoney(p.Config.WelfareAmount)
	player.LastWelfareTime = now

	p.sendText(replyTo,
		fmt.Sprintf("✅ %s 领取了 %d 金币救济金！\n当前金钱：%d",
			player.DisplayName, p.Config.WelfareAmount, player.Money))

	if err := p.saveData(); err != nil {
		slog.Error("[robbery] 保存数据失败", "err", err)
	}
}

// showHelp 显示帮助菜单
func (p *RobberyPlugin) showHelp(replyTo *contact.Contact) {
	var response strings.Builder
	response.WriteString("🎮【打劫游戏】帮助菜单🎮\n\n")
	response.WriteString("💰 基础玩法：\n")
	response.WriteString("• 打劫 - 随机打劫其他群成员\n")
	response.WriteString("• 我的资产 - 查看金钱、武力、职业\n")
	response.WriteString("• 排行榜 - 查看群内财富排行\n\n")

	response.WriteString("⚔️ 职业系统：\n")
	response.WriteString(fmt.Sprintf("• 职业列表 - 查看可选职业\n"))
	response.WriteString(fmt.Sprintf("• 转职 + 职业名 - 转换职业 (%d 金币)\n", p.Config.ChangeProfCost))
	response.WriteString("• 技能 - 查看职业技能\n\n")

	response.WriteString("🛒 装备商店：\n")
	response.WriteString("• 商店 - 查看装备\n")
	response.WriteString("• 购买 + 装备名 - 购买装备\n\n")

	response.WriteString("🎲 娱乐玩法：\n")
	response.WriteString("• 任务 - 领取每日任务\n")
	response.WriteString("• 赌博 + 金额 - 赌场博弈\n")
	response.WriteString("• 救济 - 领取破产补助\n\n")

	response.WriteString("💡 新手提示：\n")
	response.WriteString("发送\"我的资产\"开始游戏！\n")
	response.WriteString("更多功能持续开发中...\n")
	response.WriteString("\n发送\"打劫帮助\"再次查看本菜单")

	p.sendText(replyTo, response.String())
}
