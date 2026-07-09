package main

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/sbgayhub/golem/sdk/plugin"
)

type AddCommand struct {
	_                  struct{} `cmd:"universal add" help:"新增关键词规则" usage:"/universal add <id> -k <关键词1,关键词2> -u <URL模板> [选项]" example:"/universal add pixiv -k 来张图 -u \"https://api.example.com/image?width={{width}}&height={{height}}\" -t image -p data.url\n/universal add cp -k cp -u \"https://api.example.com/cp?a={{arg_1}}&b={{arg_2}}\" -t text --at true"`
	ID                 string   `arg:"id" help:"规则 ID，必须唯一" required:"true"`
	Keywords           string   `flag:"k,keywords" help:"关键词列表，逗号分隔；收到消息后第一段全文匹配" required:"true"`
	URL                string   `flag:"u,url" help:"请求地址模板，支持 {{text}}、{{keyword}}、{{param}}、{{arg_1}} 等变量" required:"true"`
	Method             string   `flag:"m,method" help:"请求方法，默认 GET"`
	Headers            string   `flag:"H,headers" help:"请求头，格式：A=B;C=D，支持模板变量"`
	Body               string   `flag:"b,body" help:"请求体模板，常用于 POST/PUT"`
	SendType           string   `flag:"t,send-type" help:"发送类型：text|emoji|image|video，默认 text"`
	ResultPath         string   `flag:"p,result-path" help:"gjson 路径；为空时直接使用响应 body 原文"`
	At                 bool     `flag:"at" value:"false" help:"是否在文本回复中 @ 参数或引用对应用户，默认 false"`
}

type UpdateCommand struct {
	_                  struct{} `cmd:"universal update" help:"更新关键词规则" usage:"/universal update <id> [选项]" example:"/universal update pixiv -u \"https://api.example.com/image?size={{arg_1}}\" -p data.url\n/universal update pixiv --at false"`
	ID                 string   `arg:"id" help:"规则 ID" required:"true"`
	Keywords           string   `flag:"k,keywords" help:"关键词列表，逗号分隔；提供后会整体替换"`
	URL                string   `flag:"u,url" help:"请求地址模板"`
	Method             string   `flag:"m,method" help:"请求方法"`
	Headers            string   `flag:"H,headers" help:"请求头，格式：A=B;C=D，支持模板变量"`
	Body               string   `flag:"b,body" help:"请求体模板"`
	SendType           string   `flag:"t,send-type" help:"发送类型：text|emoji|image|video"`
	ResultPath         string   `flag:"p,result-path" help:"gjson 路径；为空时保持不变"`
	At                 *bool    `flag:"at" value:"false" help:"是否在文本回复中 @ 参数或引用对应用户：true|false"`
	ClearHeaders       bool     `flag:"clear-headers" help:"清空请求头" value:"false"`
	ClearBody          bool     `flag:"clear-body" help:"清空请求体" value:"false"`
	ClearResultPath    bool     `flag:"clear-result-path" help:"清空结果路径，之后直接使用响应 body 原文" value:"false"`
}

type ListCommand struct {
	_ struct{} `cmd:"universal list" help:"查看全部规则" usage:"/universal list" example:"/universal list"`
}

type GetCommand struct {
	_  struct{} `cmd:"universal get" help:"查看规则详情" usage:"/universal get <id>" example:"/universal get pixiv"`
	ID string   `arg:"id" help:"规则 ID" required:"true"`
}

type DeleteCommand struct {
	_  struct{} `cmd:"universal delete" help:"删除规则" usage:"/universal delete <id>" example:"/universal delete pixiv"`
	ID string   `arg:"id" help:"规则 ID" required:"true"`
}

type EnableCommand struct {
	_  struct{} `cmd:"universal enable" help:"启用规则" usage:"/universal enable <id>" example:"/universal enable pixiv"`
	ID string   `arg:"id" help:"规则 ID" required:"true"`
}

type DisableCommand struct {
	_  struct{} `cmd:"universal disable" help:"禁用规则" usage:"/universal disable <id>" example:"/universal disable pixiv"`
	ID string   `arg:"id" help:"规则 ID" required:"true"`
}

