package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sbgayhub/golem/sdk/plugin"
)

type aiGetCommand struct {
	_ struct{} `cmd:"ai get" help:"查看 AI 配置" usage:"/ai get" example:"/ai get"`
}

type aiSetCommand struct {
	_          struct{} `cmd:"ai set" help:"更新 AI 配置" usage:"/ai set [-u url] [-k api_key] [-m model] [-r reply_rate] [--max-context n] [--timeout n]" example:"/ai set -u https://api.deepseek.com/v1 -k sk-xxx -m deepseek-chat\n/ai set -r 0.2"`
	BaseURL    *string  `flag:"u,url" help:"OpenAI 兼容接口地址，例如 https://api.openai.com/v1"`
	APIKey     *string  `flag:"k,api-key" help:"接口密钥"`
	Model      *string  `flag:"m,model" help:"模型名称"`
	ReplyRate  *float64 `flag:"r,reply-rate" help:"普通消息回复概率，取值 0~1"`
	MaxContext *int     `flag:"max-context" help:"每个会话最多保留的上下文消息数"`
	Timeout    *int     `flag:"timeout" help:"大模型请求超时时间，单位秒"`
}

type aiPromptCommand struct {
	_      struct{} `cmd:"ai prompt" help:"切换或设置提示词" usage:"/ai prompt <name> [-p prompt]" example:"/ai prompt default\n/ai prompt roleplay -p \"你是一个微信聊天机器人\""`
	Name   string   `arg:"name" help:"提示词名称" required:"true"`
	Prompt *string  `flag:"p,prompt" help:"提示词内容；提供后会新增或更新该提示词并切换到它"`
}

type aiClearContextCommand struct {
	_       struct{} `cmd:"ai clear-context" help:"清理上下文" usage:"/ai clear-context [-t target]" example:"/ai clear-context\n/ai clear-context -t chatroom::123@chatroom"`
	Target  string   `flag:"t,target" help:"会话 key；不传则清理当前命令来源会话"`
	Command *plugin.Command
}

type aiHelpCommand struct {
	_ struct{} `cmd:"ai help" help:"显示 AI 插件帮助" usage:"/ai help" example:"/ai help"`
}

func registerCommands(p *AiPlugin) error {
	handlers := []func() error{
		func() error { return plugin.RegisterCommand(p.handleGet) },
		func() error { return plugin.RegisterCommand(p.handleSet) },
		func() error { return plugin.RegisterCommand(p.handlePrompt) },
		func() error { return plugin.RegisterCommand(p.handleClearContext) },
		func() error { return plugin.RegisterCommand(p.handleHelp) },
	}
	for _, register := range handlers {
		if err := register(); err != nil {
			return err
		}
	}
	return nil
}

func (p *AiPlugin) handleGet(aiGetCommand) (string, error) {
	config := p.configSnapshot()
	return strings.Join([]string{
		"AI 配置：",
		fmt.Sprintf("url=%s", emptyDash(config.BaseURL)),
		fmt.Sprintf("api_key=%s", maskSecret(config.APIKey)),
		fmt.Sprintf("model=%s", emptyDash(config.Model)),
		fmt.Sprintf("active_prompt=%s", config.ActivePrompt),
		fmt.Sprintf("prompts=%s", strings.Join(promptNames(config.Prompts), ",")),
		fmt.Sprintf("reply_rate=%.4g", config.ReplyRate),
		fmt.Sprintf("max_context=%d", config.MaxContextMessages),
		fmt.Sprintf("timeout=%d", config.HTTPTimeoutSeconds),
		fmt.Sprintf("prompt=%s", emptyDash(activePromptContent(config))),
	}, "\n"), nil
}

func (p *AiPlugin) handleSet(cmd aiSetCommand) (string, error) {
	p.configMu.Lock()
	defer p.configMu.Unlock()

	if err := p.applySetCommandLocked(cmd); err != nil {
		return "", err
	}
	if err := p.SaveConfig(p); err != nil {
		return "", fmt.Errorf("保存 AI 配置失败: %w", err)
	}
	return "AI 配置已更新", nil
}

func (p *AiPlugin) applySetCommand(cmd aiSetCommand) error {
	p.configMu.Lock()
	defer p.configMu.Unlock()

	return p.applySetCommandLocked(cmd)
}

