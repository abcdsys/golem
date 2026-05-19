package login

import "context"

// Client 实现 Ability 接口，通过 gRPC 调用远程登录服务
type Client struct {
	Client LoginServiceClient
}

var _ Ability = (*Client)(nil)

// Login 执行扫码登录
func (c Client) Login() (*QRCodeResult, error) {
	return c.Client.Login(context.Background(), &LoginRequest{})
}

// Init 首次登录后初始化
func (c Client) Init() (*InitResponse, error) {
	return c.Client.Init(context.Background(), &InitRequest{})
}

// Refresh 刷新登录状态
func (c Client) Refresh() (*OperateResponse, error) {
	return c.Client.Refresh(context.Background(), &LoginRequest{})
}

// Wakeup 唤醒登录
func (c Client) Wakeup() (*OperateResponse, error) {
	return c.Client.Wakeup(context.Background(), &LoginRequest{})
}

// Logout 登出
func (c Client) Logout() (*OperateResponse, error) {
	return c.Client.Logout(context.Background(), &LoginRequest{})
}

// PasswordLogin 使用账号密码登录
func (c Client) PasswordLogin(req *PasswordLoginRequest) (*PasswordLoginResult, error) {
	return c.Client.PasswordLogin(context.Background(), req)
}

// Server 实现 LoginServiceServer 接口，将 gRPC 请求委托给 Ability 实现
type Server struct {
	UnimplementedLoginServiceServer
	Impl Ability
}

// Login 执行扫码登录
func (s Server) Login(ctx context.Context, request *LoginRequest) (*QRCodeResult, error) {
	return s.Impl.Login()
}

// Init 首次登录后初始化
func (s Server) Init(ctx context.Context, request *InitRequest) (*InitResponse, error) {
	return s.Impl.Init()
}

// Refresh 刷新登录状态
func (s Server) Refresh(ctx context.Context, request *LoginRequest) (*OperateResponse, error) {
	return s.Impl.Refresh()
}

// Wakeup 唤醒登录
func (s Server) Wakeup(ctx context.Context, request *LoginRequest) (*OperateResponse, error) {
	return s.Impl.Wakeup()
}

// Logout 登出
func (s Server) Logout(ctx context.Context, request *LoginRequest) (*OperateResponse, error) {
	return s.Impl.Logout()
}

// PasswordLogin 使用账号密码登录
func (s Server) PasswordLogin(ctx context.Context, request *PasswordLoginRequest) (*PasswordLoginResult, error) {
	return s.Impl.PasswordLogin(request)
}
