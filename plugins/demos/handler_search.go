package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/sbgayhub/golem/sdk/contact"
)

// ==================== 搜索类处理器 ====================

func (p *DemosPlugin) handleBdbk(receiver *contact.Contact, arg string) (bool, error) {
	if p.Config.BdbkURL == "" {
		p.sendText(receiver, "百度百科功能未配置，请联系管理员配置 bdbk_url")
		return true, nil
	}
	if arg == "" {
		p.sendText(receiver, "请输入查询关键词，例如：百度百科 人工智能")
		return true, nil
	}
	body, err := p.httpGet(p.Config.BdbkURL + url.QueryEscape(arg))
	if err != nil {
		return true, err
	}
	var resp struct {
		Key        string `json:"key"`
		Desc       string `json:"desc"`
		Abstract   string `json:"abstract"`
		Copyrights string `json:"copyrights"`
	}
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		return true, fmt.Errorf("解析百科失败")
	}
	if resp.Key == "" {
		p.sendText(receiver, "没查到该百科含义")
		return true, nil
	}
	p.sendText(receiver, fmt.Sprintf("百科：%s\n%s\n摘要：%s\n版权：%s", resp.Key, resp.Desc, resp.Abstract, resp.Copyrights))
	return true, nil
}

func (p *DemosPlugin) handleSdj(receiver *contact.Contact, arg string) (bool, error) {
	if arg == "" {
		p.sendText(receiver, "请输入短剧名称，例如：搜短剧 爱情")
		return true, nil
	}
	body, err := p.httpGet(sdjURL + url.QueryEscape(arg))
	if err != nil {
		return true, err
	}
	var resp struct {
		Code int `json:"code"`
		Data []struct {
			Name     string `json:"name"`
			Viewlink string `json:"viewlink"`
			Addtime  string `json:"addtime"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(body), &resp); err != nil || resp.Code != 200 {
		p.sendText(receiver, "功能不可用了，请联系管理员")
		return true, nil
	}
	n := len(resp.Data)
	if n == 0 {
		p.sendText(receiver, "没有找到相关短剧")
		return true, nil
	}
	limit := p.Config.MaxList
	if n < limit {
		limit = n
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "查询到%d条\n", n)
	for i := 0; i < limit; i++ {
		item := resp.Data[i]
		fmt.Fprintf(&sb, "%d-> 短剧名称：【%s】\n短剧链接：%s\n添加时间：%s\n", i+1, item.Name, item.Viewlink, item.Addtime)
	}
	p.sendText(receiver, sb.String())
	return true, nil
}

// ==================== 生成类处理器 ====================

func (p *DemosPlugin) handleHunYanQingJian(receiver *contact.Contact, arg string) (bool, error) {
	defaultMsg := "婚宴请柬邀请函生成（仅供娱乐！）\n输入格式为（不需要【】符号）：婚宴请柬 【新郎名】,【新娘名】,【邀请人名】"
	if arg == "" {
		p.sendText(receiver, defaultMsg)
		return true, nil
	}
	parts := strings.Split(arg, ",")
	if len(parts) != 3 {
		p.sendText(receiver, defaultMsg)
		return true, nil
	}
	xl := strings.TrimSpace(parts[0])
	xn := strings.TrimSpace(parts[1])
	yq := strings.TrimSpace(parts[2])
	date := time.Now().AddDate(0, 0, 1).Format("20060102")
	imgURL := fmt.Sprintf("%s?to=%s&date=%s&head=%s|%s&event=结婚典礼恭备喜宴&time=中午十二时恭候&place=明月大酒店&tail=%s|%s",
		hunyanURL, url.QueryEscape(yq), date, url.QueryEscape(xl), url.QueryEscape(xn), url.QueryEscape(xl), url.QueryEscape(xn))
	return true, p.sendImage(receiver, imgURL)
}

func (p *DemosPlugin) handleDogDoc(receiver *contact.Contact, arg string) (bool, error) {
	defaultMsg := "狗屁不通文章生成\n输入格式为（不需要【】符号）：狗屁不通文章生成 +【内容描述】"
	if arg == "" {
		p.sendText(receiver, defaultMsg)
		return true, nil
	}
	body, err := p.httpGet(dogDocURL + url.QueryEscape(arg))
	if err != nil {
		return true, err
	}
	var resp struct {
		Code int    `json:"code"`
		Data string `json:"data"`
	}
	if err := json.Unmarshal([]byte(body), &resp); err != nil || resp.Code != 200 {
		return true, fmt.Errorf("解析失败")
	}
	p.sendText(receiver, resp.Data)
	return true, nil
}

// ==================== 音乐类处理器 ====================

func (p *DemosPlugin) handleSjSing(receiver *contact.Contact, arg string) (bool, error) {
	body, err := p.httpGet(sjSingURL)
	if err != nil {
		return true, err
	}
	var resp struct {
		Code     int    `json:"code"`
		Nickname string `json:"nickname"`
		Pic      string `json:"pic"`
		Title    string `json:"title"`
		Singer   string `json:"singer"`
		Lyrics   string `json:"lyrics"`
		URL      string `json:"url"`
	}
	if err := json.Unmarshal([]byte(body), &resp); err != nil || resp.Code != 200 {
		p.sendText(receiver, "获取随机唱失败")
		return true, nil
	}
	lines := strings.Split(resp.Lyrics, "\n")
	var sb strings.Builder
	sb.WriteString("\n[歌词]\n")
	for i, line := range lines {
		prefix := "> "
		if i%2 == 1 {
			prefix = "* "
		}
		fmt.Fprintf(&sb, "%s%s\n", prefix, line)
	}
	p.sendText(receiver, fmt.Sprintf("%s\n%s-%s\n%s", resp.Nickname, resp.Title, resp.Singer, sb.String()))
	if err := p.sendImage(receiver, resp.Pic); err != nil {
		slog.Warn("[demos] 随机唱头像发送失败", "err", err)
	}
	p.sendVoice(receiver, resp.URL)
	return true, nil
}

func (p *DemosPlugin) handleSearchMusic(receiver *contact.Contact, arg string) (bool, error) {
	if arg == "" {
		p.sendText(receiver, "输入格式为（不需要【】符号）：火子搜歌+【搜索词】")
		return true, nil
	}
	searchURL := kuwoSearchURL +
		"?prod=kwplayer_ar_9.3.7.2&corp=kuwo&newver=2&vipver=9.3.7.2" +
		"&source=kwplayer_ar_9.3.7.2_meizu.apk&p2p=1&notrace=0&client=kt" +
		"&pn=0&rn=45&ver=kwplayer_ar_9.3.7.2&vipver=1&show_copyright_off=1" +
		"&newver=2&correct=1&ft=music&cluster=0&strategy=2012&encoding=utf8" +
		"&rformat=json&vermerge=1&mobi=1&searchapi=2&issubtitle=1" +
		"&spPrivilege=0&all=" + url.QueryEscape(arg)

	headers := map[string]string{
		"User-Agent": "Dalvik/2.1.0 (Linux; U; Android 9; GM1910 Build/PQ3B.190801.07101020)",
		"Host":       "search.kuwo.cn",
	}
	body, err := p.httpGetWithHeaders(searchURL, headers)
	if err != nil {
		return true, err
	}
	var resp struct {
		Abslist []struct {
			SongName   string `json:"SONGNAME"`
			Artist     string `json:"ARTIST"`
			DCTargetID string `json:"DC_TARGETID"`
		} `json:"abslist"`
	}
	if err := json.Unmarshal([]byte(body), &resp); err != nil || len(resp.Abslist) == 0 {
		p.sendText(receiver, "没有找到相关歌曲")
		return true, nil
	}
	n := 10
	if len(resp.Abslist) < n {
		n = len(resp.Abslist)
	}
	var sb strings.Builder
	sb.WriteString("为你找到以下内容：\n")
	for i := 0; i < n; i++ {
		s := resp.Abslist[i]
		fmt.Fprintf(&sb, "%d. 【%s】\n%s\n点歌号：%s\n\n", i+1, s.SongName, s.Artist, s.DCTargetID)
	}
	p.sendText(receiver, sb.String())
	return true, nil
}

func (p *DemosPlugin) handleChooseMusic(receiver *contact.Contact, arg string) (bool, error) {
	if arg == "" {
		p.sendText(receiver, "输入格式为（不需要【】符号）：火子点歌+【点歌号】")
		return true, nil
	}
	chooseURL := kuwoMusicURL +
		"?f=web&source=kwplayerhd_ar_4.3.0.8_tianbao_T1A_qirui.apk" +
		"&type=convert_url_with_sign&br=128kmp3&rid=" + url.QueryEscape(arg)

	headers := map[string]string{
		"User-Agent":      "Apache-HttpClient/UNAVAILABLE (java 1.4)",
		"Accept-Encoding": "identity",
		"Host":            "nmobi.kuwo.cn",
	}
	body, err := p.httpGetWithHeaders(chooseURL, headers)
	if err != nil || body == "" {
		p.sendText(receiver, "点歌失败，也许点歌号不可用。")
		return true, nil
	}
	var musicResp struct {
		Data struct {
			URL string `json:"url"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(body), &musicResp); err != nil || musicResp.Data.URL == "" {
		p.sendText(receiver, "点歌失败，也许点歌号不可用。")
		return true, nil
	}

	infoURL := kuwoInfoURL + "?httpsStatus=1&reqId=fcd6bc60-3e06-11ec-8722-67bb659a8433&musicId=" + url.QueryEscape(arg)
	infoHeaders := map[string]string{
		"User-Agent":      "Apache-HttpClient/UNAVAILABLE (java 1.4)",
		"Accept-Encoding": "identity",
		"Host":            "m.kuwo.cn",
	}
	infoBody, err := p.httpGetWithHeaders(infoURL, infoHeaders)
	if err != nil {
		p.sendText(receiver, "点歌失败，也许点歌号不可用。")
		return true, nil
	}
	var infoResp struct {
		Data struct {
			Songinfo struct {
				SongName string `json:"songName"`
				Artist   string `json:"artist"`
			} `json:"songinfo"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(infoBody), &infoResp); err != nil {
		p.sendText(receiver, "点歌失败，也许点歌号不可用。")
		return true, nil
	}
	p.sendText(receiver, fmt.Sprintf("%s - %s", infoResp.Data.Songinfo.SongName, infoResp.Data.Songinfo.Artist))
	p.sendVoice(receiver, musicResp.Data.URL)
	return true, nil
}

// ==================== 帮助处理器 ====================

func (p *DemosPlugin) handleExplain(receiver *contact.Contact, arg string) (bool, error) {
	msg := "粗略记录一下\n" +
		"【一言/今日句子】【小姐姐视频】【搜短剧】【百度百科】【撸猫】【旺财】【看星空】【温馨提示】【一句】【吟诗】【讲笑话】\n" +
		"【名画赏析】【婚宴请柬】【热点视频】【娱乐视频】【靓仔视频】【脑筋急转弯】【来段绕口令】\n" +
		"【谚语】【抽签】【答案之书】【吃什么】【king台词】【l台词】\n" +
		"【小黑子表情】【随机二次元】【保安日记】【acg美图】\n" +
		"【狗屁不通文章生成】【--小姐姐视频】【懒羊羊k歌】【怼脸自拍视频】【看穿搭】【丝滑舞蹈视频】【快手随机翻唱】【随机坤坤】\n" +
		"【随机唱】【火子搜歌】【火子点歌】"
	p.sendText(receiver, msg)
	return true, nil
}
