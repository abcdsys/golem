package main

import (
	"encoding/json"
	"fmt"

	"github.com/sbgayhub/golem/sdk/contact"
)

// ==================== 视频类处理器 ====================

func (p *DemosPlugin) handleXjj(receiver *contact.Contact, arg string) (bool, error) {
	u, err := p.getRedirectURL(xjjURL)
	if err != nil {
		return true, err
	}
	p.sendVideoOrCard(receiver, u)
	return true, nil
}

func (p *DemosPlugin) handleXjj2(receiver *contact.Contact, arg string) (bool, error) {
	body, err := p.httpGet(xjj2URL)
	if err != nil {
		return true, err
	}
	var resp struct {
		Mp4Video string `json:"mp4_video"`
	}
	if err := json.Unmarshal([]byte(body), &resp); err != nil || resp.Mp4Video == "" {
		return true, fmt.Errorf("解析小姐姐视频失败")
	}
	p.sendVideoOrCard(receiver, resp.Mp4Video)
	return true, nil
}

func (p *DemosPlugin) handleRdVideo(receiver *contact.Contact, arg string) (bool, error) {
	body, err := p.httpGet(rdVideoURL)
	if err != nil {
		return true, err
	}
	var resp struct {
		Data struct {
			Video string `json:"video"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(body), &resp); err != nil || resp.Data.Video == "" {
		return true, fmt.Errorf("解析热点视频失败")
	}
	p.sendVideoOrCard(receiver, resp.Data.Video)
	return true, nil
}

func (p *DemosPlugin) handleYlVideo(receiver *contact.Contact, arg string) (bool, error) {
	body, err := p.httpGet(ylVideoURL)
	if err != nil {
		return true, err
	}
	var resp struct {
		Data struct {
			Video string `json:"video"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(body), &resp); err != nil || resp.Data.Video == "" {
		return true, fmt.Errorf("解析娱乐视频失败")
	}
	p.sendVideoOrCard(receiver, resp.Data.Video)
	return true, nil
}

func (p *DemosPlugin) handleBoyVideo(receiver *contact.Contact, arg string) (bool, error) {
	body, err := p.httpGet(boyURL)
	if err != nil {
		return true, err
	}
	var resp struct {
		Data struct {
			Video string `json:"video"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(body), &resp); err != nil || resp.Data.Video == "" {
		return true, fmt.Errorf("解析靓仔视频失败")
	}
	p.sendVideoOrCard(receiver, resp.Data.Video)
	return true, nil
}

func (p *DemosPlugin) handleLyyKg(receiver *contact.Contact, arg string) (bool, error) {
	u, err := p.getRedirectURL(lyyKgURL)
	if err != nil {
		return true, err
	}
	p.sendVideoOrCard(receiver, u)
	return true, nil
}

func (p *DemosPlugin) handleDuilian(receiver *contact.Contact, arg string) (bool, error) {
	u, err := p.getRedirectURL(duilianURL)
	if err != nil {
		return true, err
	}
	p.sendVideoOrCard(receiver, u)
	return true, nil
}

func (p *DemosPlugin) handleChuanda(receiver *contact.Contact, arg string) (bool, error) {
	u, err := p.getRedirectURL(chuandaURL)
	if err != nil {
		return true, err
	}
	p.sendVideoOrCard(receiver, u)
	return true, nil
}

func (p *DemosPlugin) handleShwd(receiver *contact.Contact, arg string) (bool, error) {
	u, err := p.getRedirectURL(shwdURL)
	if err != nil {
		return true, err
	}
	p.sendVideoOrCard(receiver, u)
	return true, nil
}

func (p *DemosPlugin) handleKsfc(receiver *contact.Contact, arg string) (bool, error) {
	u, err := p.getRedirectURL(ksFcURL)
	if err != nil {
		return true, err
	}
	p.sendVideoOrCard(receiver, u)
	return true, nil
}
