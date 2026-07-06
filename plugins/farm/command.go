package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/sbgayhub/golem/sdk/contact"
	"github.com/sbgayhub/golem/sdk/message"
)

func (p *FarmPlugin) plant(chatroomID, userID, name string, replyTo *contact.Contact) {
	crop := cropByName[name]
	if crop == nil {
		p.sendText(replyTo, "没有这个种子哦！请使用\"农场商店\"查看")
		return
	}
	player := p.getOrCreatePlayer(chatroomID, userID)
	nowTime := now()
	builder := strings.Builder{}
	var expUp int64

	for i := 0; i < player.Fields; i++ {
		fmt.Fprintf(&builder, "土地(%d) ", i+1)
		key := strconv.Itoa(i)
		field := player.LandFields[key]
		if field != nil && field.Level > 0 {
			planted := cropByLevel[field.Level]
			state := p.cropState(planted, field.PlantTime, nowTime)
			fmt.Fprintf(&builder, "%s (%s 已存在)", state.Emoji, planted.Name)
		} else {
			stock := player.CropCount[strconv.Itoa(crop.Level)]
			if stock > 0 {
				player.CropCount[strconv.Itoa(crop.Level)] = stock - 1
				newField := &Field{
					Level:     crop.Level,
					PlantTime: nowTime,
					Watered:   make(map[string]string),
					Stealer:   []string{},
					Alerted:   []string{},
				}
				player.LandFields[key] = newField
				expUp += int64(crop.FruitExp)
				fmt.Fprintf(&builder, " => %s", crop.FruitEmoji)
			} else {
				fmt.Fprintf(&builder, "%s种子不足", crop.Name)
			}
		}
		builder.WriteString("\n")
	}
	if expUp > 0 {
		player.Exp += expUp
		fmt.Fprintf(&builder, "\n%s ↑ %d => %d", emojiExp, expUp, player.Exp)
		p.saveData()
	}
	p.sendText(replyTo, builder.String())
	if expUp > 0 {
		p.sendImageIfAvailable(replyTo, plantedImagePath)
		p.sendImageIfAvailable(replyTo, growthImagePaths[0])
	}
}

func (p *FarmPlugin) collect(chatroomID, userID string, replyTo *contact.Contact) {
	player := p.getOrCreatePlayer(chatroomID, userID)
	nowTime := now()
	builder := strings.Builder{}
	var expUp, coinsUp int64
	waterSet := make(map[string]struct{})
	stealerSet := make(map[string]struct{})

	for i := 0; i < player.Fields; i++ {
		fmt.Fprintf(&builder, "土地(%d) ", i+1)
		key := strconv.Itoa(i)
		field := player.LandFields[key]
		if field != nil && field.Level > 0 {
			planted := cropByLevel[field.Level]
			state := p.cropState(planted, field.PlantTime, nowTime)
			if state.State == mature {
				fruitNumber := planted.RandomFruitCount(p.random)
				fruitNumber += len(field.Watered)
				for waterer := range field.Watered {
					if waterer != userID {
						waterSet[waterer] = struct{}{}
					}
				}
				fruitNumber -= len(field.Stealer)
				if fruitNumber < 0 {
					fruitNumber = 0
				}
				fmt.Fprintf(&builder, "%s (%s %d枚)", state.Emoji, planted.Name, fruitNumber)
				if len(field.Stealer) > 0 {
					fmt.Fprintf(&builder, "(被偷%d枚)", len(field.Stealer))
					for _, s := range field.Stealer {
						stealerSet[s] = struct{}{}
					}
				}
				expUp += int64(fruitNumber) * int64(planted.FruitExp)
				coinsUp += int64(fruitNumber) * int64(planted.FruitPrice)
				delete(player.LandFields, key)
			} else {
				_, watered := field.Watered[strconv.Itoa(state.State)]
				if watered {
					fmt.Fprintf(&builder, "%s (%s 未成熟)", state.Emoji+emojiWater, planted.Name)
				} else {
					fmt.Fprintf(&builder, "%s (%s 未成熟)", state.Emoji, planted.Name)
				}
			}
		} else {
			builder.WriteString("未种植")
		}
		builder.WriteString("\n")
	}
	if expUp > 0 {
		player.Exp += expUp
		player.Coins += coinsUp

		if len(waterSet) > 0 {
			builder.WriteString("\n帮你浇水的群友 : \n")
			for waterer := range waterSet {
				fmt.Fprintf(&builder, "    %s\n", p.getDisplayName(chatroomID, waterer))
			}
		}
		if len(stealerSet) > 0 {
			builder.WriteString("\n偷你菜的群友 : \n")
			for stealer := range stealerSet {
				fmt.Fprintf(&builder, "    %s\n", p.getDisplayName(chatroomID, stealer))
			}
		}
		fmt.Fprintf(&builder, "\n%s ↑ %d => %d\n%s ↑ %d => %d", emojiExp, expUp, player.Exp, emojiSun, coinsUp, player.Coins)
		p.saveData()
	}
	p.sendText(replyTo, builder.String())
	if expUp > 0 {
		p.sendImageIfAvailable(replyTo, unplantedImagePath)
	}
}

