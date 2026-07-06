package main

import "log/slog"

// OnLoad 插件加载时调用
func (p *RobberyPlugin) OnLoad() error {
	// 初始化内部状态
	p.data = make(map[string]map[string]*PlayerData)

	// 加载持久化数据
	p.loadData()

	slog.Info("[robbery] 插件加载完成",
		"data_file", p.Config.DataFile,
		"initial_money", p.Config.InitialMoney,
		"initial_strength", p.Config.InitialStrength,
		"groups", len(p.data),
	)
	return nil
}

// OnUnload 插件卸载时调用
func (p *RobberyPlugin) OnUnload() error {
	slog.Info("[robbery] 插件卸载中...")
	p.mu.Lock()
	p.data = nil
	p.mu.Unlock()
	slog.Info("[robbery] 插件卸载完成")
	return nil
}

// OnEnable 插件启用时调用
// 注意：必须实现，否则不满足 plugin.Lifecycle 接口，OnLoad 不会被调用
func (p *RobberyPlugin) OnEnable() error {
	slog.Info("[robbery] 插件已启用")
	return nil
}

// OnDisable 插件禁用时调用
func (p *RobberyPlugin) OnDisable() error {
	slog.Info("[robbery] 插件已禁用")
	return nil
}
