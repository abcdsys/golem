package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/sbgayhub/golem/sdk/chatroom"
	"github.com/sbgayhub/golem/sdk/contact"
	"github.com/sbgayhub/golem/sdk/message"
)

func displayNameOf(n named) string {
	for _, v := range []string{n.GetRemark(), n.GetNickname(), n.GetAlias(), n.GetUsername()} {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// handleProfile 人物画像触发入口：识别 issuer / chatroomWxid / @ 的 wxid，异步生成 + 回复。
// 返回 (handled=true, nil) 以消费事件，避免 ai 重复回复。
func (p *ProfilePlugin) handleProfile(msg *message.Message, opts triggerOpts) (bool, error) {
	if msg.GetSender() == nil {
		return true, fmt.Errorf("无法确定消息来源")
	}

	// 异步生成+发送：冷启动可能涉及数十块 × AI 调用，耗时远超事件分发超时（1 分钟）。
	// 同步会触发超时→事件链继续→ai 重复回复。这里立即返回 handled=true 消费事件，后台完成。
	go p.processProfile(msg, opts)
	return true, nil
}

// processProfile 实际的画像生成与发送（在后台 goroutine 中运行）。
func (p *ProfilePlugin) processProfile(msg *message.Message, opts triggerOpts) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("[profile] 画像生成崩溃", "err", r)
		}
	}()

	// 事件消息自带 Member（真实发消息的人）；群聊里 Sender 是群，Member 是人
	var issuer named
	if m := msg.GetMember(); m != nil {
		issuer = m
	} else {
		issuer = msg.GetSender()
	}
	chatroomWxid := ""
	if msg.Sender.GetType() == contact.ContactType_CONTACT_TYPE_CHATROOM {
		chatroomWxid = msg.GetSender().GetUsername()
	}

	// @ 提人时，微信附带被 @ 人的 wxid（atuserlist/Reminds），优先用于定位
	selfWxid := ""
	if p.contact != nil {
		if self := p.contact.GetSelf(); self != nil {
			selfWxid = self.GetUsername()
		}
	}
	atWxid := extractAtTargetWxid(msg, selfWxid)

	text, err := p.runProfile(issuer, chatroomWxid, opts, atWxid)
	if err != nil {
		slog.Warn("[profile] 画像生成失败", "err", err)
		errMsg := "画像生成失败：" + err.Error()
		if sendErr := p.sendText(msg.GetSender(), errMsg); sendErr != nil {
			slog.Warn("[profile] 发送画像错误信息失败", "err", sendErr)
		}
		return
	}
	if text != "" {
		if sendErr := p.sendProfileResult(msg.GetSender(), text); sendErr != nil {
			slog.Warn("[profile] 发送画像失败", "err", sendErr)
		}
	}
}

// runProfile 权限与范围判定入口：按私聊 / 群聊分流。
func (p *ProfilePlugin) runProfile(issuer named, chatroomWxid string, opts triggerOpts, atWxid string) (string, error) {
	if issuer == nil || issuer.GetUsername() == "" {
		return "", fmt.Errorf("无法确定消息来源")
	}
	var owner *contact.Contact
	if p.contact != nil {
		owner = p.contact.GetOwner()
	}
	if chatroomWxid == "" {
		return p.runPrivateProfile(issuer, owner, opts)
	}
	return p.runChatroomProfile(issuer, owner, chatroomWxid, opts, atWxid)
}

// runPrivateProfile 私聊路径。
// 任何人：查自己的全局画像（默认）或自己在 #指定群 的画像；
// 主人：还可查指定成员的全局 / #指定群画像。
// 私聊默认即全局范围，--global 开关在此路径无实际作用；范围由 #群名 决定。
func (p *ProfilePlugin) runPrivateProfile(issuer named, owner *contact.Contact, opts triggerOpts) (string, error) {
	isOwner := owner != nil && issuer.GetUsername() == owner.GetUsername()

	// 解析 #群名 → 群 wxid（scope；空 = 全局跨群）
	scopeChatroom, scopeLabel := "", ""
	if opts.group != "" {
		wxid, display, err := p.resolveChatroom(opts.group)
		if err != nil {
			return err.Error(), nil
		}
		scopeChatroom, scopeLabel = wxid, display
	}

	// 未指名或名字就是自己 → 查自己（人人可用）
	targetName := strings.TrimSpace(opts.name)
	if targetName == "" || isSelfName(issuer, targetName) {
		return p.generate(scopeChatroom, issuer.GetUsername(), displayNameOf(issuer), opts.rebuild)
	}

	// 指定他人：仅主人可用
	if !isOwner {
		return "私聊中仅主人可查看他人画像；发「人物画像」查看自己的全局画像，或「人物画像 #群名」查看自己在指定群的画像", nil
	}

	// 主人查指定成员：#指定群时优先按群成员匹配（群显示名/昵称等），
	// 未命中再按全局联系人解析（好友备注与群内显示名不一致的场景）
	if scopeChatroom != "" {
		if mem, ok := p.findMember(scopeChatroom, targetName); ok {
			return p.generate(scopeChatroom, mem.GetUsername(), displayNameOf(mem), opts.rebuild)
		}
	}
	wxid, display, err := p.resolveGlobal(targetName)
	if err != nil {
		if scopeChatroom != "" {
			return fmt.Sprintf("未在群「%s」及联系人中找到成员：%s", scopeLabel, targetName), nil
		}
		return err.Error(), nil
	}
	return p.generate(scopeChatroom, wxid, display, opts.rebuild)
}

