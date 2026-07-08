package main

import (
	"strconv"
	"strings"
	"unicode"
)

// estimateTokens 估算文本 token 数（无需真实分词器）。
// 中文/中日韩字符约 1 token/字；其他字符按 4 字符≈1 token。结果偏保守，不会低估太多。
func estimateTokens(text string) int {
	var cjk, other int
	for _, r := range text {
		if unicode.Is(unicode.Han, r) ||
			unicode.Is(unicode.Hiragana, r) ||
			unicode.Is(unicode.Katakana, r) {
			cjk++
		} else if r > unicode.MaxASCII {
			cjk++ // 其他 CJK 及全角字符也按 1 计
		} else {
			other++
		}
	}
	return cjk + other/4 + 1
}

// truncateMessage 单条消息过长时硬截断，附标记以免模型误以为内容完整
func truncateMessage(content string, maxChars int) string {
	if maxChars <= 0 || len([]rune(content)) <= maxChars {
		return content
	}
	runes := []rune(content)
	return string(runes[:maxChars]) + "…(内容已截断)"
}

// splitIntoChunks 按 token 估算预算切块，返回切块结果。
// 水位线（maxID）不在这里返回——采样/截断后必须基于「实际送入模型的块」
// 计算水位线，否则会虚报覆盖范围。由调用方用 maxChunkID 计算。
func splitIntoChunks(msgs []historyMsg, budget, maxSingleChars int) [][]historyMsg {
	var chunks [][]historyMsg
	var cur []historyMsg
	used := 0

	for _, m := range msgs {
		content := truncateMessage(m.Content, maxSingleChars)
		t := estimateTokens(content)
		// 单条本身已超预算：单独成块
		if t > budget && len(cur) > 0 {
			chunks = append(chunks, cur)
			cur = nil
			used = 0
		}
		if used+t > budget && len(cur) > 0 {
			chunks = append(chunks, cur)
			cur = nil
			used = 0
		}
		cur = append(cur, historyMsg{ID: m.ID, Content: content, Timestamp: m.Timestamp})
		used += t
	}
	if len(cur) > 0 {
		chunks = append(chunks, cur)
	}
	return chunks
}

// formatChunk 把一块消息拼成喂给模型的文本。
// 每条带时间标记（本机本地时间），便于模型准确判断「活跃时段」，而非只能从内容词猜测。
func formatChunk(ch []historyMsg) string {
	var b strings.Builder
	b.WriteString("以下是该成员的部分历史发言（按时间顺序，时间为本机本地时间）：\n")
	for i, m := range ch {
		b.WriteString("--- 发言 ")
		b.WriteString(strconv.Itoa(i + 1))
		b.WriteString(" [")
		b.WriteString(compactTime(m.Timestamp))
		b.WriteString("] ---\n")
		b.WriteString(m.Content)
		b.WriteString("\n")
	}
	return b.String()
}

// compactTime 从 statistics 库的时间字符串（如 "2026-07-08 10:51:52"）
// 提取更紧凑的 "07-08 10:51" 形式，既省 token 又保留日期+时分，便于判断活跃时段。
// 解析失败时原样返回，保证健壮。
func compactTime(s string) string {
	s = strings.TrimSpace(s)
	// 形如 YYYY-MM-DD HH:MM:SS
	if len(s) >= 16 && s[4] == '-' && s[7] == '-' && s[10] == ' ' {
		return s[5:16] // "MM-DD HH:MM"
	}
	return s
}

// maxChunkID 返回给定切片中所有消息的最大 id（用作真实水位线）。
// 截断后只保留部分块，必须基于「实际送入模型的块」计算水位线，否则会把
// 被丢弃块的消息也算进已覆盖范围，导致这些消息永久不再被分析。
func maxChunkID(chunks [][]historyMsg) int64 {
	var maxID int64
	for _, ch := range chunks {
		for _, m := range ch {
			if m.ID > maxID {
				maxID = m.ID
			}
		}
	}
	return maxID
}

// keepRecentChunks 块数超限时，只保留最新的 max 块，丢弃更早的块。
func keepRecentChunks(chunks [][]historyMsg, max int) [][]historyMsg {
	if len(chunks) <= max {
		return chunks
	}
	return chunks[len(chunks)-max:]
}