func (p *FarmPlugin) steal(chatroomID, userID, name string, msg *message.Message, replyTo *contact.Contact) {
	targetUserID := p.extractMentionedUser(chatroomID, userID, name, msg)
	if targetUserID == "" {
		p.printHelpSteal(replyTo)
		return
	}
	if userID == targetUserID {
		p.sendText(replyTo, "你不能偷自己的菜")
		return
	}

	builder := strings.Builder{}
	fmt.Fprintf(&builder, "偷偷进入了 %s的农场\n\n", p.getDisplayName(chatroomID, targetUserID))

	target := p.getOrCreatePlayer(chatroomID, targetUserID)
	hasDog := false
	for _, petLevel := range target.Pets {
		if petLevel == 1 {
			hasDog = true
			break
		}
	}
	if hasDog {
		dog := petByLevel[1]
		fmt.Fprintf(&builder, "%s%s ", emojiDog, dog.Name)
		sleepy := p.randomInt64(p.random.Int63()-int64(targetUserIDHash(targetUserID)))%100 < 50
		alertPercentage := int64(20)
		if sleepy {
			builder.WriteString("正在瞌睡 ")
			alertPercentage /= 2
		}
		alert := p.randomInt64(p.random.Int63()-int64(targetUserIDHash(targetUserID)))%100 < alertPercentage
		if alert {
			player := p.getOrCreatePlayer(chatroomID, userID)
			penalty := min(int64(100), player.Coins)
			player.Coins -= penalty
			fmt.Fprintf(&builder, "把你咬了 损失 %s%d", emojiSun, penalty)
			p.sendText(replyTo, builder.String())
			p.saveData()
			return
		}
	}

	var expUp, coinsUp int64
	nowTime := now()
	for i := 0; i < target.Fields; i++ {
		fmt.Fprintf(&builder, "土地(%d) ", i+1)
		key := strconv.Itoa(i)
		field := target.LandFields[key]
		if field != nil && field.Level > 0 {
			planted := cropByLevel[field.Level]
			state := p.cropState(planted, field.PlantTime, nowTime)
			if state.State != mature {
				fmt.Fprintf(&builder, "%s (%s 未成熟)", state.Emoji, planted.Name)
			} else if contains(field.Stealer, userID) {
				fmt.Fprintf(&builder, "%s (%s 偷过了)", state.Emoji, planted.Name)
			} else if len(field.Stealer) >= 2 {
				fmt.Fprintf(&builder, "%s (%s 快被偷光了)", state.Emoji, planted.Name)
			} else {
				expUp += int64(planted.FruitExp)
				coinsUp += int64(planted.FruitPrice)
				field.Stealer = append(field.Stealer, userID)
				target.LandFields[key] = field
				fmt.Fprintf(&builder, "%s (%s %d枚)", state.Emoji, planted.Name, 1)
			}
		} else {
			builder.WriteString("未种植")
		}
		builder.WriteString("\n")
	}
	if expUp > 0 {
		player := p.getOrCreatePlayer(chatroomID, userID)
		player.Exp += expUp
		player.Coins += coinsUp
		fmt.Fprintf(&builder, "\n%s ↑ %d => %d\n%s ↑ %d => %d", emojiExp, expUp, player.Exp, emojiSun, coinsUp, player.Coins)
		p.saveData()
	}
	p.sendText(replyTo, builder.String())
}

