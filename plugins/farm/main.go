package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sbgayhub/golem/sdk/cdn"
	"github.com/sbgayhub/golem/sdk/chatroom"
	"github.com/sbgayhub/golem/sdk/contact"
	"github.com/sbgayhub/golem/sdk/message"
	"github.com/sbgayhub/golem/sdk/plugin"
)

// Config 插件配置
type Config struct {
	DataFile      string `toml:"data_file" comment:"游戏数据文件路径"`
	ImageDir      string `toml:"image_dir" comment:"农场图片目录"`
	InitialCoins  int64  `toml:"initial_coins" comment:"初始阳光数量"`
	InitialFields int    `toml:"initial_fields" comment:"初始土地数量"`
}

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

const (
	emojiSun   = "🔆"
	emojiStock = "🏕"
	emojiExp   = "📒"
	emojiLevel = "🔰"
	emojiField = "📜"
	emojiWater = "💦"
	emojiRain  = "🌧️"
	emojiDog   = "🐕"

	mature = -1
)

var (
	menuImagePath      = "菜单_农场.jpg"
	unplantedImagePath = "植物_未耕.jpg"
	plantedImagePath   = "植物_已耕.jpg"
	growthImagePaths   = []string{
		"植物一_1.png",
		"植物一_2.png",
		"植物一_3.png",
		"植物一_4.png",
		"植物一_5.png",
	}
)

// Crop 作物定义
type Crop struct {
	Level      int      `json:"level"`
	Name       string   `json:"name"`
	SeedPrice  int      `json:"seedPrice"`
	FruitsMin  int      `json:"fruitsMin"`
	FruitsMax  int      `json:"fruitsMax"`
	FruitPrice int      `json:"fruitPrice"`
	FruitExp   int      `json:"fruitExp"`
	StepHours  []int    `json:"stepHours"`
	StepEmojis []string `json:"stepEmojis"`
	FruitEmoji string   `json:"fruitEmoji"`
}

// RandomFruitCount 随机果实数量
func (c *Crop) RandomFruitCount(r *rand.Rand) int {
	if c.FruitsMax <= c.FruitsMin {
		return c.FruitsMax
	}
	return r.Intn(c.FruitsMax-c.FruitsMin+1) + c.FruitsMin
}

// TotalGrowSeconds 总生长秒数
func (c *Crop) TotalGrowSeconds() int64 {
	var total int64
	for _, h := range c.StepHours {
		total += int64(h) * 3600
	}
	return total
}

// Pet 守卫定义
type Pet struct {
	Level int    `json:"level"`
	Name  string `json:"name"`
	Price int    `json:"price"`
}

var crops = []*Crop{
	{1, "土豆", 10, 8, 12, 4, 4, []int{1, 2, 3}, []string{"🌱", "🌱", "🎍"}, "🥔"},
	{2, "萝卜", 20, 10, 15, 8, 4, []int{1, 2, 3}, []string{"🌱", "🎍", "🎍"}, "🥕"},
	{3, "花生", 30, 15, 17, 8, 4, []int{1, 3, 4}, []string{"🌱", "🎍", "🌿"}, "🥜"},
	{4, "番茄", 40, 10, 15, 20, 9, []int{1, 3, 4}, []string{"🌱", "🎍", "🌿"}, "🍅"},
	{5, "茄子", 50, 10, 15, 25, 12, []int{2, 4, 5}, []string{"🌱", "🎍", "🌿"}, "🍆"},
	{6, "辣椒", 120, 20, 25, 25, 12, []int{2, 4, 5}, []string{"🌱", "🎍", "🌾"}, "🌶"},
	{7, "蘑菇", 140, 25, 30, 25, 12, []int{2, 4, 6}, []string{"🌱", "🎍", "🌾"}, "🍄"},
	{8, "玉米", 160, 30, 35, 50, 20, []int{2, 4, 6}, []string{"🌱", "🎍", "🌾"}, "🌽"},
	{11, "苹果", 220, 30, 35, 60, 30, []int{3, 6, 8}, []string{"🌱", "🎍", "🌳"}, "🍎"},
	{13, "雪梨", 260, 30, 35, 70, 30, []int{3, 6, 8}, []string{"🌱", "🎍", "🌳"}, "🍐"},
	{15, "桃子", 300, 30, 35, 100, 70, []int{3, 6, 8}, []string{"🌱", "🎍", "🌳"}, "🍑"},
	{17, "橙子", 510, 30, 35, 150, 100, []int{3, 6, 8}, []string{"🌱", "🎍", "🌳"}, "🍊"},
	{19, "柠檬", 999, 30, 35, 200, 150, []int{3, 6, 8}, []string{"🌱", "🎍", "🌳"}, "🍋"},
}

var pets = []*Pet{
	{1, "斗牛犬", 10000},
}

var (
	cropByLevel = make(map[int]*Crop)
	cropByName  = make(map[string]*Crop)
	petByLevel  = make(map[int]*Pet)
	petByName   = make(map[string]*Pet)
)

