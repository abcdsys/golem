package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// timeLayout statistics 插件存储与查询使用的本地时间格式
const timeLayout = "2006-01-02 15:04:05"

// historyMsg statistics.query_messages 能力返回的单条历史发言
type historyMsg struct {
	ID        int64  `json:"id"`
	Content   string `json:"content"`
	Timestamp string `json:"timestamp"`
}

// queryHistory 经跨插件调用 statistics.query_messages 查询历史发言（不直接读 statistics.db）。
// member 为空表示统计整群；since 零值表示全部历史；limit>0 时超出取最近 limit 条。
func (p *WordCloudPlugin) queryHistory(chatroomID, member string, since time.Time, limit int) ([]historyMsg, error) {
	if p.caller == nil {
		return nil, errors.New("调用能力未注入（需要 statistics 插件提供 statistics.query_messages）")
	}
	args := map[string]string{
		"chatroom": chatroomID,
	}
	if member != "" {
		args["member"] = member
	}
	if !since.IsZero() {
		args["since"] = since.Format(timeLayout)
	}
	if limit > 0 {
		args["limit"] = strconv.Itoa(limit)
	}
	_, data, err := p.caller.CallPlugin("statistics.query_messages", args)
	if err != nil {
		return nil, err
	}
	var msgs []historyMsg
	if err := json.Unmarshal(data, &msgs); err != nil {
		return nil, fmt.Errorf("解析历史消息失败: %w", err)
	}
	return msgs, nil
}

// filterMessages 过滤不参与统计的消息：截止时间之后的、疑似机器人命令的。
// timestamp 与 until 均为本地时间的固定格式字符串，直接按字典序比较即可。
func filterMessages(msgs []historyMsg, until time.Time) []historyMsg {
	untilStr := ""
	if !until.IsZero() {
		untilStr = until.Format(timeLayout)
	}
	filtered := make([]historyMsg, 0, len(msgs))
	for _, m := range msgs {
		if untilStr != "" && m.Timestamp >= untilStr {
			continue
		}
		if isCommandLike(m.Content) {
			continue
		}
		filtered = append(filtered, m)
	}
	return filtered
}

// isCommandLike 判断消息是否像机器人命令（本插件触发词、/ 开头的命令），
// 这类消息不参与词频统计，避免「词云」本身刷进词云。
func isCommandLike(content string) bool {
	c := strings.TrimSpace(content)
	if strings.HasPrefix(c, "/") {
		return true
	}
	_, ok := cutTriggerPrefix(c)
	return ok
}
