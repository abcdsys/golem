package main

import (
	"bytes"
	"fmt"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/sbgayhub/golem/sdk/contact"
	"github.com/sbgayhub/golem/sdk/message"
)

func (p *FarmPlugin) sendText(receiver *contact.Contact, text string) {
	msg := &message.Message{
		Type:     message.TypeText,
		Receiver: receiver,
		Content:  text,
		Data:     &message.Message_Text{Text: &message.TextData{Content: text}},
	}
	if _, err := p.message.Send(msg); err != nil {
		slog.Error("[farm] 发送文本失败", "err", err)
	}
}

func (p *FarmPlugin) sendImageIfAvailable(receiver *contact.Contact, imageName string) {
	path := filepath.Join(p.Config.ImageDir, imageName)
	data, err := os.ReadFile(path)
	if err != nil {
		slog.Warn("[farm] 读取图片失败", "path", path, "err", err)
		return
	}
	if len(data) == 0 {
		return
	}
	_, err = p.cdn.UploadImage(receiver.GetUsername(), bytes.NewReader(data))
	if err != nil {
		slog.Warn("[farm] 发送图片失败", "path", path, "err", err)
	}
}

func (p *FarmPlugin) printMenu(replyTo *contact.Contact) {
	p.sendText(replyTo,
		" === 农场菜单 === \n\n"+
			"农场帮助\n"+
			"农场商店 守卫商店\n"+
			"农场购买种子 查询种子\n"+
			"农场购买守卫 查询守卫\n"+
			"农场购买 种植 收菜 偷菜 浇水\n"+
			"我的农场 农场等级\n"+
			"购买土地 ")
}

func (p *FarmPlugin) printHelp(replyTo *contact.Contact) {
	p.sendText(replyTo,
		"　　农场: 主人无聊开发的小游戏\n\n"+
			"货币系统: "+emojiSun+"(阳光)是农场中的基本货币\n\n"+
			"升级系统: "+emojiExp+"(经验值)可以提高农场等级\n\n"+
			"　　作物: 种植种子, 经过一段时间可以 收获"+emojiSun+"(阳光)和"+emojiExp+"(经验值)\n\n"+
			"　　土地: 土地越多, 可以同时种的种子个数\n\n"+
			"　　偷菜: 赚点小外快?\n\n"+
			"　　查询: 查询种子或者其他物品的功能 例如'查询土豆'\n\n"+
			"    守卫: 特效宠物, 防止被偷, 打盹时触发减半\n"+
			"    浇水: 获得经验值, 并且增加产量, 一株植物在成熟之前每个阶段可以浇水一次")
}

func (p *FarmPlugin) printHelpBuy(replyTo *contact.Contact) {
	p.sendText(replyTo,
		"发送 \"农场购买+种子名称\" 购买相应种子, 例如 \"农场购买土豆\".\n\n"+
			"发送 \"农场购买+种子名称+数量\" 购买多个种子, 例如 \"农场购买土豆15\".\n\n"+
			"发送 \"农场购买+守卫名称\" 购买相应守卫, 例如 \"农场购买"+pets[0].Name+"\".\n\n"+
			"使用\"农场商店\"或者\"守卫商店\"查看列表")
}

func (p *FarmPlugin) printHelpSearch(replyTo *contact.Contact) {
	p.sendText(replyTo,
		"发送 \"查询+种子名称\" 查询预计收益, 例如 \"查询土豆\".\n\n"+
			"发送 \"查询+守卫名称\" 查询预计收益, 例如 \"查询"+pets[0].Name+"\".")
}

func (p *FarmPlugin) printHelpPlant(replyTo *contact.Contact) {
	p.sendText(replyTo, "发送 \"种+种子名称\" 种植作物, 例如 \"种土豆\".")
}

func (p *FarmPlugin) printHelpSteal(replyTo *contact.Contact) {
	p.sendText(replyTo, "发送 \"偷菜+@一个人\" 可以偷菜, 例如 \"偷菜@张三\".")
}

func (p *FarmPlugin) printSelf(chatroomID, userID string, replyTo *contact.Contact) {
	player := p.getOrCreatePlayer(chatroomID, userID)
	level := computeLevel(player.Exp)
	builder := strings.Builder{}
	fmt.Fprintf(&builder, "阳光　%s　%d\n土地　%s️　%d\n经验　%s　%d\n等级　%s️　%d\n",
		emojiSun, player.Coins,
		emojiField, player.Fields,
		emojiExp, player.Exp,
		emojiLevel, level)
	p.appendAssetSummary(&builder, player)
	p.sendText(replyTo, builder.String())
	p.sendFarmStatusImage(replyTo, player, now())
}

func (p *FarmPlugin) appendAssetSummary(builder *strings.Builder, player *FarmPlayer) {
	var seedLines []string
	for _, crop := range crops {
		count := player.CropCount[strconv.Itoa(crop.Level)]
		if count > 0 {
			seedLines = append(seedLines, crop.FruitEmoji+crop.Name+"x"+strconv.Itoa(count))
		}
	}
	builder.WriteString("\n种子: ")
	if len(seedLines) == 0 {
		builder.WriteString("无")
	} else {
		builder.WriteString(strings.Join(seedLines, "，"))
	}

	var petLines []string
	for _, pet := range pets {
		for _, owned := range player.Pets {
			if owned == pet.Level {
				petLines = append(petLines, emojiDog+pet.Name)
				break
			}
		}
	}
	builder.WriteString("\n守卫: ")
	if len(petLines) == 0 {
		builder.WriteString("无")
	} else {
		builder.WriteString(strings.Join(petLines, "，"))
	}
}