type HelpCommand struct {
	_ struct{} `cmd:"universal help" help:"显示 Universal 插件帮助" usage:"/universal help" example:"/universal help"`
}

func (p *UniversalPlugin) registerCommands() error {
	handlers := []func() error{
		func() error { return plugin.RegisterCommand(p.handleAdd) },
		func() error { return plugin.RegisterCommand(p.handleUpdate) },
		func() error { return plugin.RegisterCommand(p.handleList) },
		func() error { return plugin.RegisterCommand(p.handleGet) },
		func() error { return plugin.RegisterCommand(p.handleDelete) },
		func() error { return plugin.RegisterCommand(p.handleEnable) },
		func() error { return plugin.RegisterCommand(p.handleDisable) },
		func() error { return plugin.RegisterCommand(p.handleHelp) },
	}
	for _, register := range handlers {
		if err := register(); err != nil {
			return err
		}
	}
	return nil
}

func (p *UniversalPlugin) handleAdd(cmd AddCommand) (string, error) {
	keywords, err := parseKeywords(cmd.Keywords)
	if err != nil {
		return "", err
	}

	rule := Rule{
		ID:         strings.TrimSpace(cmd.ID),
		Keywords:   keywords,
		URL:        strings.TrimSpace(cmd.URL),
		Method:     cmd.Method,
		Headers:    cmd.Headers,
		Body:       cmd.Body,
		SendType:   cmd.SendType,
		ResultPath: strings.TrimSpace(cmd.ResultPath),
		At:         cmd.At,
		Enabled:    new(true),
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.findRuleIndexLocked(rule.ID) >= 0 {
		return "", fmt.Errorf("规则已存在：%s", rule.ID)
	}
	previous := cloneRules(p.Config.Rules)
	p.Config.Rules = append(p.Config.Rules, rule)
	if err := p.rebuildAndSaveLocked(previous); err != nil {
		return "", err
	}
	return fmt.Sprintf("已新增规则：%s", rule.ID), nil
}

func (p *UniversalPlugin) handleUpdate(cmd UpdateCommand) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	index := p.findRuleIndexLocked(strings.TrimSpace(cmd.ID))
	if index < 0 {
		return "", fmt.Errorf("规则不存在：%s", cmd.ID)
	}

	previous := cloneRules(p.Config.Rules)
	changed := false
	rule := &p.Config.Rules[index]
	if cmd.Keywords != "" {
		keywords, err := parseKeywords(cmd.Keywords)
		if err != nil {
			return "", err
		}
		rule.Keywords = keywords
		changed = true
	}
	if cmd.URL != "" {
		rule.URL = strings.TrimSpace(cmd.URL)
		changed = true
	}
	if cmd.Method != "" {
		rule.Method = cmd.Method
		changed = true
	}
	if cmd.ClearHeaders {
		rule.Headers = ""
		changed = true
	}
	if cmd.Headers != "" {
		rule.Headers = cmd.Headers
		changed = true
	}
	if cmd.ClearBody {
		rule.Body = ""
		changed = true
	}
	if cmd.Body != "" {
		rule.Body = cmd.Body
		changed = true
	}
	if cmd.SendType != "" {
		rule.SendType = cmd.SendType
		changed = true
	}
	if cmd.ClearResultPath {
		rule.ResultPath = ""
		changed = true
	}
	if cmd.ResultPath != "" {
		rule.ResultPath = strings.TrimSpace(cmd.ResultPath)
		changed = true
	}
	if cmd.At != nil {
		rule.At = *cmd.At
		changed = true
	}
	if !changed {
		return "", errors.New("未提供要更新的参数")
	}

	if err := p.rebuildAndSaveLocked(previous); err != nil {
		return "", err
	}
	return fmt.Sprintf("已更新规则：%s", rule.ID), nil
}

