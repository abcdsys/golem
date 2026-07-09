package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/tidwall/gjson"
)

// executeResult 承载请求链路的最终产物。
// text  非空：作为文本消息发送（send_type=text）。
// mediaData 非空：作为媒体消息发送（send_type=image/video/emoji）。
type executeResult struct {
	text      string
	mediaData []byte
}

func executeRule(rule Rule, vars map[string]string, timeout time.Duration) (executeResult, error) {
	url, err := renderTemplate(rule.URL, vars)
	if err != nil {
		return executeResult{}, fmt.Errorf("render url: %w", err)
	}
	headersText, err := renderTemplate(rule.Headers, vars)
	if err != nil {
		return executeResult{}, fmt.Errorf("render headers: %w", err)
	}
	body, err := renderTemplate(rule.Body, vars)
	if err != nil {
		return executeResult{}, fmt.Errorf("render body: %w", err)
	}
	headers, err := parseHeaders(headersText)
	if err != nil {
		return executeResult{}, err
	}

	response, err := doRequest(rule.Method, strings.TrimSpace(url), headers, body, timeout)
	if err != nil {
		return executeResult{}, err
	}

	// send_type × result_path 双控：
	// - text：result_path 空→body 原文；非空→gjson 提取文本
	// - 媒体：result_path 空→body 即二进制直接发送；非空→gjson 提取 URL 后二次 GET，body 即二进制
	if normalizeSendType(rule.SendType) == "text" {
		text, err := extractResult(response, rule.ResultPath)
		if err != nil {
			return executeResult{}, err
		}
		return executeResult{text: text}, nil
	}

	if strings.TrimSpace(rule.ResultPath) == "" {
		if len(response) == 0 {
			return executeResult{}, errors.New("media response body is empty")
		}
		return executeResult{mediaData: response}, nil
	}

	nextURL, err := extractResult(response, rule.ResultPath)
	if err != nil {
		return executeResult{}, err
	}
	nextURL = strings.TrimSpace(nextURL)
	if nextURL == "" {
		return executeResult{}, errors.New("media url extracted from result_path is empty")
	}
	mediaData, err := doRequest(http.MethodGet, nextURL, headers, "", timeout)
	if err != nil {
		return executeResult{}, fmt.Errorf("fetch media: %w", err)
	}
	if len(mediaData) == 0 {
		return executeResult{}, errors.New("media response body is empty")
	}
	return executeResult{mediaData: mediaData}, nil
}

func doRequest(method, url string, headers map[string]string, body string, timeout time.Duration) ([]byte, error) {
	if url == "" {
		return nil, errors.New("request url is empty")
	}
	escapedURL, err := escapeURL(url)
	if err != nil {
		return nil, err
	}
	if timeout <= 0 {
		timeout = time.Duration(defaultHTTPTimeoutSeconds) * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, normalizeMethod(method), escapedURL, reader)
	if err != nil {
		return nil, err
	}

	hasContentType := false
	for key, value := range headers {
		req.Header.Set(key, value)
		if strings.EqualFold(key, "Content-Type") {
			hasContentType = true
		}
	}
	if body != "" && !hasContentType {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("request failed: status=%d body=%s", resp.StatusCode, truncate(data, 300))
	}
	return data, nil
}

func escapeURL(rawURL string) (string, error) {
	rawURL = strings.TrimSpace(rawURL)
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	parsed.RawQuery = escapeRawQuery(parsed.RawQuery)
	return parsed.String(), nil
}

func escapeRawQuery(rawQuery string) string {
	if rawQuery == "" {
		return ""
	}

	parts := strings.Split(rawQuery, "&")
	for i, part := range parts {
		key, value, hasValue := strings.Cut(part, "=")
		key = escapeQueryPart(key)
		if !hasValue {
			parts[i] = key
			continue
		}
		parts[i] = key + "=" + escapeQueryPart(value)
	}
	return strings.Join(parts, "&")
}

func escapeQueryPart(part string) string {
	unescaped, err := url.QueryUnescape(part)
	if err != nil {
		return url.QueryEscape(part)
	}
	return url.QueryEscape(unescaped)
}

func extractResult(body []byte, path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return string(body), nil
	}
	result := gjson.GetBytes(body, path)
	if !result.Exists() {
		return "", fmt.Errorf("result_path has no result: %s", path)
	}
	return result.String(), nil
}

func parseHeaders(raw string) (map[string]string, error) {
	headers := map[string]string{}
	for _, part := range strings.Split(raw, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		key, value, ok := strings.Cut(part, "=")
		if !ok {
			return nil, fmt.Errorf("invalid header: %s", part)
		}
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, fmt.Errorf("invalid header: %s", part)
		}
		headers[key] = strings.TrimSpace(value)
	}
	return headers, nil
}