// runChatroomProfile 群聊路径：任意成员可查本群成员画像；--global 跨群；#群名 仅私聊可用。
func (p *ProfilePlugin) runChatroomProfile(issuer named, owner *contact.Contact, chatroomWxid string, opts triggerOpts, atWxid string) (string, error) {
	if opts.group != "" {
		return "「#群名」仅私聊可用，群聊内默认生成本群画像", nil
	}

	var scopeChatroom, memberWxid, displayName string
	targetName := strings.TrimSpace(opts.name)

	switch {
	case opts.global:
		// @ 提人时直接用 wxid；否则按名字在全局联系人缓存中解析
		if atWxid != "" {
			memberWxid = atWxid
			displayName = targetName
			if p.contact != nil {
				if c := p.contact.Get(atWxid); c != nil && c.GetUsername() != "" {
					displayName = displayNameOf(c)
				}
			}
		} else {
			if targetName == "" {
				return "全局画像需指定成员昵称，例如「人物画像 张三 --global」", nil
			}
			wxid, disp, err := p.resolveGlobal(targetName)
			if err != nil {
				return err.Error(), nil
			}
			memberWxid, displayName = wxid, disp
		}
		scopeChatroom = ""

	default:
		// 优先级：@ 提人 wxid > 指定名字 > 未指定则默认查发起人自己
		switch {
		case atWxid != "":
			memberWxid = atWxid
			displayName = targetName
			if mem, ok := p.findMemberByWxid(chatroomWxid, atWxid); ok {
				displayName = displayNameOf(mem)
			}
		case targetName == "":
			// 群内未指定成员：默认查发起人自己
			memberWxid = issuer.GetUsername()
			displayName = displayNameOf(issuer)
		default:
			mem, ok := p.findMember(chatroomWxid, targetName)
			if !ok {
				return "未在当前群找到成员：" + targetName + "（可尝试 @ 该成员）", nil
			}
			memberWxid = mem.GetUsername()
			displayName = displayNameOf(mem)
		}
		scopeChatroom = chatroomWxid
	}

	// 主人画像保护：仅主人可查主人的画像
	if owner != nil && memberWxid == owner.GetUsername() && issuer.GetUsername() != owner.GetUsername() {
		return "无权查看该成员的画像", nil
	}

	return p.generate(scopeChatroom, memberWxid, displayName, opts.rebuild)
}

// resolveChatroom 按群名 / 群 wxid 解析群聊，返回群 wxid 与展示名。
// 依次尝试：直接 wxid（@chatroom 结尾）→ 联系人缓存精确匹配（群名/备注）
// → 包含匹配（唯一命中才采用，多个命中提示写全名）。
// 按名字解析要求该群已在联系人缓存中（通常为已保存到通讯录的群）。
func (p *ProfilePlugin) resolveChatroom(key string) (wxid, display string, err error) {
	key = strings.TrimSpace(key)
	if strings.HasSuffix(key, "@chatroom") {
		display = key
		if p.contact != nil {
			if c := p.contact.Get(key); c != nil && c.GetUsername() != "" {
				display = displayNameOf(c)
			}
		}
		return key, display, nil
	}
	if p.contact == nil {
		return "", "", fmt.Errorf("未找到群聊：%s（contact 能力未注入）", key)
	}
	var fuzzy []*contact.Contact
	for _, c := range p.contact.List() {
		if c == nil || !strings.HasSuffix(c.GetUsername(), "@chatroom") {
			continue
		}
		nick, remark := strings.TrimSpace(c.GetNickname()), strings.TrimSpace(c.GetRemark())
		if strings.EqualFold(nick, key) || strings.EqualFold(remark, key) {
			return c.GetUsername(), displayNameOf(c), nil
		}
		if strings.Contains(nick, key) || strings.Contains(remark, key) {
			fuzzy = append(fuzzy, c)
		}
	}
	switch len(fuzzy) {
	case 0:
		return "", "", fmt.Errorf("未找到群聊：%s（群需已保存到通讯录，或直接使用群 wxid）", key)
	case 1:
		return fuzzy[0].GetUsername(), displayNameOf(fuzzy[0]), nil
	default:
		names := make([]string, 0, 6)
		for i, c := range fuzzy {
			if i >= 5 {
				names = append(names, "…")
				break
			}
			names = append(names, displayNameOf(c))
		}
		return "", "", fmt.Errorf("「%s」匹配到多个群聊：%s，请使用更完整的群名", key, strings.Join(names, "、"))
	}
}