func (p *AiPlugin) applySetCommandLocked(cmd aiSetCommand) error {
	changed := false
	if cmd.BaseURL != nil {
		p.Config.BaseURL = strings.TrimSpace(*cmd.BaseURL)
		changed = true
	}
	if cmd.APIKey != nil {
		p.Config.APIKey = strings.TrimSpace(*cmd.APIKey)
		changed = true
	}
	if cmd.Model != nil {
		p.Config.Model = strings.TrimSpace(*cmd.Model)
		changed = true
	}
	if cmd.ReplyRate != nil {
		if *cmd.ReplyRate < 0 || *cmd.ReplyRate > 1 {
			return fmt.Errorf("reply_rate 必须在 0~1 之间")
		}
		p.Config.ReplyRate = *cmd.ReplyRate
		changed = true
	}
	if cmd.MaxContext != nil {
		if *cmd.MaxContext <= 0 {
			return fmt.Errorf("max_context 必须大于 0")
		}
		p.Config.MaxContextMessages = *cmd.MaxContext
		changed = true
	}
	if cmd.Timeout != nil {
		if *cmd.Timeout <= 0 {
			return fmt.Errorf("timeout 必须大于 0")
		}
		p.Config.HTTPTimeoutSeconds = *cmd.Timeout
		changed = true
	}
	if !changed {
		return fmt.Errorf("未提供要更新的配置")
	}
	p.Config = normalizeConfigValue(p.Config)
	return nil
}

func (p *AiPlugin) handlePrompt(cmd aiPromptCommand) (string, error) {
	p.configMu.Lock()
	defer p.configMu.Unlock()

	updated, err := p.applyPromptCommandLocked(cmd)
	if err != nil {
		return "", err
	}
	if err := p.SaveConfig(p); err != nil {
		return "", fmt.Errorf("保存 AI 配置失败: %w", err)
	}
	if updated {
		return fmt.Sprintf("提示词已保存并切换：%s", p.Config.ActivePrompt), nil
	}
	return fmt.Sprintf("已切换提示词：%s", p.Config.ActivePrompt), nil
}

func (p *AiPlugin) applyPromptCommand(cmd aiPromptCommand) (bool, error) {
	p.configMu.Lock()
	defer p.configMu.Unlock()
	return p.applyPromptCommandLocked(cmd)
}

func (p *AiPlugin) applyPromptCommandLocked(cmd aiPromptCommand) (bool, error) {
	name := strings.TrimSpace(cmd.Name)
	if name == "" {
		return false, fmt.Errorf("提示词名称不能为空")
	}
	p.Config = normalizeConfigValue(p.Config)
	if cmd.Prompt == nil {
		if _, ok := p.Config.Prompts[name]; !ok {
			return false, fmt.Errorf("提示词不存在：%s", name)
		}
		p.Config.ActivePrompt = name
		return false, nil
	}
	p.Config.Prompts[name] = strings.TrimSpace(*cmd.Prompt)
	p.Config.ActivePrompt = name
	p.Config = normalizeConfigValue(p.Config)
	return true, nil
}

func (p *AiPlugin) handleClearContext(cmd aiClearContextCommand) (string, error) {
	key := strings.TrimSpace(cmd.Target)
	if key == "" {
		key = commandSessionKey(cmd.Command)
	}
	if key == "" {
		return "", fmt.Errorf("无法确定要清理的会话")
	}
	p.clearContext(key)
	return fmt.Sprintf("上下文已清理：%s", key), nil
}

func (p *AiPlugin) handleHelp(aiHelpCommand) (string, error) {
	return strings.Join([]string{
		"AI 插件",
		"",
		"命令：",
		"/ai get",
		"/ai set [-u url] [-k api_key] [-m model] [-r reply_rate] [--max-context n] [--timeout n]",
		"/ai prompt <name> [-p prompt]",
		"/ai clear-context [-t target]",
		"/ai help",
		"",
		"说明：reply_rate 取值 0~1；被 @ 登录账号昵称或引用机器人消息时必定回复，普通消息按概率回复。",
	}, "\n"), nil
}

func commandSessionKey(cmd *plugin.Command) string {
	if cmd == nil || cmd.GetSender() == nil {
		return ""
	}
	sender := cmd.GetSender()
	if sender.GetUsername() == "" {
		return ""
	}
	if sender.GetType() == contactTypeChatroom {
		return "chatroom:" + sender.GetUsername()
	}
	return "private:" + sender.GetUsername()
}

func maskSecret(secret string) string {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return "-"
	}
	if len(secret) <= 8 {
		return "****"
	}
	return secret[:4] + "****" + secret[len(secret)-4:]
}

func emptyDash(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "-"
	}
	return value
}

func promptNames(prompts map[string]string) []string {
	names := make([]string, 0, len(prompts))
	for name := range prompts {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