func (p *FarmPlugin) water(chatroomID, userID, name string, msg *message.Message, replyTo *contact.Contact) {
	targetUserID := p.extractMentionedUser(chatroomID, userID, name, msg)
	if targetUserID == "" {
		targetUserID = userID
	}

	target := p.getOrCreatePlayer(chatroomID, targetUserID)
	builder := strings.Builder{}
	if userID != targetUserID {
		fmt.Fprintf(&builder, "%s的农场\n\n", p.getDisplayName(chatroomID, targetUserID))
	} else {
		builder.WriteString("浇水@一个人可以为群友浇水\n\n")
	}

	var expUp int64
	nowTime := now()
	for i := 0; i < target.Fields; i++ {
		fmt.Fprintf(&builder, "土地(%d) ", i+1)
		key := strconv.Itoa(i)
		field := target.LandFields[key]
		if field != nil && field.Level > 0 {
			planted := cropByLevel[field.Level]
			state := p.cropState(planted, field.PlantTime, nowTime)
			if state.State == mature {
				fmt.Fprintf(&builder, "%s (%s 已成熟)", state.Emoji, planted.Name)
			} else {
				stateKey := strconv.Itoa(state.State)
				if _, watered := field.Watered[stateKey]; watered {
					fmt.Fprintf(&builder, "%s (%s 无需浇水)", state.Emoji+emojiWater, planted.Name)
				} else {
					field.Watered[stateKey] = userID
					expUp += int64(planted.FruitExp)
					fmt.Fprintf(&builder, "%s (%s 浇水成功)", state.Emoji+emojiRain, planted.Name)
				}
			}
		} else {
			builder.WriteString("未种植")
		}
		builder.WriteString("\n")
	}

	if expUp > 0 {
		player := p.getOrCreatePlayer(chatroomID, userID)
		player.Exp += expUp
		fmt.Fprintf(&builder, "\n%s ↑ %d => %d", emojiExp, expUp, player.Exp)
		p.saveData()
	}
	p.sendText(replyTo, builder.String())
}

func (p *FarmPlugin) extractMentionedUser(chatroomID, userID, rawName string, msg *message.Message) string {
	// 优先从 TextData.Reminds 获取被 @ 用户
	textData := msg.GetText()
	if textData != nil {
		for _, id := range textData.Reminds {
			trimmed := strings.TrimSpace(id)
			if trimmed != "" && trimmed != userID {
				return trimmed
			}
		}
	}

	name := strings.TrimSpace(strings.ReplaceAll(rawName, "@", ""))
	if name == "" {
		return ""
	}
	if strings.HasPrefix(name, "wxid_") {
		return name
	}

	// 按昵称搜索群成员
	members := p.chatroom.ListMembers(chatroomID)
	for _, m := range members {
		if m.GetDisplayName() == name || m.GetNickname() == name {
			return m.GetUsername()
		}
	}

	// 按联系人搜索
	if c := p.contact.Search(name, 0, 0); c != nil {
		return c.GetUsername()
	}

	return ""
}

func (p *FarmPlugin) getDisplayName(chatroomID, userID string) string {
	if chatroomID == "" || userID == "" {
		return userID
	}
	if m := p.chatroom.GetMember(chatroomID, userID); m != nil {
		if name := strings.TrimSpace(m.GetDisplayName()); name != "" {
			return name
		}
		if name := strings.TrimSpace(m.GetNickname()); name != "" {
			return name
		}
	}
	if c := p.contact.Get(userID); c != nil {
		if name := strings.TrimSpace(c.GetRemark()); name != "" {
			return name
		}
		if name := strings.TrimSpace(c.GetNickname()); name != "" {
			return name
		}
	}
	return userID
}

var buyRegex = regexp.MustCompile(`^([\p{Han}A-Za-z]+)\s*(\d{1,5})?\s*$`)

func (p *FarmPlugin) parseBuyRequest(content string) *BuyRequest {
	matches := buyRegex.FindStringSubmatch(content)
	if matches == nil {
		return nil
	}
	name := matches[1]
	number := 1
	if matches[2] != "" {
		n, err := strconv.Atoi(matches[2])
		if err != nil || n <= 0 {
			return nil
		}
		number = n
	}
	return &BuyRequest{Name: name, Number: number}
}

func (p *FarmPlugin) buy(chatroomID, userID string, req *BuyRequest, replyTo *contact.Contact) {
	crop := cropByName[req.Name]
	if crop != nil {
		p.buyCrop(chatroomID, userID, crop, req.Number, replyTo)
		return
	}
	pet := petByName[req.Name]
	if pet != nil {
		p.buyPet(chatroomID, userID, pet, replyTo)
		return
	}
	p.sendText(replyTo, "没有这个物品哦！请使用\"农场商店/守卫商店\"查看")
}

