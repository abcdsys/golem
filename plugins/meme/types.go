package main

type memeInfo struct {
	Key    string `json:"key"`
	Params struct {
		MinImages    int      `json:"min_images"`
		MaxImages    int      `json:"max_images"`
		MinTexts     int      `json:"min_texts"`
		MaxTexts     int      `json:"max_texts"`
		DefaultTexts []string `json:"default_texts"`
	} `json:"params"`
	Keywords []string `json:"keywords"`
}

type userInfo struct {
	Avatar   string
	Nickname string
}

type record struct {
	name    string
	avatar  string
	content string
	time    string
}
