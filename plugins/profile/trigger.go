package main

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/sbgayhub/golem/sdk/message"
)

// parseTrigger 解析「人物画像」类触发语。
//
// 触发形式：
//  1. 单输「人物画像」
//  2. 「人物画像」+ 空白 + 成员名 / 开关（如「人物画像 张三」「人物画像 --global」）
//  3. 「人物画像」+ @ + 成员名（如「人物画像@张三」，群聊 @ 提人常不带空格）
//
// 前缀后若既不是空白也不是 @（如「人物画像张三」「人物画像功能真不错」），一律不触发，
// 交由其它插件（如 ai）处理。成员名可以是任意字符，不做语义拦截。
//
// 注意：微信 @ 提人时插入的空白可能是非常规空格（NBSP U+00A0 / U+2005 等），
// 因此分隔符判定使用 unicode.IsSpace，而非只认 ASCII 空格。
//
// 支持开关：--global / -g（跨群全局画像）、--rebuild / -r（忽略已有画像从头冷启动）。
// 成员名前的 @（ASCII 或全角）会被自动忽略。返回 name（成员名，可空）、global、rebuild、triggered。
func parseTrigger(msg *message.Message) (name string, global, rebuild, triggered bool) {
	content := strings.TrimSpace(msg.GetContent())
	const prefix = "人物画像"
	if !strings.HasPrefix(content, prefix) {
		return "", false, false, false
	}
	rest := content[len(prefix):] // 前缀之后的原始剩余内容（未 trim）

	// 形式 1：单输「人物画像」（其后无实质内容）
	if strings.TrimSpace(rest) == "" {
		return "", false, false, true
	}

	// 形式 2 / 3：前缀后必须紧跟「任意空白」或「@」，否则不触发
	first, _ := utf8.DecodeRuneInString(rest)
	if !isAtSign(first) && !unicode.IsSpace(first) {
		return "", false, false, false
	}
	rest = strings.TrimSpace(rest)

	// 解析名字与开关（name 解析会忽略开头的 @，含全角）
	var nameParts []string
	for _, field := range strings.Fields(rest) {
		switch field {
		case "--global", "-g":
			global = true
		case "--rebuild", "-r":
			rebuild = true
		default:
			nameParts = append(nameParts, field)
		}
	}
	joined := strings.Join(nameParts, " ")
	joined = strings.TrimPrefix(joined, "@") // ASCII @
	joined = strings.TrimPrefix(joined, "＠") // 全角 @
	name = strings.TrimSpace(joined)
	return name, global, rebuild, true
}

// isAtSign 判断是否为 @ 符号（ASCII U+0040 或全角 U+FF20）
func isAtSign(r rune) bool {
	return r == '@' || r == '＠'
}

// extractAtTargetWxid 从消息的 @ 提人列表（atuserlist/Reminds）中取第一个
// 非机器人自身的 wxid，作为画像目标的直接定位。
// 没有 @ 提人、或只 @ 了机器人时返回空（交由按名字匹配的回退路径处理）。
func extractAtTargetWxid(msg *message.Message, selfWxid string) string {
	if msg == nil {
		return ""
	}
	text := msg.GetText()
	if text == nil {
		return ""
	}
	for _, r := range text.GetReminds() {
		r = strings.TrimSpace(r)
		if r == "" {
			continue
		}
		if r == selfWxid {
			continue // 排除 @ 机器人本身
		}
		return r
	}
	return ""
}
