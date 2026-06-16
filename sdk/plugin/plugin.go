package plugin

import (
	"errors"
	"log/slog"
	"os"
	"os/exec"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/phsym/console-slog"
)

var clients = make(map[string]*plugin.Client)

// 插件握手配置
var handshakeConfig = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "golem",
	MagicCookieValue: "golem",
}

func Start(p Plugin) {
	clientLogger := hclog.New(&hclog.LoggerOptions{
		Level:      hclog.Debug,
		JSONFormat: true,
	})

	slog.SetDefault(slog.New(Adapt(clientLogger)))
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig:  handshakeConfig,
		TLSProvider:      nil,
		Plugins:          map[string]plugin.Plugin{"plugin": &wrapper{impl: p}},
		VersionedPlugins: nil,
		GRPCServer:       plugin.DefaultGRPCServer,
		Logger:           clientLogger,
		Test:             nil,
	})
}

func Get(path string) (*Metadata, *Plugin, error) {
	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig:  handshakeConfig,
		Plugins:          plugin.PluginSet{"plugin": &wrapper{}},
		VersionedPlugins: map[int]plugin.PluginSet{},
		Cmd:              exec.Command(path),
		//Reattach:         &plugin.ReattachConfig{},
		//RunnerFunc: func(l hclog.Logger, cmd *exec.Cmd, tmpDir string) (runner.Runner, error) {
		//	panic("TODO")
		//},
		//SecureConfig:        &plugin.SecureConfig{},
		//TLSConfig:           &tls.Config{},
		//Managed:             false,
		//MinPort:             0,
		//MaxPort:             0,
		//StartTimeout:        0,
		//Stderr:              nil,
		//SyncStdout:          nil,
		//SyncStderr:          nil,
		AllowedProtocols: []plugin.Protocol{plugin.ProtocolGRPC},
		Logger:           creatManagerLogger(),
		//PluginLogBufferSize: 0,
		//AutoMTLS:            false,
		//GRPCDialOptions:     []grpc.DialOption{},
		//GRPCBrokerMultiplex: false,
		//SkipHostEnv:         false,
		//UnixSocketConfig:    &plugin.UnixSocketConfig{},
	})

	// 连接插件
	rpc, err := client.Client()
	if err != nil {
		return nil, nil, err
	}

	// 获取插件实例
	raw, err := rpc.Dispense("plugin")
	if err != nil {
		return nil, nil, err
	}

	// 类型断言获取插件接口
	p, ok := raw.(Plugin)
	if !ok {
		return nil, nil, errors.New("plugin does not implement Plugin interface")
	}

	metadata := p.GetMetadata()
	if metadata == nil {
		return nil, nil, errors.New("获取插件信息失败，元数据为nil")
	}

	clients[metadata.Name] = client

	return metadata, &p, nil
}

func Kill(name string) {
	if client, ok := clients[name]; ok {
		client.Kill()
		delete(clients, name)
	}
	slog.Debug("插件退出", "name", name)
}

func creatManagerLogger() hclog.Logger {
	levelVar := slog.LevelVar{}
	//levelVar.Set(slog.Level(4))
	levelVar.Set(slog.LevelDebug)
	logger := slog.New(console.NewHandler(os.Stderr, &console.HandlerOptions{Level: slog.LevelDebug}))
	//slog.SetDefault(logger)
	return newLogger(logger, &levelVar)
}