// resolveGlobal 按昵称/备注/用户名在全局联系人缓存中解析成员 wxid
func (p *ProfilePlugin) resolveGlobal(name string) (wxid, display string, err error) {
	if p.contact == nil {
		return "", "", fmt.Errorf("全局未找到成员：%s（contact 能力未注入）", name)
	}
	for _, key := range []string{"nickname::" + name, "remark::" + name, "username::" + name} {
		c := p.contact.Get(key)
		if c != nil && c.GetUsername() != "" {
			return c.GetUsername(), displayNameOf(c), nil
		}
	}
	return "", "", fmt.Errorf("全局未找到成员：%s（非好友无法全局查询，可到其所在群内发「人物画像 %s」，或私聊主人加「#群名」在指定群内匹配）", name, name)
}

// findMember 在当前群成员列表中按群显示名/昵称/备注/用户名匹配。
func (p *ProfilePlugin) findMember(chatroomWxid, name string) (*chatroom.Member, bool) {
	if p.chatroom == nil {
		return nil, false
	}
	name = strings.TrimSpace(name)
	for _, m := range p.chatroom.ListMembers(chatroomWxid) {
		if m == nil {
			continue
		}
		for _, v := range []string{m.GetDisplayName(), m.GetRemark(), m.GetNickname(), m.GetAlias(), m.GetUsername()} {
			if strings.EqualFold(strings.TrimSpace(v), name) {
				return m, true
			}
		}
	}
	return nil, false
}

// findMemberByWxid 在当前群成员列表中按 wxid 精确匹配
func (p *ProfilePlugin) findMemberByWxid(chatroomWxid, wxid string) (*chatroom.Member, bool) {
	if p.chatroom == nil {
		return nil, false
	}
	for _, m := range p.chatroom.ListMembers(chatroomWxid) {
		if m != nil && m.GetUsername() == wxid {
			return m, true
		}
	}
	return nil, false
}

func isSelfName(sender named, name string) bool {
	name = strings.TrimSpace(name)
	for _, v := range []string{sender.GetRemark(), sender.GetNickname(), sender.GetAlias(), sender.GetUsername()} {
		if strings.EqualFold(strings.TrimSpace(v), name) {
			return true
		}
	}
	return false
}

// generate 读取历史发言 → 切块 → 调用 ai.chat → 合并/冷启动 → 持久化
func (p *ProfilePlugin) generate(scopeChatroom, memberWxid, displayName string, rebuild bool) (string, error) {
	if p.store == nil {
		return "", errStoreNotReady
	}
	cfg := normalizeConfig(p.Config)
	rec, exists := p.store.loadProfile(scopeChatroom, memberWxid)

	sinceID := int64(0)
	limit := 0
	if exists && !rebuild {
		sinceID = rec.LastMsgID
	} else {
		limit = cfg.ColdStartMaxMessages // 冷启动安全天花板
	}

	msgs, err := p.queryHistory(scopeChatroom, memberWxid, sinceID, limit)
	if err != nil {
		return "", fmt.Errorf("查询历史发言失败: %w", err)
	}
	if len(msgs) == 0 {
		if exists && strings.TrimSpace(rec.Profile) != "" {
			// 已有画像且无新发言：直接把已有画像发出来，不空跑一次 AI
			return rec.Profile, nil
		}
		if exists {
			return "该成员暂无新发言，且尚无已生成的画像", nil
		}
		return "该成员暂无可分析的发言记录", nil
	}

	chunks := splitIntoChunks(msgs, cfg.ChunkTokenBudget, cfg.MaxSingleMsgChars)
	if len(chunks) > cfg.ColdStartMaxChunks {
		chunks = keepRecentChunks(chunks, cfg.ColdStartMaxChunks)
	}
	// 水位线必须基于「实际送入模型的块」计算，否则采样丢弃的块会被永久跳过
	coveredID := maxChunkID(chunks)

	observations := make([]string, 0, len(chunks))
	for _, ch := range chunks {
		obs, err := p.callAIChunk(ch)
		if err != nil {
			return "", fmt.Errorf("分析发言片段失败: %w", err)
		}
		observations = append(observations, obs)
	}

	quant := summarizeQuant(msgs)
	var final string
	if exists && !rebuild {
		final, err = p.callAIMerge(displayName, rec.Profile, observations, quant, cfg.RenderImage)
	} else {
		final, err = p.callAIMerge(displayName, "", observations, quant, cfg.RenderImage)
	}
	if err != nil {
		return "", fmt.Errorf("生成画像失败: %w", err)
	}
	if strings.TrimSpace(final) == "" {
		return "", fmt.Errorf("模型返回空画像（可能上下文不足或调用异常）")
	}

	if err := p.store.saveProfile(profileRecord{
		Chatroom:  scopeChatroom,
		Member:    memberWxid,
		Profile:   final,
		LastMsgID: coveredID,
	}); err != nil {
		return "", fmt.Errorf("保存画像失败: %w", err)
	}
	slog.Info("[profile] 画像已生成", "scope", scopeChatroom, "member", memberWxid, "chunks", len(chunks))
	return final, nil
}

