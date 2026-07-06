package main

import (
	"log/slog"

	"github.com/sbgayhub/golem/sdk/plugin"
)

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
