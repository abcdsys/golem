package main

func (p *AiPlugin) appendContext(key string, msg openAIMessage) {
	if key == "" || msg.Content == "" {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.sessions == nil {
		p.sessions = map[string][]openAIMessage{}
	}
	items := append(p.sessions[key], msg)
	limit := p.maxContextMessages()
	if len(items) > limit {
		items = items[len(items)-limit:]
	}
	p.sessions[key] = items
}

func (p *AiPlugin) contextMessages(key string) []openAIMessage {
	p.mu.Lock()
	defer p.mu.Unlock()
	items := p.sessions[key]
	if len(items) == 0 {
		return nil
	}
	return append([]openAIMessage(nil), items...)
}

func (p *AiPlugin) clearContext(key string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.sessions, key)
}