// queryHistory 经跨插件调用 statistics.query_messages 能力取历史发言（不直接读 statistics.db）
func (p *ProfilePlugin) queryHistory(scopeChatroom, memberWxid string, sinceID int64, limit int) ([]historyMsg, error) {
	if p.caller == nil {
		return nil, fmt.Errorf("调用能力未注入（需要 statistics 插件提供 statistics.query_messages）")
	}
	args := map[string]string{
		"member":   memberWxid,
		"since_id": strconv.FormatInt(sinceID, 10),
	}
	if scopeChatroom != "" {
		args["chatroom"] = scopeChatroom
	}
	if limit > 0 {
		args["limit"] = strconv.Itoa(limit)
	}
	mime, data, err := p.caller.CallPlugin("statistics.query_messages", args)
	if err != nil {
		return nil, err
	}
	_ = mime
	var msgs []historyMsg
	if err := json.Unmarshal(data, &msgs); err != nil {
		return nil, fmt.Errorf("解析历史发言失败: %w", err)
	}
	return msgs, nil
}

// callAIChunk 对单块历史发言产出局部观察
func (p *ProfilePlugin) callAIChunk(ch []historyMsg) (string, error) {
	payload, err := json.Marshal(aiChatPayload{
		System:   systemColdChunk,
		Messages: []chatMessage{{Role: "user", Content: formatChunk(ch)}},
	})
	if err != nil {
		return "", err
	}
	return p.callAI(string(payload))
}

// callAIMerge 合并已有画像与新增观察，产出完整画像。renderImage 决定系统提示要求 LLM 输出 markdown 还是纯文本。
func (p *ProfilePlugin) callAIMerge(displayName, existing string, observations []string, quant string, renderImage bool) (string, error) {
	user := buildMergeUserContent(displayName, existing, observations, quant)
	payload, err := json.Marshal(aiChatPayload{
		System:   systemMergeForRender(renderImage),
		Messages: []chatMessage{{Role: "user", Content: user}},
	})
	if err != nil {
		return "", err
	}
	return p.callAI(string(payload))
}

// callAI 经跨插件调用使用 ai 插件的 ai.chat 能力
func (p *ProfilePlugin) callAI(payload string) (string, error) {
	if p.caller == nil {
		return "", fmt.Errorf("调用能力未注入（需要 ai 插件提供 ai.chat）")
	}
	mime, data, err := p.caller.CallPlugin("ai.chat", map[string]string{"payload": payload})
	if err != nil {
		return "", err
	}
	_ = mime
	return strings.TrimSpace(string(data)), nil
}

// summarizeQuant 基于本次拉取的历史发言生成量化指标
func summarizeQuant(msgs []historyMsg) string {
	if len(msgs) == 0 {
		return "无"
	}
	total := 0
	for _, m := range msgs {
		total += len([]rune(m.Content))
	}
	first, last := msgs[0].Timestamp, msgs[len(msgs)-1].Timestamp
	if first == "" {
		first = "未知"
	}
	if last == "" {
		last = "未知"
	}
	return fmt.Sprintf("发言条数: %d\n时间跨度: %s ~ %s\n平均每条字数: %d",
		len(msgs), first, last, total/len(msgs))
}
