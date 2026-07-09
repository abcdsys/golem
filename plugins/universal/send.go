package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/sbgayhub/golem/sdk/contact"
	"github.com/sbgayhub/golem/sdk/message"
)

func (p *UniversalPlugin) sendResult(receiver *contact.Contact, sendType string, result executeResult, mentions []mentionTarget) error {
	if p.message == nil {
		return errors.New("message ability is not injected")
	}

	msg, err := buildMessage(receiver, sendType, result, mentions)
	if err != nil {
		return err
	}
	_, err = p.message.Send(msg)
	return err
}

func buildMessage(receiver *contact.Contact, sendType string, result executeResult, mentions []mentionTarget) (*message.Message, error) {
	if receiver == nil {
		return nil, errors.New("receiver is empty")
	}

	mediaType := normalizeSendType(sendType)
	if mediaType == "text" {
		if strings.TrimSpace(result.text) == "" {
			return nil, errors.New("result is empty")
		}
		content, reminds := applyMentionPrefix(result.text, mentions)
		return &message.Message{
			Receiver: receiver,
			Content:  content,
			Type:     message.TypeText,
			Data:     &message.Message_Text{Text: &message.TextData{Content: content, Reminds: reminds}},
		}, nil
	}

	if len(result.mediaData) == 0 {
		return nil, errors.New(mediaType + " data is empty")
	}
	msg := &message.Message{
		Receiver: receiver,
		Content:  "",
	}
	switch mediaType {
	case "image":
		msg.Type = message.TypeImage
		msg.Data = &message.Message_Image{Image: &message.ImageData{Media: &message.Media{Data: result.mediaData}}}
	case "video":
		duration, thumb, err := extractVideoMeta(result.mediaData)
		if err != nil {
			return nil, err
		}
		msg.Type = message.TypeVideo
		msg.Data = &message.Message_Video{Video: &message.VideoData{
			Media:    &message.Media{Data: result.mediaData},
			Duration: duration,
			Thumb:    thumb,
		}}
	case "emoji":
		msg.Type = message.TypeEmoji
		msg.Data = &message.Message_Emoji{Emoji: &message.EmojiData{Media: &message.Media{Data: result.mediaData}}}
	default:
		return nil, fmt.Errorf("unsupported send_type: %s", sendType)
	}
	return msg, nil
}

func applyMentionPrefix(result string, mentions []mentionTarget) (string, []string) {
	prefixes := make([]string, 0, len(mentions))
	reminds := make([]string, 0, len(mentions))
	for _, mention := range mentions {
		displayName := strings.TrimSpace(strings.TrimPrefix(mention.DisplayName, "@"))
		username := strings.TrimSpace(mention.Username)
		if displayName == "" || username == "" {
			continue
		}
		prefixes = append(prefixes, "@"+displayName)
		reminds = append(reminds, username)
	}
	if len(prefixes) == 0 {
		return result, nil
	}
	return strings.Join(prefixes, " ") + " " + result, reminds
}
