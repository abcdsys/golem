package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sbgayhub/golem/sdk/contact"
)

// ==================== 文本类处理器 ====================

func (p *DemosPlugin) handleWxts(receiver *contact.Contact, arg string) (bool, error) {
	body, err := p.httpGet(wxtsURL)
	if err != nil {
		return true, err
	}
	var resp struct {
		Code int `json:"code"`
		Data struct {
			Greeting string `json:"greeting"`
			Tip      string `json:"tip"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(body), &resp); err != nil || resp.Code != 200 {
		return true, fmt.Errorf("解析温馨提示失败")
	}
	p.sendText(receiver, fmt.Sprintf("%s\n%s", resp.Data.Greeting, resp.Data.Tip))
	return true, nil
}

func (p *DemosPlugin) handleYiju(receiver *contact.Contact, arg string) (bool, error) {
	body, err := p.httpGet(yijuURL)
	if err != nil {
		return true, err
	}
	var resp struct {
		Code int `json:"code"`
		Data struct {
			Name string `json:"name"`
			From string `json:"from"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(body), &resp); err != nil || resp.Code != 200 {
		return true, fmt.Errorf("解析一句失败")
	}
	p.sendText(receiver, fmt.Sprintf("【%s】\n——【%s】", resp.Data.Name, resp.Data.From))
	return true, nil
}

func (p *DemosPlugin) handleYiyan(receiver *contact.Contact, arg string) (bool, error) {
	body, err := p.httpGet("https://v1.hitokoto.cn/")
	if err != nil {
		return true, err
	}
	var resp struct {
		Hitokoto string `json:"hitokoto"`
		From     string `json:"from"`
	}
	if err := json.Unmarshal([]byte(body), &resp); err != nil || resp.Hitokoto == "" {
		return true, fmt.Errorf("解析一言失败")
	}
	if resp.From == "" {
		resp.From = "未知"
	}
	p.sendText(receiver, fmt.Sprintf("【%s】\n——【%s】", resp.Hitokoto, resp.From))
	return true, nil
}

func (p *DemosPlugin) handleShici(receiver *contact.Contact, arg string) (bool, error) {
	body, err := p.httpGet(shiciURL)
	if err != nil {
		return true, err
	}
	var resp struct {
		Code int `json:"code"`
		Data struct {
			Origin   string `json:"origin"`
			Author   string `json:"author"`
			Content  string `json:"content"`
			Category string `json:"category"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(body), &resp); err != nil || resp.Code != 200 {
		return true, fmt.Errorf("解析诗词失败")
	}
	d := resp.Data
	p.sendText(receiver, fmt.Sprintf("【%s】\n——%s\n%s\n\n诗词类型：%s", d.Origin, d.Author, d.Content, d.Category))
	return true, nil
}

func (p *DemosPlugin) handleHaha(receiver *contact.Contact, arg string) (bool, error) {
	body, err := p.httpGet(hahaURL)
	if err != nil {
		return true, err
	}
	var resp struct {
		Code int `json:"code"`
		Data struct {
			Title   string `json:"title"`
			Content string `json:"content"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(body), &resp); err != nil || resp.Code != 200 {
		return true, fmt.Errorf("解析笑话失败")
	}
	p.sendText(receiver, fmt.Sprintf("\"%s\"\n%s", resp.Data.Title, resp.Data.Content))
	return true, nil
}

func (p *DemosPlugin) handleJzw(receiver *contact.Contact, arg string) (bool, error) {
	body, err := p.httpGet(jzwURL)
	if err != nil {
		return true, err
	}
	var resp struct {
		Code int `json:"code"`
		Data struct {
			Question string `json:"question"`
			Answer   string `json:"answer"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(body), &resp); err != nil || resp.Code != 200 {
		return true, fmt.Errorf("解析脑筋急转弯失败")
	}
	p.sendText(receiver, fmt.Sprintf("问题：%s\n\n答案：%s", resp.Data.Question, resp.Data.Answer))
	return true, nil
}

func (p *DemosPlugin) handleRao(receiver *contact.Contact, arg string) (bool, error) {
	body, err := p.httpGet(raoURL)
	if err != nil {
		return true, err
	}
	var resp struct {
		Code int `json:"code"`
		Data struct {
			Title string `json:"title"`
			Msg   string `json:"msg"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(body), &resp); err != nil || resp.Code != 200 {
		return true, fmt.Errorf("解析绕口令失败")
	}
	p.sendText(receiver, fmt.Sprintf("\"%s\"\n%s", resp.Data.Title, resp.Data.Msg))
	return true, nil
}

func (p *DemosPlugin) handleYanyu(receiver *contact.Contact, arg string) (bool, error) {
	body, err := p.httpGet(yanyuURL)
	if err != nil {
		return true, err
	}
	var resp struct {
		Code int `json:"code"`
		Data struct {
			Content string `json:"content"`
			Source  string `json:"source"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(body), &resp); err != nil || resp.Code != 200 {
		return true, fmt.Errorf("解析谚语失败")
	}
	p.sendText(receiver, fmt.Sprintf("\"%s\"\n分类：%s", resp.Data.Content, resp.Data.Source))
	return true, nil
}

func (p *DemosPlugin) handleChouqian(receiver *contact.Contact, arg string) (bool, error) {
	body, err := p.httpGet(chouqianURL)
	if err != nil {
		return true, err
	}
	var resp struct {
		Code int `json:"code"`
		Data struct {
			ID   string `json:"id"`
			Text string `json:"text"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(body), &resp); err != nil || resp.Code != 200 {
		return true, fmt.Errorf("解析抽签失败")
	}
	p.sendText(receiver, fmt.Sprintf("\"%s\"\n%s", resp.Data.Text, resp.Data.ID))
	return true, nil
}

func (p *DemosPlugin) handleBay(receiver *contact.Contact, arg string) (bool, error) {
	body, err := p.httpGet(bayURL)
	if err != nil {
		return true, err
	}
	var resp struct {
		Code int `json:"code"`
		Data struct {
			Zh string `json:"zh"`
			En string `json:"en"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(body), &resp); err != nil || resp.Code != 200 {
		return true, fmt.Errorf("解析答案之书失败")
	}
	p.sendText(receiver, fmt.Sprintf("%s\n%s", resp.Data.Zh, resp.Data.En))
	return true, nil
}

func (p *DemosPlugin) handleEat(receiver *contact.Contact, arg string) (bool, error) {
	body, err := p.httpGet(eatURL)
	if err != nil {
		return true, err
	}
	var resp struct {
		Code int    `json:"code"`
		Data string `json:"data"`
	}
	if err := json.Unmarshal([]byte(body), &resp); err != nil || resp.Code != 200 {
		return true, fmt.Errorf("解析吃什么失败")
	}
	p.sendText(receiver, resp.Data)
	return true, nil
}

func (p *DemosPlugin) handleRj(receiver *contact.Contact, arg string) (bool, error) {
	body, err := p.httpGet(rjURL)
	if err != nil {
		return true, err
	}
	body = strings.ReplaceAll(body, "\\n", "\n")
	body = strings.ReplaceAll(body, "\\t", "\t")
	p.sendText(receiver, body)
	return true, nil
}

func (p *DemosPlugin) handleKingTc(receiver *contact.Contact, arg string) (bool, error) {
	body, err := p.httpGet(kingTcURL)
	if err != nil {
		return true, err
	}
	var resp struct {
		Code int `json:"code"`
		Data struct {
			Name    string `json:"name"`
			Content string `json:"content"`
			Img     string `json:"img"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(body), &resp); err != nil || resp.Code != 200 {
		return true, fmt.Errorf("解析 king 台词失败")
	}
	p.sendText(receiver, fmt.Sprintf("~%s~\n%s", resp.Data.Name, resp.Data.Content))
	return true, p.sendImage(receiver, resp.Data.Img)
}

func (p *DemosPlugin) handleLolTc(receiver *contact.Contact, arg string) (bool, error) {
	body, err := p.httpGet(lolTcURL)
	if err != nil {
		return true, err
	}
	var resp struct {
		Code int `json:"code"`
		Data struct {
			Name    string `json:"name"`
			Content string `json:"content"`
			Img     string `json:"img"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(body), &resp); err != nil || resp.Code != 200 {
		return true, fmt.Errorf("解析 l 台词失败")
	}
	p.sendText(receiver, fmt.Sprintf("~%s~\n%s", resp.Data.Name, resp.Data.Content))
	return true, p.sendImage(receiver, resp.Data.Img)
}

func (p *DemosPlugin) handleSjkk(receiver *contact.Contact, arg string) (bool, error) {
	p.sendVoice(receiver, sjkkURL)
	return true, nil
}
