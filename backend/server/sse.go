package server

import (
	"github.com/liujitcn/kratos-kit/bootstrap"
	sseServer "github.com/liujitcn/kratos-kit/transport/sse"
	"github.com/liujitcn/kratos-kit/utils"
)

// NewSseServer 创建 SSE Server 并加载 server.sse 配置。
func NewSseServer(ctx *bootstrap.Context) (*sseServer.Server, error) {
	cfg := ctx.GetConfig()

	// 未配置 SSE 时不创建对应传输服务。
	if cfg == nil || cfg.Server == nil || cfg.Server.Sse == nil {
		return nil, nil
	}

	l := ctx.NewLoggerHelper("sse-server/admin-service")
	sseCfg := cfg.Server.Sse

	options := []sseServer.ServerOption{
		sseServer.WithAutoStream(sseCfg.GetAutoStream()),
		sseServer.WithAutoReply(sseCfg.GetAutoReply()),
		sseServer.WithSplitData(sseCfg.GetSplitData()),
		sseServer.WithEncodeBase64(sseCfg.GetEncodeBase64()),
		sseServer.WithSubscriberFunction(func(streamID sseServer.StreamID, sub *sseServer.Subscriber) {
			// 订阅者连入时记录流 ID，便于排查前端长连接问题。
			l.Infof("subscriber [%s] connected", streamID)
		}),
	}

	if sseCfg.GetNetwork() != "" {
		options = append(options, sseServer.WithNetwork(sseCfg.GetNetwork()))
	}
	if sseCfg.GetAddr() != "" {
		options = append(options, sseServer.WithAddress(sseCfg.GetAddr()))
	}
	if sseCfg.GetPath() != "" {
		options = append(options, sseServer.WithPath(sseCfg.GetPath()))
	}
	if sseCfg.GetCodec() != "" {
		options = append(options, sseServer.WithCodec(sseCfg.GetCodec()))
	}
	if sseCfg.GetTimeout() != nil {
		options = append(options, sseServer.WithTimeout(sseCfg.GetTimeout().AsDuration()))
	}
	if sseCfg.GetEventTtl() != nil {
		options = append(options, sseServer.WithEventTTL(sseCfg.GetEventTtl().AsDuration()))
	}
	if sseCfg.GetTls() != nil {
		tlsCfg, err := utils.LoadServerTlsConfig(sseCfg.GetTls())
		if err != nil {
			return nil, err
		}
		if tlsCfg != nil {
			options = append(options, sseServer.WithTLSConfig(tlsCfg))
		}
	}

	return sseServer.NewServer(options...), nil
}
