package main

import (
	"math/rand"
)

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
