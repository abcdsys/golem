package main

import (
	"bytes"
	"image/png"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gogpu/gg/text"
	"github.com/sbgayhub/golem/sdk/message"
)

// now 固定测试基准时间：2026-07-10 15:30（周五）
var testNow = time.Date(2026, 7, 10, 15, 30, 0, 0, time.Local)

func msgOf(content string) *message.Message {
	return &message.Message{Content: content}
}

func TestParseTriggerNotTriggered(t *testing.T) {
	for _, content := range []string{
		"词云图怎么做", "wordcloudabc", "你好", "", "生成词云", "人物画像",
	} {
		if _, ok := parseTrigger(msgOf(content), testNow); ok {
			t.Errorf("%q 不应触发", content)
		}
	}
}

func TestParseTriggerDefault(t *testing.T) {
	trg, ok := parseTrigger(msgOf("词云"), testNow)
	if !ok {
		t.Fatal("「词云」应触发")
	}
	if trg.label != "近7天" {
		t.Errorf("label = %q, want 近7天", trg.label)
	}
	if want := testNow.Add(-7 * 24 * time.Hour); !trg.since.Equal(want) {
		t.Errorf("since = %v, want %v", trg.since, want)
	}
	if trg.name != "" {
		t.Errorf("name = %q, want 空", trg.name)
	}
}

func TestParseTriggerTimeRanges(t *testing.T) {
	midnight := time.Date(2026, 7, 10, 0, 0, 0, 0, time.Local)
	cases := []struct {
		content string
		label   string
		since   time.Time
		until   time.Time
	}{
		{"词云 今日", "今日", midnight, time.Time{}},
		{"词云 昨天", "昨日", midnight.AddDate(0, 0, -1), midnight},
		{"词云 本周", "本周", midnight.AddDate(0, 0, -4), time.Time{}}, // 周五 → 周一
		{"词云 本月", "本月", time.Date(2026, 7, 1, 0, 0, 0, 0, time.Local), time.Time{}},
		{"词云 全部", "全部", time.Time{}, time.Time{}},
		{"词云 30天", "近30天", testNow.Add(-30 * 24 * time.Hour), time.Time{}},
		{"词云 30", "近30天", testNow.Add(-30 * 24 * time.Hour), time.Time{}},
		{"wordcloud 今日", "今日", midnight, time.Time{}},
	}
	for _, c := range cases {
		trg, ok := parseTrigger(msgOf(c.content), testNow)
		if !ok {
			t.Errorf("%q 应触发", c.content)
			continue
		}
		if trg.label != c.label {
			t.Errorf("%q label = %q, want %q", c.content, trg.label, c.label)
		}
		if !trg.since.Equal(c.since) {
			t.Errorf("%q since = %v, want %v", c.content, trg.since, c.since)
		}
		if !trg.until.Equal(c.until) {
			t.Errorf("%q until = %v, want %v", c.content, trg.until, c.until)
		}
	}
}

func TestParseTriggerInvalidDays(t *testing.T) {
	// 全角数字、超上限、非法数字都不应被当作天数，而是当作成员名
	for _, content := range []string{"词云 ３０", "词云 99999", "词云 0天"} {
		trg, ok := parseTrigger(msgOf(content), testNow)
		if !ok {
			t.Fatalf("%q 应触发", content)
		}
		if trg.label != "近7天" {
			t.Errorf("%q 应保持默认时间范围，got label=%q", content, trg.label)
		}
		if trg.name == "" {
			t.Errorf("%q 的参数应落入成员名", content)
		}
	}
}

func TestParseTriggerName(t *testing.T) {
	cases := []struct {
		content string
		name    string
		label   string
	}{
		{"词云 @张三", "张三", "近7天"},
		{"词云@张三", "张三", "近7天"},
		{"词云 本周 @张三", "张三", "本周"},
		{"词云 张三", "张三", "近7天"},
		{"词云 ＠李四", "李四", "近7天"},
	}
	for _, c := range cases {
		trg, ok := parseTrigger(msgOf(c.content), testNow)
		if !ok {
			t.Errorf("%q 应触发", c.content)
			continue
		}
		if trg.name != c.name {
			t.Errorf("%q name = %q, want %q", c.content, trg.name, c.name)
		}
		if trg.label != c.label {
			t.Errorf("%q label = %q, want %q", c.content, trg.label, c.label)
		}
	}
}