func (p *FarmPlugin) sendFarmStatusImage(replyTo *contact.Contact, player *FarmPlayer, nowTime int64) {
	if player.Fields <= 0 {
		p.sendImageIfAvailable(replyTo, unplantedImagePath)
		return
	}
	hasPlant := false
	maxStage := -1
	for i := 0; i < player.Fields; i++ {
		field := player.LandFields[strconv.Itoa(i)]
		if field == nil || field.Level <= 0 {
			continue
		}
		crop := cropByLevel[field.Level]
		if crop == nil {
			continue
		}
		hasPlant = true
		stage := p.growthStageIndex(crop, field.PlantTime, nowTime)
		if stage > maxStage {
			maxStage = stage
		}
	}
	if !hasPlant {
		p.sendImageIfAvailable(replyTo, unplantedImagePath)
		return
	}
	index := max(0, min(len(growthImagePaths)-1, maxStage))
	p.sendImageIfAvailable(replyTo, growthImagePaths[index])
}

func (p *FarmPlugin) growthStageIndex(crop *Crop, plantTime, nowTime int64) int {
	elapsed := max(0, nowTime-plantTime)
	total := crop.TotalGrowSeconds()
	if total <= 0 {
		return len(growthImagePaths) - 1
	}
	if elapsed >= total {
		return len(growthImagePaths) - 1
	}
	ratio := float64(elapsed) / float64(total)
	preMatureMax := len(growthImagePaths) - 2
	index := int(math.Floor(ratio * float64(len(growthImagePaths)-1)))
	return max(0, min(preMatureMax, index))
}

func (p *FarmPlugin) printLevels(chatroomID, userID string, replyTo *contact.Contact) {
	player := p.getOrCreatePlayer(chatroomID, userID)
	level := computeLevel(player.Exp)
	builder := strings.Builder{}
	fmt.Fprintf(&builder, "当前农场等级为%d级(%s%d), ", level, emojiExp, player.Exp)
	if level >= 20 {
		builder.WriteString("您已满级.")
	} else {
		needExp := ((int64(math.Pow(float64(level+1), 4)) - 1) / 5) - player.Exp
		fmt.Fprintf(&builder, "距离升级还需要%s%d", emojiExp, needExp)
	}
	p.sendText(replyTo, builder.String())
}

func (p *FarmPlugin) printCrops(chatroomID, userID string, replyTo *contact.Contact) {
	player := p.getOrCreatePlayer(chatroomID, userID)
	level := computeLevel(player.Exp)
	builder := strings.Builder{}
	builder.WriteString(fmt.Sprintf("%s 　　　　　　%s　 %s\n", emojiLevel, emojiSun, emojiStock))
	for _, crop := range crops {
		fmt.Fprintf(&builder, "%02d　%s　%s　%d　", crop.Level, crop.FruitEmoji, crop.Name, crop.SeedPrice)
		padding := ""
		if crop.SeedPrice < 10 {
			padding = "     "
		} else if crop.SeedPrice < 100 {
			padding = "   "
		} else if crop.SeedPrice < 1000 {
			padding = " "
		}
		builder.WriteString(padding)
		count := player.CropCount[strconv.Itoa(crop.Level)]
		builder.WriteString(strconv.Itoa(count))
		builder.WriteString("\n")
	}
	fmt.Fprintf(&builder, "\n%s　%d　　　%s　%d", emojiLevel, level, emojiSun, player.Coins)
	p.sendText(replyTo, builder.String())
}

func (p *FarmPlugin) printPets(chatroomID, userID string, replyTo *contact.Contact) {
	player := p.getOrCreatePlayer(chatroomID, userID)
	level := computeLevel(player.Exp)
	builder := strings.Builder{}
	builder.WriteString(fmt.Sprintf("%s 　　　　     　　%s　%s\n", emojiLevel, emojiSun, emojiStock))
	for _, pet := range pets {
		fmt.Fprintf(&builder, "%02d　%s　%s　%d　", pet.Level, emojiDog, pet.Name, pet.Price)
		has := false
		for _, owned := range player.Pets {
			if owned == pet.Level {
				has = true
				break
			}
		}
		if has {
			builder.WriteString("🈶️")
		} else {
			builder.WriteString("🈚️")
		}
		builder.WriteString("\n")
	}
	fmt.Fprintf(&builder, "\n%s　%d　　　%s　%d", emojiLevel, level, emojiSun, player.Coins)
	p.sendText(replyTo, builder.String())
}

func (p *FarmPlugin) cropState(crop *Crop, plantTime, nowTime int64) CropState {
	elapsed := nowTime - plantTime
	state := 0
	emoji := crop.StepEmojis[0]
	for i, hours := range crop.StepHours {
		state = i
		emoji = crop.StepEmojis[i]
		band := int64(hours) * 3600
		if elapsed > band {
			if i == len(crop.StepHours)-1 {
				state = mature
				emoji = crop.FruitEmoji
				break
			}
			elapsed -= band
		} else {
			break
		}
	}
	return CropState{State: state, Emoji: emoji}
}