func (p *FarmPlugin) buyCrop(chatroomID, userID string, crop *Crop, number int, replyTo *contact.Contact) {
	player := p.getOrCreatePlayer(chatroomID, userID)
	level := computeLevel(player.Exp)
	if crop.Level > level {
		p.sendText(replyTo,
			fmt.Sprintf("您不能购买超过您自身等级的作物种子, 购买%s需要%d级, 您当前为%d级. ",
				crop.Name, crop.Level, level))
		return
	}
	cost := int64(crop.SeedPrice * number)
	if player.Coins < cost {
		p.sendText(replyTo,
			fmt.Sprintf("您的阳光不足, 购买%d枚%s种子需要%d阳光, 您只有%d阳光. ",
				number, crop.Name, cost, player.Coins))
		return
	}
	current := player.CropCount[strconv.Itoa(crop.Level)]
	if current+number > 99 {
		p.sendText(replyTo, "一种种子持有量不能超过99枚")
		return
	}
	player.CropCount[strconv.Itoa(crop.Level)] = current + number
	player.Coins -= cost
	p.saveData()
	p.sendText(replyTo,
		fmt.Sprintf("购买成功\n\n%s ↑ %d => %d\n%s ↓ %d => %d",
			crop.FruitEmoji, number, current+number, emojiSun, cost, player.Coins))
}

func (p *FarmPlugin) buyPet(chatroomID, userID string, pet *Pet, replyTo *contact.Contact) {
	player := p.getOrCreatePlayer(chatroomID, userID)
	if player.Coins < int64(pet.Price) {
		p.sendText(replyTo,
			fmt.Sprintf("您的阳光不足, 购买%s需要%d阳光, 您只有%d阳光. ",
				pet.Name, pet.Price, player.Coins))
		return
	}
	for _, owned := range player.Pets {
		if owned == pet.Level {
			p.sendText(replyTo, "您已经有了该守卫")
			return
		}
	}
	player.Pets = append(player.Pets, pet.Level)
	player.Coins -= int64(pet.Price)
	p.saveData()
	p.sendText(replyTo,
		fmt.Sprintf("购买成功\n\n%s %s\n%s ↓ %d => %d",
			emojiDog, pet.Name, emojiSun, pet.Price, player.Coins))
}

func (p *FarmPlugin) buyField(chatroomID, userID string, replyTo *contact.Contact) {
	player := p.getOrCreatePlayer(chatroomID, userID)
	price := fieldPrice(player.Fields)
	if player.Coins >= price {
		player.Coins -= price
		player.Fields++
		p.saveData()
		p.sendText(replyTo,
			fmt.Sprintf("购买成功 土地+1\n%s ↓ %d => %d", emojiSun, price, player.Coins))
	} else {
		p.sendText(replyTo,
			fmt.Sprintf("购买第%d块土地需要%s%d", player.Fields+1, emojiSun, price))
	}
}

func (p *FarmPlugin) search(chatroomID, userID, name string, replyTo *contact.Contact) {
	if name == "" {
		p.printHelpSearch(replyTo)
		return
	}
	crop := cropByName[name]
	if crop != nil {
		totalHours := 0
		for _, h := range crop.StepHours {
			totalHours += h
		}
		p.sendText(replyTo,
			fmt.Sprintf("%s　%s, %d级别作物, 种子售价%s%d, 成熟时间%d小时. 每株结出果实%d到%d枚, 预计最少收益%s%d+%s%d。",
				crop.FruitEmoji, crop.Name, crop.Level, emojiSun, crop.SeedPrice, totalHours,
				crop.FruitsMin, crop.FruitsMax,
				emojiSun, crop.FruitsMin*crop.FruitPrice,
				emojiExp, crop.FruitsMin*crop.FruitExp))
		return
	}
	pet := petByName[name]
	if pet != nil {
		p.sendText(replyTo,
			fmt.Sprintf("%s %s, 价格%s%d, 防止被偷, 打盹时触发减半", emojiDog, pet.Name, emojiSun, pet.Price))
		return
	}
	p.sendText(replyTo, "没有找到该物品，请使用“农场商店/守卫商店”查看")
}
