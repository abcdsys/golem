package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/sbgayhub/golem/sdk/message"
)

// maxDays 天数参数上限，拦截无意义的超大数字
const maxDays = 3650

// trigger 词云触发解析结果
type trigger struct {
	since time.Time // 统计起始时间（含边界）；零值=全部历史
	until time.Time // 统计截止时间（不含边界）；零值=不限，仅「昨日」这类封闭区间使用
	label string    // 时间范围描述，用于图片脚注与提示文案
	name  string    // 指定成员名（可空，来自「词云 @张三」或「词云 张三」）
}

// parseTrigger 解析词云触发命令。
//
// 触发形式（前缀后必须是消息结尾、空白或 @，避免「词云图怎么做」误触发）：
//   - 「词云」——默认近 7 天
//   - 「词云 今日/昨日/本周/本月/全部」
//   - 「词云 30天」/「词云 30」——近 N 天
//   - 「词云 @张三」/「词云 张三」——指定成员，可与时间参数组合（如「词云 本周 @张三」）
//
// 注意：微信 @ 提人插入的空白可能是 NBSP 等非常规空格，分隔符判定使用 unicode.IsSpace。
func parseTrigger(msg *message.Message, now time.Time) (trigger, bool) {
	content := strings.TrimSpace(msg.GetContent())

	rest, ok := cutTriggerPrefix(content)
	if !ok {
		return trigger{}, false
	}
	if rest != "" {
		first, _ := utf8.DecodeRuneInString(rest)
		if !isAtSign(first) && !unicode.IsSpace(first) {
			return trigger{}, false
		}
	}

	// 默认近 7 天
	trg := trigger{
		since: now.Add(-7 * 24 * time.Hour),
		label: "近7天",
	}

	var nameParts []string
	for _, field := range strings.Fields(rest) {
		if !applyTimeToken(&trg, field, now) {
			nameParts = append(nameParts, field)
		}
	}
	name := strings.Join(nameParts, " ")
	name = strings.TrimPrefix(name, "@") // ASCII @
	name = strings.TrimPrefix(name, "＠") // 全角 @
	trg.name = strings.TrimSpace(name)
	return trg, true
}

// cutTriggerPrefix 匹配触发前缀，返回前缀之后的剩余内容
func cutTriggerPrefix(content string) (string, bool) {
	for _, prefix := range []string{"词云", "wordcloud"} {
		if strings.HasPrefix(content, prefix) {
			return content[len(prefix):], true
		}
	}
	return "", false
}

// applyTimeToken 尝试把 field 解析为时间范围参数，成功时更新 trg 并返回 true；
// 无法识别的 field 交由调用方当作成员名处理。
func applyTimeToken(trg *trigger, field string, now time.Time) bool {
	midnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	switch field {
	case "今日", "今天":
		trg.since, trg.until, trg.label = midnight, time.Time{}, "今日"
	case "昨日", "昨天":
		trg.since, trg.until, trg.label = midnight.AddDate(0, 0, -1), midnight, "昨日"
	case "本周", "这周":
		offset := (int(now.Weekday()) + 6) % 7 // 周一为一周起点
		trg.since, trg.until, trg.label = midnight.AddDate(0, 0, -offset), time.Time{}, "本周"
	case "本月", "这个月":
		firstDay := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		trg.since, trg.until, trg.label = firstDay, time.Time{}, "本月"
	case "全部", "所有":
		trg.since, trg.until, trg.label = time.Time{}, time.Time{}, "全部"
	default:
		days, ok := parseDays(field)
		if !ok {
			return false
		}
		trg.since, trg.until, trg.label = now.Add(-time.Duration(days)*24*time.Hour), time.Time{}, fmt.Sprintf("近%d天", days)
	}
	return true
}

// parseDays 解析「30天」「30」形式的天数参数。
// 仅接受 ASCII 数字（strconv.Atoi 会拒绝全角数字），范围 [1, maxDays]。
func parseDays(field string) (int, bool) {
	days, err := strconv.Atoi(strings.TrimSuffix(field, "天"))
	if err != nil || days < 1 || days > maxDays {
		return 0, false
	}
	return days, true
}

// isAtSign 判断是否为 @ 符号（ASCII U+0040 或全角 U+FF20）
func isAtSign(r rune) bool {
	return r == '@' || r == '＠'
}