func init() {
	for _, c := range crops {
		cropByLevel[c.Level] = c
		cropByName[c.Name] = c
	}
	for _, p := range pets {
		petByLevel[p.Level] = p
		petByName[p.Name] = p
	}
}

// FarmPlayer 玩家数据
type FarmPlayer struct {
	UserID      string            `json:"userId"`
	DisplayName string            `json:"displayName"`
	Coins       int64             `json:"coins"`
	Exp         int64             `json:"exp"`
	Fields      int               `json:"fields"`
	Ponds       int               `json:"ponds"`
	CropCount   map[string]int    `json:"cropCount"`
	LandFields  map[string]*Field `json:"landFields"`
	Pets        []int             `json:"pets"`
}

// Field 土地作物
type Field struct {
	Level     int               `json:"level"`
	PlantTime int64             `json:"plantTime"`
	Watered   map[string]string `json:"watered"`
	Stealer   []string          `json:"stealer"`
	Alerted   []string          `json:"alerted"`
}

// CropState 作物状态
type CropState struct {
	State int
	Emoji string
}

// BuyRequest 购买请求
type BuyRequest struct {
	Name   string
	Number int
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
	p.ensureDefaults()
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

// OnEvent 处理事件
func (p *FarmPlugin) OnEvent(e *plugin.Event) (bool, error) {
	p.ensureDefaults()

	msg := e.Payload.(*plugin.Event_Message).Message
	if msg == nil {
		return false, nil
	}

	text := strings.TrimSpace(msg.GetContent())
	if text == "" {
		if td := msg.GetText(); td != nil {
			text = strings.TrimSpace(td.Content)
		}
	}
	if text == "" {
		return false, nil
	}

	receiverType := contact.ContactType(0)
	if msg.Receiver != nil {
		receiverType = msg.Receiver.GetType()
	}
	senderID := ""
	if msg.Sender != nil {
		senderID = msg.Sender.GetUsername()
	}
	slog.Info("[farm] 收到消息", "text", text, "receiver_type", receiverType, "sender", senderID)

	// 非群聊只允许 "农场" 提示
	if !p.isGroupChat(msg) {
		if text == "农场" {
			p.sendText(msg.Sender, "农场功能只能在群中使用")
		}
		return false, nil
	}

	// 只处理农场相关命令
	if !p.isFarmCommand(text) {
		return false, nil
	}

	chatroomID := p.getChatroomID(msg)
	userID := p.getUserID(e, msg)
	replyTo := p.getReplyTo(msg)
	if chatroomID == "" || userID == "" || replyTo == nil {
		slog.Warn("[farm] 无法确定群聊或用户", "chatroom_id", chatroomID, "user_id", userID)
		return false, nil
	}
	slog.Info("[farm] 处理群聊命令", "chatroom_id", chatroomID, "user_id", userID, "text", text)

	defer func() {
		if r := recover(); r != nil {
			slog.Error("[farm] 处理命令时 panic", "err", r)
			p.sendText(replyTo, "农场出错了，请稍后再试")
		}
	}()

	switch text {
	case "农场":
		p.printMenu(replyTo)
		p.sendImageIfAvailable(replyTo, menuImagePath)
	case "农场帮助":
		p.printHelp(replyTo)
	case "农场商店":
		p.printCrops(chatroomID, userID, replyTo)
	case "守卫商店":
		p.printPets(chatroomID, userID, replyTo)
	case "农场购买种子", "农场购买守卫":
		p.printHelpBuy(replyTo)
	case "查询种子", "查询守卫":
		p.printHelpSearch(replyTo)
	case "种植":
		p.printHelpPlant(replyTo)
	case "偷菜":
		p.printHelpSteal(replyTo)
	case "收菜":
		p.collect(chatroomID, userID, replyTo)
	case "浇水":
		p.water(chatroomID, userID, "", msg, replyTo)
	case "我的农场":
		p.printSelf(chatroomID, userID, replyTo)
	case "农场等级":
		p.printLevels(chatroomID, userID, replyTo)
	case "购买土地":
		p.buyField(chatroomID, userID, replyTo)
	default:
		if strings.HasPrefix(text, "查询") {
			name := strings.TrimSpace(text[6:])
			p.search(chatroomID, userID, name, replyTo)
		} else if strings.HasPrefix(text, "农场购买") || strings.HasPrefix(text, "农场买") {
			normalized := p.normalizeFarmBuyContent(text)
			req := p.parseBuyRequest(normalized)
			if req == nil {
				p.sendText(replyTo, "格式错误！请使用：农场购买+名称(+数量)")
				return true, nil
			}
			p.buy(chatroomID, userID, req, replyTo)
		} else if strings.HasPrefix(text, "种植") || strings.HasPrefix(text, "播种") || strings.HasPrefix(text, "种") {
			name := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(strings.TrimPrefix(text, "种植"), "播种"), "种"))
			if name == "" {
				p.printHelpPlant(replyTo)
			} else {
				p.plant(chatroomID, userID, name, replyTo)
			}
		} else if strings.HasPrefix(text, "偷菜") {
			name := strings.TrimSpace(text[6:])
			p.steal(chatroomID, userID, name, msg, replyTo)
		} else if strings.HasPrefix(text, "浇水") {
			name := strings.TrimSpace(text[6:])
			p.water(chatroomID, userID, name, msg, replyTo)
		}
	}

	return true, nil
}

