package main

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func (p *UniversalPlugin) ruleForKeyword(keyword string) (Rule, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.rulesByKeyword == nil {
		return Rule{}, false
	}
	rule := p.rulesByKeyword[keyword]
	if rule == nil || !ruleEnabled(rule) {
		return Rule{}, false
	}
	return *rule, true
}

func (p *UniversalPlugin) rebuildIndex() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.rebuildIndexLocked()
}

func (p *UniversalPlugin) rebuildIndexLocked() error {
	index := make(map[string]*Rule, len(p.Config.Rules))
	ruleIDs := make(map[string]struct{}, len(p.Config.Rules))
	seenKeywords := map[string]string{}

	for i := range p.Config.Rules {
		rule := &p.Config.Rules[i]
		applyRuleDefaults(rule)
		if err := validateRule(*rule); err != nil {
			return err
		}
		if _, exists := ruleIDs[rule.ID]; exists {
			return fmt.Errorf("规则 ID 重复：%s", rule.ID)
		}
		ruleIDs[rule.ID] = struct{}{}

		for _, keyword := range rule.Keywords {
			if owner, exists := seenKeywords[keyword]; exists {
				return fmt.Errorf("关键词重复：%s，被规则 %s 和 %s 同时使用", keyword, owner, rule.ID)
			}
			seenKeywords[keyword] = rule.ID
			if ruleEnabled(rule) {
				index[keyword] = rule
			}
		}
	}

	p.rulesByKeyword = index
	return nil
}

func (p *UniversalPlugin) rebuildAndSaveLocked(previous []Rule) error {
	if err := p.rebuildIndexLocked(); err != nil {
		p.Config.Rules = previous
		_ = p.rebuildIndexLocked()
		return err
	}
	if err := p.SaveConfig(p); err != nil {
		p.Config.Rules = previous
		_ = p.rebuildIndexLocked()
		return err
	}
	return nil
}

func (p *UniversalPlugin) findRuleIndexLocked(id string) int {
	for i := range p.Config.Rules {
		if p.Config.Rules[i].ID == id {
			return i
		}
	}
	return -1
}

func parseKeywords(raw string) ([]string, error) {
	parts := strings.Split(raw, ",")
	keywords := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, part := range parts {
		keyword := strings.TrimSpace(part)
		if keyword == "" {
			continue
		}
		if strings.ContainsAny(keyword, " \t\r\n") {
			return nil, fmt.Errorf("关键词不能包含空白字符：%s", keyword)
		}
		if _, exists := seen[keyword]; exists {
			continue
		}
		seen[keyword] = struct{}{}
		keywords = append(keywords, keyword)
	}
	if len(keywords) == 0 {
		return nil, errors.New("至少需要一个关键词")
	}
	return keywords, nil
}

func validateRule(rule Rule) error {
	if strings.TrimSpace(rule.ID) == "" {
		return errors.New("规则 ID 不能为空")
	}
	if len(rule.Keywords) == 0 {
		return fmt.Errorf("规则 %s 至少需要一个关键词", rule.ID)
	}
	if strings.TrimSpace(rule.URL) == "" {
		return fmt.Errorf("规则 %s 的 URL 不能为空", rule.ID)
	}
	if err := validateMethod(rule.Method); err != nil {
		return fmt.Errorf("规则 %s: %w", rule.ID, err)
	}
	if err := validateSendType(rule.SendType); err != nil {
		return fmt.Errorf("规则 %s: %w", rule.ID, err)
	}
	if _, err := parseKeywords(strings.Join(rule.Keywords, ",")); err != nil {
		return fmt.Errorf("规则 %s: %w", rule.ID, err)
	}
	return nil
}

func applyRuleDefaults(rule *Rule) {
	rule.ID = strings.TrimSpace(rule.ID)
	rule.URL = strings.TrimSpace(rule.URL)
	rule.Method = normalizeMethod(rule.Method)
	rule.SendType = normalizeSendType(rule.SendType)
	rule.ResultPath = strings.TrimSpace(rule.ResultPath)
	if rule.Enabled == nil {
		rule.Enabled = new(true)
	}

	keywords := rule.Keywords[:0]
	seen := map[string]struct{}{}
	for _, keyword := range rule.Keywords {
		keyword = strings.TrimSpace(keyword)
		if keyword == "" {
			continue
		}
		if _, exists := seen[keyword]; exists {
			continue
		}
		seen[keyword] = struct{}{}
		keywords = append(keywords, keyword)
	}
	rule.Keywords = keywords
}

func validateMethod(method string) error {
	method = normalizeMethod(method)
	if method == "" || strings.ContainsAny(method, " \t\r\n") {
		return fmt.Errorf("invalid method: %s", method)
	}
	return nil
}

func validateSendType(sendType string) error {
	switch normalizeSendType(sendType) {
	case "text", "image", "video", "emoji":
		return nil
	default:
		return fmt.Errorf("unsupported send_type: %s", sendType)
	}
}

func normalizeMethod(method string) string {
	method = strings.ToUpper(strings.TrimSpace(method))
	if method == "" {
		return http.MethodGet
	}
	return method
}

func normalizeSendType(sendType string) string {
	sendType = strings.ToLower(strings.TrimSpace(sendType))
	if sendType == "" {
		return defaultSendType
	}
	return sendType
}

func ruleEnabled(rule *Rule) bool {
	return rule == nil || rule.Enabled == nil || *rule.Enabled
}

func ruleStatus(rule *Rule) string {
	if ruleEnabled(rule) {
		return "enabled"
	}
	return "disabled"
}

func formatRule(rule Rule) string {
	applyRuleDefaults(&rule)
	return strings.Join([]string{
		"规则：" + rule.ID,
		"状态：" + ruleStatus(&rule),
		"关键词：" + strings.Join(rule.Keywords, ","),
		"请求方法：" + rule.Method,
		"请求地址：" + rule.URL,
		"请求头：" + emptyPlaceholder(rule.Headers),
		"请求体：" + emptyPlaceholder(rule.Body),
		"发送类型：" + rule.SendType,
		"结果路径：" + emptyPlaceholder(rule.ResultPath),
		"艾特：" + strconv.FormatBool(rule.At),
	}, "\n")
}

func cloneRules(rules []Rule) []Rule {
	clone := make([]Rule, len(rules))
	for i := range rules {
		clone[i] = rules[i]
		if rules[i].Keywords != nil {
			clone[i].Keywords = append([]string(nil), rules[i].Keywords...)
		}
		if rules[i].Enabled != nil {
			clone[i].Enabled = new(*rules[i].Enabled)
		}
	}
	return clone
}
