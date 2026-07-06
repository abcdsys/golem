package main

// Config 插件配置
type Config struct {
	VideoNative bool   `toml:"video_native" comment:"视频使用 CDN 原生上传（失败则回退到链接卡片）"`
	MaxList     int    `toml:"max_list" comment:"列表类结果最大条数"`
	BdbkURL     string `toml:"bdbk_url" comment:"百度百科 API 地址（需自行配置）"`
}
