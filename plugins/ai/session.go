package main

// getReplyRate 获取会话的回复概率（优先使用会话配置）
func (p *AiPlugin) getReplyRate(sessionKey string) float64 {
	if cfg := p.getSessionConfig(sessionKey); cfg != nil && cfg.ReplyRate != nil {
		return *cfg.ReplyRate
	}
	return p.configSnapshot().ReplyRate
}

// getActivePrompt 获取会话的活动提示词（优先使用会话配置）
func (p *AiPlugin) getActivePrompt(sessionKey string) string {
	if cfg := p.getSessionConfig(sessionKey); cfg != nil && cfg.ActivePrompt != nil {
		return *cfg.ActivePrompt
	}
	return p.configSnapshot().ActivePrompt
}

// getMaxContextMessages 获取会话的上下文消息数（优先使用会话配置）
func (p *AiPlugin) getMaxContextMessages(sessionKey string) int {
	if cfg := p.getSessionConfig(sessionKey); cfg != nil && cfg.MaxContextMessages != nil {
		return *cfg.MaxContextMessages
	}
	return p.configSnapshot().MaxContextMessages
}

// getActiveProvider 获取会话的活动 provider（优先使用会话配置，回退全局）
func (p *AiPlugin) getActiveProvider(sessionKey string) string {
	if cfg := p.getSessionConfig(sessionKey); cfg != nil && cfg.ActiveProvider != nil {
		return *cfg.ActiveProvider
	}
	return p.configSnapshot().ActiveProvider
}

// isSilent 判断会话是否处于静默模式（优先使用会话配置，回退全局）
func (p *AiPlugin) isSilent(sessionKey string) bool {
	if cfg := p.getSessionConfig(sessionKey); cfg != nil && cfg.Silence != nil {
		return *cfg.Silence
	}
	return p.configSnapshot().Silence
}

// getSessionConfig 线程安全地获取会话配置
func (p *AiPlugin) getSessionConfig(sessionKey string) *SessionConfig {
	p.sessionConfigMu.RLock()
	defer p.sessionConfigMu.RUnlock()
	if p.Config.SessionConfigs == nil {
		return nil
	}
	return p.Config.SessionConfigs[sessionKey]
}

// setSessionConfig 线程安全地设置会话配置
func (p *AiPlugin) setSessionConfig(sessionKey string, cfg *SessionConfig) {
	p.sessionConfigMu.Lock()
	defer p.sessionConfigMu.Unlock()
	if p.Config.SessionConfigs == nil {
		p.Config.SessionConfigs = make(map[string]*SessionConfig)
	}
	p.Config.SessionConfigs[sessionKey] = cfg
}

// deleteSessionConfig 线程安全地删除会话配置
func (p *AiPlugin) deleteSessionConfig(sessionKey string) {
	p.sessionConfigMu.Lock()
	defer p.sessionConfigMu.Unlock()
	if p.Config.SessionConfigs != nil {
		delete(p.Config.SessionConfigs, sessionKey)
	}
}

// listSessionKeys 获取所有已配置的会话 key
func (p *AiPlugin) listSessionKeys() []string {
	p.sessionConfigMu.RLock()
	defer p.sessionConfigMu.RUnlock()
	if p.Config.SessionConfigs == nil {
		return nil
	}
	keys := make([]string, 0, len(p.Config.SessionConfigs))
	for k := range p.Config.SessionConfigs {
		keys = append(keys, k)
	}
	return keys
}
