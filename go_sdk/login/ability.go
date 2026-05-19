package login

// Ability 登录能力接口（供插件嵌入使用）
type Ability interface {
	// Login 执行扫码登录
	Login() (*QRCodeResult, error)
	// Init 首次登录后初始化
	Init() (*InitResponse, error)
	// Refresh 刷新登录状态
	Refresh() (*OperateResponse, error)
	// Wakeup 唤醒登录
	Wakeup() (*OperateResponse, error)
	// Logout 登出
	Logout() (*OperateResponse, error)
	// PasswordLogin 使用账号密码登录
	PasswordLogin(req *PasswordLoginRequest) (*PasswordLoginResult, error)
}

// Instance 登录能力实例（由 host/ability 层注入）
var Instance Ability
