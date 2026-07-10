package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type chatCompletionRequest struct {
	Model    string          `json:"model"`
	Messages []openAIMessage `json:"messages"`
}

type chatCompletionResponse struct {
	Choices []struct {
		Message openAIMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

// resolveProvider 解析会话当前生效的 provider（会话级覆盖优先，回退全局）
func (p *AiPlugin) resolveProvider(sessionKey string) (*Provider, error) {
	config := p.configSnapshot()
	name := p.getActiveProvider(sessionKey)
	if strings.TrimSpace(name) == "" {
		return nil, errors.New("未配置 active provider，请先 /ai provider-add 新增再 /ai set -p 切换")
	}
	prov, ok := config.Providers[name]
	if !ok || prov == nil {
		return nil, fmt.Errorf("provider 不存在：%s", name)
	}
	if strings.TrimSpace(prov.BaseURL) == "" {
		return nil, fmt.Errorf("provider %s 缺少 base_url", name)
	}
	if strings.TrimSpace(prov.APIKey) == "" {
		return nil, fmt.Errorf("provider %s 缺少 api_key", name)
	}
	if strings.TrimSpace(prov.Model) == "" {
		return nil, fmt.Errorf("provider %s 缺少 model", name)
	}
	return prov, nil
}

func (p *AiPlugin) chat(sessionKey string) (string, error) {
	config := p.configSnapshot()
	prov, err := p.resolveProvider(sessionKey)
	if err != nil {
		return "", err
	}

	messages := make([]openAIMessage, 0, p.getMaxContextMessages(sessionKey)+1)
	activePrompt := p.getActivePrompt(sessionKey)
	if prompt, ok := config.Prompts[activePrompt]; ok && strings.TrimSpace(prompt) != "" {
		messages = append(messages, openAIMessage{Role: "system", Content: prompt + p.getPreMadePrompts()})
	}
	messages = append(messages, p.contextMessages(sessionKey)...)
	if len(messages) == 0 {
		return "", errors.New("AI 上下文为空")
	}

	timeout := prov.HTTPTimeoutSeconds
	if timeout <= 0 {
		timeout = config.HTTPTimeoutSeconds
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()
	return callOpenAI(ctx, http.DefaultClient, prov.BaseURL, prov.APIKey, chatCompletionRequest{
		Model:    prov.Model,
		Messages: messages,
	})
}

func callOpenAI(ctx context.Context, client *http.Client, baseURL, apiKey string, payload chatCompletionRequest) (string, error) {
	if client == nil {
		client = http.DefaultClient
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("序列化 AI 请求失败: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, chatCompletionURL(baseURL), bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("创建 AI 请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36 Edg/141.0.0.0")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("请求 AI 接口失败: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取 AI 响应失败: %w", err)
	}
	var result chatCompletionResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("解析 AI 响应失败: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if result.Error != nil && result.Error.Message != "" {
			return "", fmt.Errorf("AI 接口返回错误: %s", result.Error.Message)
		}
		return "", fmt.Errorf("AI 接口返回状态码: %d", resp.StatusCode)
	}
	if result.Error != nil && result.Error.Message != "" {
		return "", fmt.Errorf("AI 接口返回错误: %s", result.Error.Message)
	}
	if len(result.Choices) == 0 {
		return "", errors.New("AI 响应缺少 choices")
	}
	return strings.TrimSpace(result.Choices[0].Message.Content), nil
}

func chatCompletionURL(baseURL string) string {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if strings.HasSuffix(baseURL, "/chat/completions") {
		return baseURL
	}
	return baseURL + "/chat/completions"
}
