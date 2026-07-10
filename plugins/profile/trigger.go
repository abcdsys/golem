package main

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/sbgayhub/golem/sdk/message"
)

// triggerOpts 「人物画像」触发语的解析结果
type triggerOpts struct {
	name    string // 目标成员名（可空=查发起人自己；私聊中指定他人仅主人可用）
	group   string // #指定的群名或群 wxid（可空=全局/当前群；仅私聊生效）
	global  bool   // --global / -g：跨群全局画像（群聊中使用；私聊默认即全局）
	rebuild bool   // --rebuild / -r：忽略已有画像从头冷启动
}

// parseTrigger 解析「人物画像」类触发语。
//
// 触发形式：
//  1. 单输「人物画像」
//  2. 「人物画像」+ 空白 + 成员名 / 开关（如「人物画像 张三」「人物画像 --global」）
//  3. 「人物画像」+ @ + 成员名（如「人物画像@张三」，群聊 @ 提人常不带空格）
//  4. 「人物画像」+ # + 群名（如「人物画像#摸鱼群」「人物画像 张三 #摸鱼群」，私聊指定群范围）
//
// 前缀后若既不是空白也不是 @ / #（如「人物画像张三」「人物画像功能真不错」），一律不触发，
// 交由其它插件（如 ai）处理。成员名可以是任意字符，不做语义拦截。
//
// 注意：微信 @ 提人时插入的空白可能是非常规空格（NBSP U+00A0 / U+2005 等），
// 因此分隔符判定使用 unicode.IsSpace，而非只认 ASCII 空格。
//
// 支持开关：--global / -g（跨群全局画像）、--rebuild / -r（忽略已有画像从头冷启动）。
// 成员名前的 @（ASCII 或全角）会被自动忽略。
// #（ASCII 或全角＃）之后、下一个开关之前的字段都归入群名，因此群名可含空格；
// 群名需写在成员名之后（如「人物画像 张三 #高数 学习群」）。
func parseTrigger(msg *message.Message) (opts triggerOpts, triggered bool) {
	content := strings.TrimSpace(msg.GetContent())
	const prefix = "人物画像"
	if !strings.HasPrefix(content, prefix) {
		return triggerOpts{}, false
	}
	rest := content[len(prefix):] // 前缀之后的原始剩余内容（未 trim）

	// 形式 1：单输「人物画像」（其后无实质内容）
	if strings.TrimSpace(rest) == "" {
		return triggerOpts{}, true
	}

	// 形式 2 / 3 / 4：前缀后必须紧跟「任意空白」或「@」「#」，否则不触发
	first, _ := utf8.DecodeRuneInString(rest)
	if !isAtSign(first) && !isHashSign(first) && !unicode.IsSpace(first) {
		return triggerOpts{}, false
	}
	rest = strings.TrimSpace(rest)

	// 解析名字、群名与开关（name 解析会忽略开头的 @，含全角）
	var nameParts, groupParts []string
	inGroup := false
	for _, field := range strings.Fields(rest) {
		switch {
		case field == "--global" || field == "-g":
			opts.global = true
		case field == "--rebuild" || field == "-r":
			opts.rebuild = true
		case hasHashPrefix(field):
			inGroup = true
			if f := trimHashPrefix(field); f != "" {
				groupParts = append(groupParts, f)
			}
		case inGroup: // # 之后的非开关字段都归入群名（群名可含空格）
			groupParts = append(groupParts, field)
		default:
			nameParts = append(nameParts, field)
		}
	}
	joined := strings.Join(nameParts, " ")
	joined = strings.TrimPrefix(joined, "@") // ASCII @
	joined = strings.TrimPrefix(joined, "＠") // 全角 @
	opts.name = strings.TrimSpace(joined)
	opts.group = strings.TrimSpace(strings.Join(groupParts, " "))
	return opts, true
}

// isAtSign 判断是否为 @ 符号（ASCII U+0040 或全角 U+FF20）
func isAtSign(r rune) bool {
	return r == '@' || r == '＠'
}

// isHashSign 判断是否为 # 符号（ASCII U+0023 或全角 U+FF03）
func isHashSign(r rune) bool {
	return r == '#' || r == '＃'
}

func hasHashPrefix(s string) bool {
	r, _ := utf8.DecodeRuneInString(s)
	return isHashSign(r)
}

func trimHashPrefix(s string) string {
	s = strings.TrimPrefix(s, "#")
	s = strings.TrimPrefix(s, "＃")
	return s
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
