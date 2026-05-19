package plugin

import "strings"

// matchesTopic 检查事件主题是否匹配插件订阅
// 订阅 "message" 应匹配 "message::text"、"message::image" 等（前缀匹配）
func matchesTopic(eventTopic string, subscriptions []string) bool {
	for _, sub := range subscriptions {
		if strings.HasPrefix(eventTopic, sub) {
			return true
		}
	}
	return false
}
