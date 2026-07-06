package main

import (
	"fmt"
	"log/slog"
	"math"
	"strings"

	"github.com/sbgayhub/golem/sdk/contact"
)

// getEffectiveStrength 获取有效武力（基础武力 + 职业加成 + 装备加成）
func (p *RobberyPlugin) getEffectiveStrength(player *PlayerData) int {
	strength := player.Strength

	if player.Profession != "" {
		if prof, ok := professions[player.Profession]; ok {
			strength += prof.BonusStrength
		}
	}

	for _, equipName := range player.Equipment {
		if equip, ok := equipmentStore[equipName]; ok {
			strength += equip.StrengthBonus
		}
	}

	return strength
}

// performRobbery 执行打劫操作
func (p *RobberyPlugin) performRobbery(player *PlayerData, chatroomID string, replyTo *contact.Contact) {
	targets := p.getEligibleTargets(chatroomID, player.UserID)
	if len(targets) == 0 {
		p.sendText(replyTo, "现在没有可以打劫的目标呢~")
		return
	}

	target := targets[p.rand.Intn(len(targets))]
	successChance := p.calculateSuccessChance(player, target)
	success := p.rand.Float64() < successChance

	var response strings.Builder

	if success {
		stolen := p.calculateStolenAmount(target)

		// 商人职业有额外收益
		if player.Profession == "商人" {
			stolen = int(float64(stolen) * 1.2)
		}

		player.GainMoney(stolen)
		target.LoseMoney(stolen)
		player.IncreaseCrime()
		player.IncreaseTotalRobbery()
		player.IncreaseSuccessfulRobbery()

		response.WriteString(fmt.Sprintf("✅ %s 成功打劫了 %s，抢得 %d 金币！\n",
			player.DisplayName, target.DisplayName, stolen))
		response.WriteString(fmt.Sprintf("当前金钱：%d | 武力：%d", player.Money, p.getEffectiveStrength(player)))

		// 30% 概率触发随机事件
		if p.rand.Float64() < eventTriggerChance {
			response.WriteString("\n\n")
			response.WriteString(p.triggerRandomEvent(player))
		}

		// 检查是否被通缉
		p.checkWantedStatus(player, &response)
	} else {
		penalty := p.calculatePenaltyAmount(player)
		player.LoseMoney(penalty)
		target.GainMoney(penalty)
		player.IncreaseTotalRobbery()

		response.WriteString(fmt.Sprintf("❌ %s 打劫失败，反被 %s 抢走 %d 金币！\n",
			player.DisplayName, target.DisplayName, penalty))
		response.WriteString(fmt.Sprintf("剩余金钱：%d", player.Money))
	}

	p.sendText(replyTo, response.String())
	if err := p.saveData(); err != nil {
		slog.Error("[robbery] 保存数据失败", "err", err)
	}
}

// calculateSuccessChance 计算打劫成功率
// 公式：攻击方武力² / (攻击方武力² + 防御方武力²)
func (p *RobberyPlugin) calculateSuccessChance(attacker, defender *PlayerData) float64 {
	attackerStrength := float64(p.getEffectiveStrength(attacker))
	defenderStrength := float64(p.getEffectiveStrength(defender))

	attackerPower := math.Pow(attackerStrength, 2)
	defenderPower := math.Pow(defenderStrength, 2)
	baseProbability := attackerPower / (attackerPower + defenderPower)

	// 职业加成
	professionBonus := 0.0
	if attacker.Profession != "" {
		if prof, ok := professions[attacker.Profession]; ok {
			professionBonus = prof.SkillBonus
		}
	}

	// 添加随机波动和保底机制
	adjusted := baseProbability*0.8 + 0.1 + professionBonus + p.rand.Float64()*0.2
	// 限制在 10%-90% 之间
	if adjusted < 0.1 {
		adjusted = 0.1
	}
	if adjusted > 0.9 {
		adjusted = 0.9
	}
	return adjusted
}

// calculateStolenAmount 计算抢劫金额（目标当前金钱的 20%-50%）
func (p *RobberyPlugin) calculateStolenAmount(target *PlayerData) int {
	if target.Money <= 0 {
		return 0
	}

	// 商人被打劫时损失更多
	bonus := 1.0
	if target.Profession == "商人" {
		bonus = 1.2
	}

	maxAmount := int(float64(target.Money) * 0.5 * bonus)
	minAmount := int(float64(target.Money) * 0.2 * bonus)
	if maxAmount <= minAmount {
		return 1
	}
	return p.rand.Intn(maxAmount-minAmount+1) + minAmount
}

// calculatePenaltyAmount 计算惩罚金额（自身当前金钱的 10%-30%）
func (p *RobberyPlugin) calculatePenaltyAmount(attacker *PlayerData) int {
	if attacker.Money <= 0 {
		return 0
	}

	// 装备护甲可以减少损失
	reduction := 1.0
	if attacker.HasEquipment("护甲") {
		reduction = 0.7
	}

	maxAmount := int(float64(attacker.Money) * 0.3 * reduction)
	minAmount := int(float64(attacker.Money) * 0.1 * reduction)
	if maxAmount <= minAmount {
		return 1
	}
	return p.rand.Intn(maxAmount-minAmount+1) + minAmount
}

