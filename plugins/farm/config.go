package main

// Config 插件配置
type Config struct {
	DataFile      string `toml:"data_file" comment:"游戏数据文件路径"`
	ImageDir      string `toml:"image_dir" comment:"农场图片目录"`
	InitialCoins  int64  `toml:"initial_coins" comment:"初始阳光数量"`
	InitialFields int    `toml:"initial_fields" comment:"初始土地数量"`
}
