package main

import (
	"sort"
	"strings"
	"unicode"

	"github.com/go-ego/gse"
)

// extraStopWords gse 内嵌停用词之外的补充：聊天高频灌水词、本插件触发词
var extraStopWords = []string{
	"哈哈", "哈哈哈", "哈哈哈哈", "哈哈哈哈哈", "呵呵", "嘿嘿", "嘻嘻", "hhh", "hhhh",
	"这个", "那个", "什么", "怎么", "为什么", "然后", "就是", "但是", "所以", "如果",
	"感觉", "现在", "时候", "东西", "问题", "直接", "应该", "可能", "确实", "其实",
	"词云", "wordcloud",
}

// segmenter 中文分词器封装（gse 内嵌词典 + 内嵌停用词 + 补充停用词）
type segmenter struct {
	seg gse.Segmenter
}

// newSegmenter 创建分词器。词典与停用词均使用 gse 内嵌数据，无外部文件依赖。
func newSegmenter() (*segmenter, error) {
	var seg gse.Segmenter
	if err := seg.LoadDictEmbed(); err != nil {
		return nil, err
	}
	if err := seg.LoadStopEmbed(); err != nil {
		return nil, err
	}
	seg.AddStopArr(extraStopWords...)
	return &segmenter{seg: seg}, nil
}

// countWords 对历史发言分词并统计词频
func (s *segmenter) countWords(msgs []historyMsg) map[string]int {
	freq := make(map[string]int)
	for _, m := range msgs {
		for _, w := range s.seg.Cut(m.Content, true) {
			w = strings.TrimSpace(w)
			if s.isValidWord(w) {
				freq[w]++
			}
		}
	}
	return freq
}

// isValidWord 过滤停用词、单字符、链接片段，以及不含任何文字的词（纯数字、纯符号等）
func (s *segmenter) isValidWord(w string) bool {
	if w == "" || s.seg.IsStop(w) {
		return false
	}
	runes := []rune(w)
	if len(runes) < 2 {
		return false
	}
	if strings.Contains(strings.ToLower(w), "http") {
		return false
	}
	for _, r := range runes {
		if unicode.IsLetter(r) {
			return true
		}
	}
	return false
}

// wordCount 单个词的频次
type wordCount struct {
	word  string
	count int
}

// topN 取词频最高的 n 个词，按频次降序排列（同频按词典序，保证结果稳定）
func topN(freq map[string]int, n int) []wordCount {
	list := make([]wordCount, 0, len(freq))
	for w, c := range freq {
		list = append(list, wordCount{word: w, count: c})
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].count != list[j].count {
			return list[i].count > list[j].count
		}
		return list[i].word < list[j].word
	})
	if n > 0 && len(list) > n {
		list = list[:n]
	}
	return list
}
