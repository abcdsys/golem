package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/robfig/cron/v3"
	"github.com/sbgayhub/golem/sdk/plugin"
)

type cronAddCommand struct {
	_          struct{} `cmd:"cron add" help:"创建定时任务" usage:"/cron add -c <cron> -p <capability> -t <targets> [-a args]" example:"/cron add -c \"0 9 * * *\" -p news.today -t wxid_xxx\n/cron add -c \"*/5 * * * *\" -p text.to.image -t wxid_xxx -a \"context=hello bg_color=#fff\""`
	Cron       string   `flag:"c,cron" help:"cron 表达式，包含空格时需要加引号" required:"true"`
	Capability string   `flag:"p,capability" help:"要调用的能力名称" required:"true"`
	Targets    string   `flag:"t,targets" help:"接收者 username，多个用英文逗号分隔" required:"true"`
	Args       string   `flag:"a,args" help:"调用参数，支持 key=value 列表或 JSON 对象"`
}

type cronDeleteCommand struct {
	_  struct{} `cmd:"cron delete" help:"删除定时任务" usage:"/cron delete -i <id>" example:"/cron delete -i 1"`
	ID int      `flag:"i,id" help:"/cron list 展示的任务序号" required:"true"`
}

type cronListCommand struct {
	_ struct{} `cmd:"cron list" help:"列出定时任务" usage:"/cron list" example:"/cron list"`
}

type cronUpdateCommand struct {
	_          struct{} `cmd:"cron update" help:"更新定时任务" usage:"/cron update -i <id> [-c <cron>] [-p <capability>] [-t <targets>] [-a args]" example:"/cron update -i 1 -c \"0 10 * * *\"\n/cron update -i 1 -p news.today -t wxid_xxx\n/cron update -i 1 -a \"\""`
	ID         int      `flag:"i,id" help:"/cron list 展示的任务序号" required:"true"`
	Cron       *string  `flag:"c,cron" help:"cron 表达式，包含空格时需要加引号"`
	Capability *string  `flag:"p,capability" help:"要调用的能力名称"`
	Targets    *string  `flag:"t,targets" help:"接收者 username，多个用英文逗号分隔"`
	Args       *string  `flag:"a,args" help:"调用参数，支持 key=value 列表或 JSON 对象，传空字符串可清空参数"`
}

func newCronPlugin() (*CronPlugin, error) {
	p := &CronPlugin{
		ConfigAbility: plugin.ConfigAbility[CronConfig]{
			Config: CronConfig{},
		},
		cron: cron.New(),
	}
	if err := registerCronCommands(p); err != nil {
		return nil, err
	}
	return p, nil
}

func registerCronCommands(p *CronPlugin) error {
	if err := plugin.RegisterCommand(p.add); err != nil {
		return err
	}
	if err := plugin.RegisterCommand(p.delete); err != nil {
		return err
	}
	if err := plugin.RegisterCommand(p.list); err != nil {
		return err
	}
	return plugin.RegisterCommand(p.update)
}

func (c *CronPlugin) GetCommands() []string {
	return plugin.CommandCommands()
}

func (c *CronPlugin) GetCommandSchemas() []*plugin.CommandSchema {
	return plugin.CommandSchemas()
}

func (c *CronPlugin) OnCommand(command *plugin.Command) (string, error) {
	return plugin.DispatchCommand(command)
}

func (c *CronPlugin) add(cmd cronAddCommand) (string, error) {
	config, err := configFromAddCommand(cmd)
	if err != nil {
		return "", err
	}

	id, err := c.addConfig(config)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("定时任务已创建：id=%d \ncron=%q \ncapability=%s \ntargets=%s", id, config.Cron, config.Capability, strings.Join(config.Target, ",")), nil
}

func (c *CronPlugin) delete(cmd cronDeleteCommand) (string, error) {
	if err := c.deleteConfig(cmd.ID); err != nil {
		return "", err
	}
	return fmt.Sprintf("定时任务已删除：id=%d", cmd.ID), nil
}

func (c *CronPlugin) list(_ cronListCommand) (string, error) {
	if len(c.Config.Jobs) == 0 {
		return "当前没有定时任务", nil
	}

	lines := make([]string, 0, len(c.Config.Jobs)+1)
	lines = append(lines, "定时任务：")
	for i, config := range c.Config.Jobs {
		lines = append(lines, fmt.Sprintf("%d. cron=%q \ncapability=%s \ntargets=%s \nargs=%d", i+1, config.Cron, config.Capability, strings.Join(config.Target, ","), len(config.Args)))
	}
	return strings.Join(lines, "\n"), nil
}

func (c *CronPlugin) update(cmd cronUpdateCommand) (string, error) {
	index, err := c.configIndex(cmd.ID)
	if err != nil {
		return "", err
	}

	config, err := configFromUpdateCommand(c.Config.Jobs[index], cmd)
	if err != nil {
		return "", err
	}
	if err := c.replaceConfig(index, config); err != nil {
		return "", err
	}
	return fmt.Sprintf("定时任务已更新：id=%d \ncron=%q \ncapability=%s \ntargets=%s", cmd.ID, config.Cron, config.Capability, strings.Join(config.Target, ",")), nil
}