func TestFilterMessages(t *testing.T) {
	until := time.Date(2026, 7, 10, 0, 0, 0, 0, time.Local)
	msgs := []historyMsg{
		{ID: 1, Content: "早上好", Timestamp: "2026-07-09 08:00:00"},
		{ID: 2, Content: "词云", Timestamp: "2026-07-09 09:00:00"},     // 触发命令，过滤
		{ID: 3, Content: "/help", Timestamp: "2026-07-09 10:00:00"},  // 命令，过滤
		{ID: 4, Content: "今天天气不错", Timestamp: "2026-07-10 08:00:00"}, // 超出 until，过滤
		{ID: 5, Content: "中午吃什么", Timestamp: "2026-07-09 23:59:59"},  // 保留
	}
	got := filterMessages(msgs, until)
	if len(got) != 2 || got[0].ID != 1 || got[1].ID != 5 {
		t.Errorf("filterMessages 结果不符: %+v", got)
	}
}

func TestTopN(t *testing.T) {
	freq := map[string]int{"cc": 3, "aa": 5, "bb": 5, "dd": 1}
	got := topN(freq, 3)
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}
	// 同频按词典序：aa < bb
	if got[0].word != "aa" || got[1].word != "bb" || got[2].word != "cc" {
		t.Errorf("topN 排序不符: %+v", got)
	}
}

func TestFontSizeFor(t *testing.T) {
	if got := fontSizeFor(10, 1, 10, 16, 96); got != 96 {
		t.Errorf("最高频词字号 = %v, want 96", got)
	}
	if got := fontSizeFor(1, 1, 10, 16, 96); got != 16 {
		t.Errorf("最低频词字号 = %v, want 16", got)
	}
	if got := fontSizeFor(5, 5, 5, 16, 96); got != 56 {
		t.Errorf("同频词字号 = %v, want 56", got)
	}
}

func TestSegmenterCountWords(t *testing.T) {
	seg, err := newSegmenter()
	if err != nil {
		t.Fatalf("初始化分词器失败: %v", err)
	}
	msgs := []historyMsg{
		{Content: "今天天气真不错，适合出去玩"},
		{Content: "明天天气也不错 233333 http://example.com"},
	}
	freq := seg.countWords(msgs)
	if freq["天气"] < 1 {
		t.Errorf("「天气」词频 = %d, want >= 1", freq["天气"])
	}
	for w := range freq {
		if w == "233333" || strings.Contains(strings.ToLower(w), "http") {
			t.Errorf("噪声词 %q 未被过滤", w)
		}
	}
}

// TestRenderSmoke 渲染冒烟测试。设置 WORDCLOUD_TEST_OUT 环境变量时把 PNG 写到该路径，便于人工查看。
func TestRenderSmoke(t *testing.T) {
	font, err := text.NewFontSource(embeddedFont)
	if err != nil {
		t.Fatalf("加载内置字体失败: %v", err)
	}
	defer func() { _ = font.Close() }()

	words := []wordCount{
		{"周末爬山", 120}, {"天气", 96}, {"下班", 80}, {"干饭", 72}, {"游戏", 66},
		{"版本更新", 60}, {"外卖", 55}, {"加班", 50}, {"电影", 46}, {"奶茶", 42},
		{"睡觉", 38}, {"上号", 35}, {"周五", 32}, {"开会", 30}, {"摸鱼", 28},
		{"代码", 26}, {"健身", 24}, {"跑步", 22}, {"旅游", 20}, {"拍照", 18},
		{"火锅", 17}, {"烧烤", 16}, {"追剧", 15}, {"音乐", 14}, {"读书", 13},
		{"猫猫", 12}, {"狗狗", 11}, {"下雨", 10}, {"台风", 9}, {"高温", 9},
		{"空调", 8}, {"西瓜", 8}, {"冰淇淋", 7}, {"足球", 7}, {"篮球", 6},
		{"世界杯", 6}, {"演唱会", 5}, {"门票", 5}, {"抢票", 4}, {"红包", 4},
		{"股票", 3}, {"基金", 3}, {"理财", 2}, {"存钱", 2}, {"打工", 2},
	}
	img, err := renderWordCloud(font, renderOptions{
		width:   1000,
		height:  640,
		minFont: 16,
		maxFont: 96,
	}, words, "近7天 · 1234 条发言 · 2026-07-10")
	if err != nil {
		t.Fatalf("渲染失败: %v", err)
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("PNG 编码失败: %v", err)
	}
	if buf.Len() == 0 {
		t.Fatal("PNG 为空")
	}
	if out := os.Getenv("WORDCLOUD_TEST_OUT"); out != "" {
		if err := os.WriteFile(out, buf.Bytes(), 0o644); err != nil {
			t.Fatalf("写出测试图片失败: %v", err)
		}
		t.Logf("测试图片已写出: %s", out)
	}
}