func (p *FarmPlugin) ensureDefaults() {
	if p.Config.DataFile == "" {
		p.Config.DataFile = "data/farm_game.json"
	}
	if p.Config.ImageDir == "" {
		p.Config.ImageDir = "农场图片"
	}
	if p.Config.InitialCoins == 0 {
		p.Config.InitialCoins = 3000
	}
	if p.Config.InitialFields == 0 {
		p.Config.InitialFields = 1
	}
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

func (p *FarmPlugin) isGroupChat(msg *message.Message) bool {
	return p.getChatroomID(msg) != ""
}

// getChatroomID 从消息中解析群聊 ID。
// SDK 中 msg.Member 仅群消息有效；同时兼容 Receiver.Type == CHATROOM
// 以及 Sender 用户名以 @chatroom 结尾的两种消息模型。
func (p *FarmPlugin) getChatroomID(msg *message.Message) string {
	if msg.Receiver != nil && msg.Receiver.Type == contact.ContactType_CONTACT_TYPE_CHATROOM {
		return msg.Receiver.GetUsername()
	}
	if msg.Sender != nil && strings.HasSuffix(msg.Sender.GetUsername(), "@chatroom") {
		return msg.Sender.GetUsername()
	}
	return ""
}

// getUserID 获取真实发送者 ID。
// 群聊场景下优先使用 msg.Member（仅群消息有效，表示群内发言成员），
// 其次使用 plugin.Event.Sender，最后回退到 msg.Sender。
func (p *FarmPlugin) getUserID(e *plugin.Event, msg *message.Message) string {
	if msg.Member != nil && msg.Member.GetUsername() != "" {
		return msg.Member.GetUsername()
	}
	if e.GetSender() != "" && !strings.HasSuffix(e.GetSender(), "@chatroom") {
		return e.GetSender()
	}
	if msg.Sender != nil && !strings.HasSuffix(msg.Sender.GetUsername(), "@chatroom") {
		return msg.Sender.GetUsername()
	}
	return ""
}

// getReplyTo 返回用于回复群消息的 contact。
func (p *FarmPlugin) getReplyTo(msg *message.Message) *contact.Contact {
	if msg.Receiver != nil && msg.Receiver.Type == contact.ContactType_CONTACT_TYPE_CHATROOM {
		return msg.Receiver
	}
	if msg.Sender != nil && strings.HasSuffix(msg.Sender.GetUsername(), "@chatroom") {
		return msg.Sender
	}
	return nil
}

func (p *FarmPlugin) isFarmCommand(content string) bool {
	commands := []string{
		"农场帮助", "农场商店", "守卫商店", "农场购买种子", "农场购买守卫",
		"查询种子", "查询守卫", "种植", "偷菜", "收菜", "浇水", "我的农场",
		"农场等级", "购买土地",
	}
	for _, cmd := range commands {
		if content == cmd {
			return true
		}
	}
	prefixes := []string{"查询", "农场购买", "农场买", "种植", "播种", "种", "偷菜", "浇水"}
	for _, prefix := range prefixes {
		if strings.HasPrefix(content, prefix) {
			return true
		}
	}
	return content == "农场"
}

func (p *FarmPlugin) normalizeFarmBuyContent(content string) string {
	content = strings.TrimPrefix(content, "农场购买")
	content = strings.TrimPrefix(content, "农场买")
	return strings.TrimSpace(content)
}

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

func now() int64 {
	return time.Now().Unix()
}

func computeLevel(exp int64) int {
	for i := 21; i > 0; i-- {
		need := (int64(math.Pow(float64(i), 4)) - 1) / 5
		if exp >= need {
			return i
		}
	}
	return 0
}

func fieldPrice(currentFieldCount int) int64 {
	base := float64(currentFieldCount)
	return int64(math.Pow(2.5, 0.75*base)*base) * 1000
}

func contains(list []string, item string) bool {
	for _, s := range list {
		if s == item {
			return true
		}
	}
	return false
}

func targetUserIDHash(s string) int {
	h := 0
	for _, c := range s {
		h = 31*h + int(c)
	}
	return h
}

func (p *FarmPlugin) randomInt64(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}

func main() {
	p := &FarmPlugin{
		ConfigAbility: plugin.ConfigAbility[Config]{
			Config: Config{
				DataFile:      "data/farm_game.json",
				ImageDir:      "农场图片",
				InitialCoins:  3000,
				InitialFields: 1,
			},
		},
	}
	slog.Info("[farm] 农场插件启动中...")
	plugin.Start(p)
}