// checkWantedStatus 检查是否被通缉
func (p *RobberyPlugin) checkWantedStatus(player *PlayerData, response *strings.Builder) {
	wantedChance := baseWantedChance + float64(player.CrimeCount)*crimeIncrement

	if p.rand.Float64() < wantedChance {
		// 生成卫兵（1-3 名，每个武力 25）
		guardCount := 1 + p.rand.Intn(3)
		totalGuardStrength := guardCount * 25

		response.WriteString(fmt.Sprintf("\n\n⚠️ 你的恶行引起了卫兵注意！%d 名卫兵正在追捕你！", guardCount))

		// 逃脱判定
		if float64(player.Strength) > float64(totalGuardStrength)*0.6 {
			response.WriteString(fmt.Sprintf("\n🏃 你成功摆脱了%d名卫兵的追捕！", guardCount))
		} else {
			fine := int(float64(player.Money) * 0.4)
			player.LoseMoney(fine)
			player.InJail = true
			player.JailTurns = 3
			player.ResetCrime()

			response.WriteString(fmt.Sprintf("\n🔒 你被%d名卫兵抓获！缴纳罚金%d金币并监禁 3 回合", guardCount, fine))
		}
	}
}

// triggerRandomEvent 触发随机事件
func (p *RobberyPlugin) triggerRandomEvent(player *PlayerData) string {
	eventType := p.rand.Intn(100)

	if eventType < 40 {
		// 黑市事件（40%）
		cost := 50 + p.rand.Intn(151) // 50-200 金币
		if player.Money >= cost {
			player.LoseMoney(cost)
			player.AddStrength(3)
			return fmt.Sprintf("🕶️ 黑市商人出售强化药剂，支付%d金币提升武力值 3 点！\n当前武力：%d",
				cost, player.Strength)
		}
		return "🕶️ 黑市商人出现，但你的钱不够购买强化药剂..."
	} else if eventType < 70 {
		// 意外之财（30%）
		windfall := 200 + p.rand.Intn(301) // 200-500 金币
		player.GainMoney(windfall)
		return fmt.Sprintf("💰 你发现了逃跑的运钞车！捡到%d金币！", windfall)
	}

	// 减少犯罪记录（30%）
	reduce := 2
	if player.CrimeCount < 2 {
		reduce = player.CrimeCount
	}
	player.ReduceCrime(reduce)
	return fmt.Sprintf("💼 神秘商人提供洗钱服务，犯罪记录减少%d次！", reduce)
}

// handleJailStatus 处理监禁状态
func (p *RobberyPlugin) handleJailStatus(player *PlayerData, replyTo *contact.Contact) {
	player.DecreaseJailTurns()

	if player.JailTurns <= 0 {
		player.InJail = false
		p.sendText(replyTo,
			fmt.Sprintf("✅ %s 刑满释放，重获自由！\n当前金钱：%d",
				player.DisplayName, player.Money))
	} else {
		p.sendText(replyTo,
			fmt.Sprintf("🔒 %s 正在服刑，剩余 %d 回合\n无法进行任何活动",
				player.DisplayName, player.JailTurns))
	}

	if err := p.saveData(); err != nil {
		slog.Error("[robbery] 保存数据失败", "err", err)
	}
}

// gamble 赌博系统
func (p *RobberyPlugin) gamble(player *PlayerData, content string, replyTo *contact.Contact) {
	parts := strings.Fields(content)
	if len(parts) < 2 {
		p.sendText(replyTo, "格式错误！请使用：赌博 + 金额（如：赌博 100）")
		return
	}

	var amount int
	if _, err := fmt.Sscanf(parts[1], "%d", &amount); err != nil {
		p.sendText(replyTo, "请输入有效的金额数字！")
		return
	}

	if amount <= 0 {
		p.sendText(replyTo, "赌博金额必须大于 0！")
		return
	}

	if player.Money < amount {
		p.sendText(replyTo, "你的钱不够下注呢~")
		return
	}

	// 50% 胜率，商人有 10% 额外胜率
	winChance := 0.5
	if player.Profession == "商人" {
		winChance += 0.1
	}

	if p.rand.Float64() < winChance {
		winAmount := amount * 2
		player.GainMoney(winAmount)
		p.sendText(replyTo,
			fmt.Sprintf("🎲 %s 赢了！赢得 %d 金币！\n当前金钱：%d",
				player.DisplayName, winAmount, player.Money))
	} else {
		player.LoseMoney(amount)
		p.sendText(replyTo,
			fmt.Sprintf("❌ %s 输了！损失 %d 金币\n剩余金钱：%d",
				player.DisplayName, amount, player.Money))
	}

	if err := p.saveData(); err != nil {
		slog.Error("[robbery] 保存数据失败", "err", err)
	}
}
