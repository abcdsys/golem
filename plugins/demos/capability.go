package main

import (
	"errors"
	"strings"

	"github.com/sbgayhub/golem/sdk/contact"
)

// GetCapabilities 声明 demos 对外开放的能力，供 cron 等插件通过 OnCall 调用。
func (p *DemosPlugin) GetCapabilities() []string {
	return []string{"demos.run"}
}

// OnCall 通过 text 参数复用 OnEvent 的文本分发逻辑，
// text 写法与用户平时发送的消息完全一致，如 "温馨提示"、"百度百科 中国"。
func (p *DemosPlugin) OnCall(capability string, args map[string]string) (string, []byte, error) {
	if capability != "demos.run" {
		return "", nil, errors.New("不支持的能力：" + capability)
	}

	receiver := strings.TrimSpace(args["receiver"])
	if receiver == "" {
		return "", nil, errors.New("receiver 不可为空")
	}
	c := p.contact.Get(receiver)
	if c == nil {
		return "", nil, errors.New("未找到联系人：" + receiver)
	}

	text := strings.TrimSpace(args["text"])
	if text == "" {
		return "", nil, errors.New("text 不可为空")
	}

	handled, err := p.dispatch(c, text)
	if err != nil {
		return "", nil, err
	}
	if !handled {
		return "", nil, errors.New("未匹配到功能：" + text)
	}
	// handler 内部已通过 sendText/sendImage 等自行发送，返回 none 让调用方不再重复发送。
	return "none", nil, nil
}

// dispatch 根据文本匹配 handler 并执行，OnEvent 与 OnCall 共用；
// 返回是否匹配到功能，handler 的错误原样上抛，兜底方式由调用方决定。
func (p *DemosPlugin) dispatch(receiver *contact.Contact, text string) (bool, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return false, nil
	}
	for _, key := range p.sortedKeys() {
		var arg string
		if text == key {
			arg = ""
		} else if strings.HasPrefix(text, key+" ") {
			arg = strings.TrimSpace(text[len(key):])
		} else {
			continue
		}
		return p.handlers[key](receiver, arg)
	}
	return false, nil
}
