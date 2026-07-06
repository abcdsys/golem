package main

import (
	"encoding/json"
	"fmt"

	"github.com/sbgayhub/golem/sdk/contact"
)

// ==================== 图片类处理器 ====================

func (p *DemosPlugin) handleCat(receiver *contact.Contact, arg string) (bool, error) {
	body, err := p.httpGet(catURL)
	if err != nil {
		return true, err
	}
	var result []struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal([]byte(body), &result); err != nil || len(result) == 0 {
		return true, fmt.Errorf("解析猫咪图片失败")
	}
	return true, p.sendImage(receiver, result[0].URL)
}

func (p *DemosPlugin) handleDog(receiver *contact.Contact, arg string) (bool, error) {
	body, err := p.httpGet(dogURL)
	if err != nil {
		return true, err
	}
	var result struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal([]byte(body), &result); err != nil || result.Message == "" {
		return true, fmt.Errorf("解析狗狗图片失败")
	}
	return true, p.sendImage(receiver, result.Message)
}

func (p *DemosPlugin) handleTw(receiver *contact.Contact, arg string) (bool, error) {
	body, err := p.httpGet(twURL)
	if err != nil {
		return true, err
	}
	var list []struct {
		URL         string `json:"url"`
		Title       string `json:"title"`
		Date        string `json:"date"`
		Explanation string `json:"explanation"`
	}
	if err := json.Unmarshal([]byte(body), &list); err != nil || len(list) == 0 {
		return true, fmt.Errorf("解析天文图片失败")
	}
	data := list[0]
	p.sendText(receiver, fmt.Sprintf("看星空：%s\n时间：%s\n描述：%s", data.Title, data.Date, data.Explanation))
	return true, p.sendImage(receiver, data.URL)
}

func (p *DemosPlugin) handlePainting(receiver *contact.Contact, arg string) (bool, error) {
	body, err := p.httpGet(paintingURL)
	if err != nil {
		return true, err
	}
	var resp struct {
		Code int `json:"code"`
		Data struct {
			Img     string `json:"img"`
			Title   string `json:"title"`
			Dynasty string `json:"dynasty"`
			Source  string `json:"source"`
			Info    string `json:"info"`
			Content string `json:"content"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(body), &resp); err != nil || resp.Code != 200 {
		return true, fmt.Errorf("解析名画失败")
	}
	d := resp.Data
	p.sendText(receiver, fmt.Sprintf("《%s》\n--%s  %s\n%s\n%s", d.Title, d.Dynasty, d.Source, d.Info, d.Content))
	return true, p.sendImage(receiver, d.Img)
}

func (p *DemosPlugin) handleXhzbq(receiver *contact.Contact, arg string) (bool, error) {
	return true, p.sendImage(receiver, xhzBqURL)
}

func (p *DemosPlugin) handleSjecy(receiver *contact.Contact, arg string) (bool, error) {
	u, err := p.getRedirectURL(sjecyURL)
	if err != nil {
		return true, err
	}
	return true, p.sendImage(receiver, u)
}

func (p *DemosPlugin) handleAcg(receiver *contact.Contact, arg string) (bool, error) {
	u, err := p.getRedirectURL(acgURL)
	if err != nil {
		return true, err
	}
	return true, p.sendImage(receiver, u)
}
