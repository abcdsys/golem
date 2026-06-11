package main

import (
	"fmt"
	"strings"

	"github.com/sbgayhub/golem/sdk/chatroom"
	"github.com/sbgayhub/golem/sdk/contact"
	"github.com/sbgayhub/golem/sdk/message"
)

const (
	todayDateFilter     = "AND date = date('now', 'localtime')"
	yesterdayDateFilter = "AND date = date('now', '-1 day', 'localtime')"
	weekDateFilter      = "AND date >= date('now', 'localtime', '-' || ((CAST(strftime('%w', 'now', 'localtime') AS INTEGER) + 6) % 7) || ' days')"
	monthDateFilter     = "AND date >= date('now', 'localtime', 'start of month')"
	detailKeyword       = "发言详情"
)

type rankPeriod struct {
	Title      string
	DateFilter string
}

var rankPeriods = map[string]rankPeriod{
	"今日排行": {Title: "今日发言排行", DateFilter: todayDateFilter},
	"昨日排行": {Title: "昨日发言排行", DateFilter: yesterdayDateFilter},
	"本周排行": {Title: "本周发言排行", DateFilter: weekDateFilter},
	"本月排行": {Title: "本月发言排行", DateFilter: monthDateFilter},
	"总排行":  {Title: "总发言排行"},
	"发言详情": {},
}

func isRankingKeyword(msg *message.Message) bool {
	if msg.GetSender().GetType() != contact.ContactType_CONTACT_TYPE_CHATROOM {
		return false
	}
	_, ok := rankPeriods[strings.TrimSpace(msg.GetContent())]
	return ok
}

func (p *StatisticsPlugin) handleRanking(msg *message.Message) (bool, error) {
	content := strings.TrimSpace(msg.GetContent())
	if content == "发言详情" {
		return p.sendSpeakerDetail(msg)
	}

	period, ok := rankPeriods[content]
	if !ok {
		return false, nil
	}
	return p.sendRank(msg, period)
}

func (p *StatisticsPlugin) sendRank(msg *message.Message, period rankPeriod) (bool, error) {
	sender := msg.GetSender().GetUsername()
	if sender == "" {
		return false, nil
	}

	entries, err := p.store.QueryRank(sender, period.DateFilter, defaultRankLimit)
	if err != nil {
		return true, err
	}
	if len(entries) == 0 {
		return true, p.sendText(msg.GetSender(), period.Title+"\n暂无统计数据", nil)
	}

	for i := range entries {
		counts, err := p.store.QueryMemberTypeCounts(sender, entries[i].Member, period.DateFilter)
		if err != nil {
			return true, err
		}
		entries[i].Detail = "\n" + formatTypeCountsInline(counts)
	}

	total, err := p.store.QueryTotal(sender, period.DateFilter)
	if err != nil {
		return true, err
	}

	desc := totalSummaryText(total)
	records := p.buildRankRecords(msg, entries, desc+formatTypeCountsBlock(total.Types))
	return true, p.sendRecord(msg.GetSender(), period.Title, desc, records)
}

func (p *StatisticsPlugin) sendSpeakerDetail(msg *message.Message) (bool, error) {
	sender := msg.GetSender().GetUsername()
	member := msg.GetMember().GetUsername()
	if sender == "" || member == "" {
		return false, nil
	}

	sections := []struct {
		Title      string
		DateFilter string
	}{
		{Title: "今日详情", DateFilter: todayDateFilter},
		{Title: "昨日详情", DateFilter: yesterdayDateFilter},
		{Title: "本周详情", DateFilter: weekDateFilter},
		{Title: "本月详情", DateFilter: monthDateFilter},
		{Title: "总结详情"},
	}

	var builder strings.Builder
	speaker := p.member(sender, member, msg.GetMember())
	speakerName := member
	if speaker != nil {
		speakerName = memberName(speaker)
	}
	builder.WriteString(fmt.Sprintf("@ %s 发言详情如下\n", speakerName))
	for _, section := range sections {
		counts, err := p.store.QueryMemberTypeCounts(sender, member, section.DateFilter)
		if err != nil {
			return true, err
		}
		builder.WriteString("===== ")
		builder.WriteString(section.Title)
		builder.WriteString(" =====\n")
		builder.WriteString(formatTypeCounts(counts))
		builder.WriteString("\n")
	}

	reminds := []string{member}
	return true, p.sendText(msg.GetSender(), strings.TrimRight(builder.String(), "\n"), reminds)
}

func (p *StatisticsPlugin) buildRankRecords(msg *message.Message, entries []rankEntry, summary string) []recordItem {
	sender := msg.GetSender().GetUsername()
	records := make([]recordItem, 0, len(entries)+1)
	for i, entry := range entries {
		member := p.member(sender, entry.Member, msg.GetMember())
		name := entry.Member
		avatar := ""
		if member != nil {
			name = memberName(member)
			avatar = member.GetAvatar()
		}
		records = append(records, recordItem{
			Name:    name,
			Avatar:  avatar,
			Content: fmt.Sprintf("第 %d 名：共发言 %d 条%s", i+1, entry.Count, entry.Detail),
			Time:    fmt.Sprintf("第%d名", i+1),
		})
	}
	if summary != "" {
		records = append(records, recordItem{
			Name:    msg.GetSender().GetNickname(),
			Avatar:  msg.GetSender().GetAvatar(),
			Content: summary,
			Time:    "总结",
		})
	}
	return records
}

func (p *StatisticsPlugin) member(chatroomID, memberID string, current *chatroom.Member) *chatroom.Member {
	if current != nil && current.GetUsername() == memberID {
		return current
	}
	if p.chatroom != nil {
		if member := p.chatroom.GetMember(chatroomID, memberID); member != nil {
			return member
		}
	}
	return nil
}

func memberName(member *chatroom.Member) string {
	if member.GetNickname() == "" {
		if member.GetDisplayName() != "" {
			return member.GetDisplayName()
		}
		return member.GetUsername()
	}
	if member.GetDisplayName() != "" && member.GetDisplayName() != member.GetNickname() {
		return member.GetNickname() + " [" + member.GetDisplayName() + "]"
	}
	return member.GetNickname()
}

func totalSummaryText(total totalSummary) string {
	return fmt.Sprintf("%d 人发言，%d 条消息", total.Speakers, total.Messages)
}

func formatTypeCounts(counts []typeCount) string {
	total := 0
	var builder strings.Builder
	for _, item := range counts {
		total += item.Count
		builder.WriteString(item.Type)
		builder.WriteString(": ")
		builder.WriteString(fmt.Sprint(item.Count))
		builder.WriteString("\n")
	}
	detail := strings.TrimRight(builder.String(), "\n")
	if detail == "" {
		return "共 0 条消息"
	}
	return fmt.Sprintf("共 %d 条消息\n%s", total, detail)
}

func formatTypeCountsInline(counts []typeCount) string {
	if len(counts) == 0 {
		return "暂无类型详情"
	}

	parts := make([]string, 0, len(counts))
	for _, item := range counts {
		parts = append(parts, fmt.Sprintf("%s %d 条", item.Type, item.Count))
	}
	return strings.Join(parts, "，")
}

func formatTypeCountsBlock(counts []typeCount) string {
	if len(counts) == 0 {
		return ""
	}
	return "\n" + formatTypeCountsInline(counts)
}
