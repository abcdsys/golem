package main

import "github.com/sbgayhub/golem/sdk/contact"

const contactTypeChatroom = contact.ContactType_CONTACT_TYPE_CHATROOM

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type incomingMessage struct {
	SessionKey   string
	Receiver     *contact.Contact
	Text         string
	IsChatroom   bool
	MentionedBot bool
	QuotedBot    bool

	ChatroomName string
	SpeakerName  string
	SpeakerID    string
	Quote        quoteInfo
}

type quoteInfo struct {
	FromUser    string
	ChatUser    string
	DisplayName string
	Content     string
}