func (c *CronPlugin) addConfig(config Config) (int, error) {
	entryID, err := c.scheduleConfig(config)
	if err != nil {
		return 0, fmt.Errorf("创建定时任务失败: %w", err)
	}

	c.Config.Jobs = append(c.Config.Jobs, config)
	c.entries = append(c.entries, entryID)
	if err := c.SaveConfig(c); err != nil {
		c.cron.Remove(entryID)
		c.Config.Jobs = c.Config.Jobs[:len(c.Config.Jobs)-1]
		c.entries = c.entries[:len(c.entries)-1]
		return 0, fmt.Errorf("保存定时任务失败: %w", err)
	}
	return len(c.Config.Jobs), nil
}

func (c *CronPlugin) deleteConfig(id int) error {
	index, err := c.configIndex(id)
	if err != nil {
		return err
	}

	oldConfig := c.Config.Jobs[index]
	oldEntry := c.entries[index]
	c.removeEntry(index)
	c.Config.Jobs = append(c.Config.Jobs[:index], c.Config.Jobs[index+1:]...)
	c.entries = append(c.entries[:index], c.entries[index+1:]...)

	if err := c.SaveConfig(c); err != nil {
		c.Config.Jobs = append(c.Config.Jobs[:index], append([]Config{oldConfig}, c.Config.Jobs[index:]...)...)
		c.entries = append(c.entries[:index], append([]cron.EntryID{0}, c.entries[index:]...)...)
		if oldEntry != 0 {
			if entryID, scheduleErr := c.scheduleConfig(oldConfig); scheduleErr == nil {
				c.entries[index] = entryID
			}
		}
		return fmt.Errorf("保存定时任务失败: %w", err)
	}
	return nil
}

func (c *CronPlugin) replaceConfig(index int, config Config) error {
	entryID, err := c.scheduleConfig(config)
	if err != nil {
		return fmt.Errorf("创建定时任务失败: %w", err)
	}

	oldConfig := c.Config.Jobs[index]
	c.removeEntry(index)
	c.Config.Jobs[index] = config
	c.entries[index] = entryID
	if err := c.SaveConfig(c); err != nil {
		c.cron.Remove(entryID)
		c.Config.Jobs[index] = oldConfig
		c.entries[index] = 0
		if oldEntryID, scheduleErr := c.scheduleConfig(oldConfig); scheduleErr == nil {
			c.entries[index] = oldEntryID
		}
		return fmt.Errorf("保存定时任务失败: %w", err)
	}
	return nil
}

func (c *CronPlugin) configIndex(id int) (int, error) {
	if id <= 0 || id > len(c.Config.Jobs) {
		return 0, fmt.Errorf("定时任务不存在：id=%d", id)
	}
	c.ensureCron()
	return id - 1, nil
}

func (c *CronPlugin) removeEntry(index int) {
	if index < 0 || index >= len(c.entries) {
		return
	}
	if entryID := c.entries[index]; entryID != 0 {
		c.cron.Remove(entryID)
	}
	c.entries[index] = 0
}

func configFromAddCommand(cmd cronAddCommand) (Config, error) {
	return configFromParts(cmd.Cron, cmd.Capability, cmd.Targets, cmd.Args)
}

func configFromUpdateCommand(current Config, cmd cronUpdateCommand) (Config, error) {
	config := current
	if cmd.Cron != nil {
		schedule := strings.TrimSpace(*cmd.Cron)
		if schedule == "" {
			return Config{}, fmt.Errorf("cron 表达式不能为空")
		}
		config.Cron = schedule
	}
	if cmd.Capability != nil {
		capability := strings.TrimSpace(*cmd.Capability)
		if capability == "" {
			return Config{}, fmt.Errorf("能力名称不能为空")
		}
		config.Capability = capability
	}
	if cmd.Targets != nil {
		targets, err := parseTargets(*cmd.Targets)
		if err != nil {
			return Config{}, err
		}
		config.Target = targets
	}
	if cmd.Args != nil {
		args, err := parseCronArgs(*cmd.Args)
		if err != nil {
			return Config{}, err
		}
		config.Args = args
	}
	return config, nil
}

func configFromParts(rawCron, rawCapability, rawTargets, rawArgs string) (Config, error) {
	schedule := strings.TrimSpace(rawCron)
	if schedule == "" {
		return Config{}, fmt.Errorf("cron 表达式不能为空")
	}

	capability := strings.TrimSpace(rawCapability)
	if capability == "" {
		return Config{}, fmt.Errorf("能力名称不能为空")
	}

	targets, err := parseTargets(rawTargets)
	if err != nil {
		return Config{}, err
	}

	args, err := parseCronArgs(rawArgs)
	if err != nil {
		return Config{}, err
	}

	return Config{
		Cron:       schedule,
		Target:     targets,
		Capability: capability,
		Args:       args,
	}, nil
}

func parseTargets(raw string) ([]string, error) {
	parts := strings.Split(raw, ",")
	targets := make([]string, 0, len(parts))
	for _, part := range parts {
		target := strings.TrimSpace(part)
		if target != "" {
			targets = append(targets, target)
		}
	}
	if len(targets) == 0 {
		return nil, fmt.Errorf("接收者不能为空")
	}
	return targets, nil
}

func parseCronArgs(raw string) (map[string]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	args := map[string]string{}
	if strings.HasPrefix(raw, "{") {
		if err := json.Unmarshal([]byte(raw), &args); err != nil {
			return nil, fmt.Errorf("解析 JSON 参数失败: %w", err)
		}
		return args, nil
	}

	for _, field := range strings.Fields(raw) {
		key, value, ok := strings.Cut(field, "=")
		if !ok {
			return nil, fmt.Errorf("参数格式错误: %s", field)
		}
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, fmt.Errorf("参数名不能为空")
		}
		args[key] = value
	}
	return args, nil
}