func (p *UniversalPlugin) handleList(ListCommand) (string, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(p.Config.Rules) == 0 {
		return "暂无规则", nil
	}

	rules := append([]Rule(nil), p.Config.Rules...)
	sort.Slice(rules, func(i, j int) bool {
		return rules[i].ID < rules[j].ID
	})

	lines := []string{"Universal 规则列表："}
	for _, rule := range rules {
		lines = append(lines, fmt.Sprintf("- %s [%s] keywords=%s method=%s send_type=%s at=%t",
			rule.ID,
			ruleStatus(&rule),
			strings.Join(rule.Keywords, ","),
			normalizeMethod(rule.Method),
			normalizeSendType(rule.SendType),
			rule.At,
		))
	}
	return strings.Join(lines, "\n"), nil
}

func (p *UniversalPlugin) handleGet(cmd GetCommand) (string, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	index := p.findRuleIndexLocked(strings.TrimSpace(cmd.ID))
	if index < 0 {
		return "", fmt.Errorf("规则不存在：%s", cmd.ID)
	}
	return formatRule(p.Config.Rules[index]), nil
}

func (p *UniversalPlugin) handleDelete(cmd DeleteCommand) (string, error) {
	id := strings.TrimSpace(cmd.ID)

	p.mu.Lock()
	defer p.mu.Unlock()

	index := p.findRuleIndexLocked(id)
	if index < 0 {
		return "", fmt.Errorf("规则不存在：%s", id)
	}
	previous := cloneRules(p.Config.Rules)
	p.Config.Rules = append(p.Config.Rules[:index], p.Config.Rules[index+1:]...)
	if err := p.rebuildAndSaveLocked(previous); err != nil {
		return "", err
	}
	return fmt.Sprintf("已删除规则：%s", id), nil
}

func (p *UniversalPlugin) handleEnable(cmd EnableCommand) (string, error) {
	return p.setRuleEnabled(cmd.ID, true)
}

func (p *UniversalPlugin) handleDisable(cmd DisableCommand) (string, error) {
	return p.setRuleEnabled(cmd.ID, false)
}

func (p *UniversalPlugin) handleHelp(HelpCommand) (string, error) {
	return strings.Join([]string{
		"Universal 插件",
		"",
		"消息匹配：收到文本后按第一个空格切成 keyword 和 param，只使用 keyword 做全文匹配。",
		"参数解析：param 中 name=value 生成 {{name}}，普通参数生成 {{arg_1}}、{{arg_2}}；参数前缀 @ 会被移除。",
		"引用消息：{{quoter}} 是被引用者 nickname，{{quote}} 是被引用文本；存在引用时 quoter 自动成为 {{arg_1}}，文本参数后移。",
		"",
		"模板变量：{{text}}、{{keyword}}、{{param}}、{{quote}}、{{quoter}}、{{arg_1}}、{{arg_2}}，以及 name=value 生成的变量。",
		"艾特回复：规则 at=true 时，文本回复会优先 @ 引用消息被引用者，再追加消息参数中的 @ 用户。",
		"",
		"命令：",
		"/universal list",
		"/universal get <id>",
		"/universal add <id> -k <keywords> -u <url> [-m GET] [-t text] [-H A=B;C=D] [-b body] [-p result.path] [--at true|false]",
		"/universal update <id> [-k <keywords>] [-u <url>] [-m GET] [-t image] [-H A=B] [-b body] [-p result.path] [--at true|false] [--clear-headers] [--clear-body] [--clear-result-path]",
		"/universal enable <id>",
		"/universal disable <id>",
		"/universal delete <id>",
		"",
		"发送类型：text、emoji、image、video。",
	}, "\n"), nil
}

func (p *UniversalPlugin) setRuleEnabled(id string, enabled bool) (string, error) {
	id = strings.TrimSpace(id)

	p.mu.Lock()
	defer p.mu.Unlock()

	index := p.findRuleIndexLocked(id)
	if index < 0 {
		return "", fmt.Errorf("规则不存在：%s", id)
	}
	previous := cloneRules(p.Config.Rules)
	p.Config.Rules[index].Enabled = new(enabled)
	if err := p.rebuildAndSaveLocked(previous); err != nil {
		return "", err
	}
	if enabled {
		return fmt.Sprintf("已启用规则：%s", id), nil
	}
	return fmt.Sprintf("已禁用规则：%s", id), nil
}
