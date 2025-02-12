package server

import (
	"context"

	"fysj.net/v2/logger"
	"fysj.net/v2/pb"
	"fysj.net/v2/rpc"
	"fysj.net/v2/tunnel"
	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/samber/lo"
)

var proxyTypeList = []v1.ProxyType{
	v1.ProxyTypeTCP,
	v1.ProxyTypeUDP,
	v1.ProxyTypeTCPMUX,
	v1.ProxyTypeHTTP,
	v1.ProxyTypeHTTPS,
	v1.ProxyTypeSTCP,
	v1.ProxyTypeXTCP,
	v1.ProxyTypeSUDP,
}

func PushProxyInfo(serverID, serverSecret string) error {
	proxyInfos := []*pb.ProxyInfo{}

	if cli := tunnel.GetServerController().Get(serverID); cli != nil {
		firstSync := cli.IsFirstSync()
		for _, proxyType := range proxyTypeList {
			proxyStatsList := cli.GetProxyStatsByType(proxyType)
			for _, proxyStats := range proxyStatsList {
				if proxyStats == nil {
					continue
				}
				proxyInfos = append(proxyInfos, &pb.ProxyInfo{
					Name:            lo.ToPtr(proxyStats.Name),
					Type:            lo.ToPtr(proxyStats.Type),
					TodayTrafficIn:  lo.ToPtr(proxyStats.TodayTrafficIn),
					TodayTrafficOut: lo.ToPtr(proxyStats.TodayTrafficOut),
					FirstSync:       lo.ToPtr(firstSync),
				})
			}
		}
	}

	if len(proxyInfos) > 0 {
		ctx := context.Background()
		cli, err := rpc.MasterCli(ctx)
		if err != nil {
			logger.Logger(context.Background()).WithError(err).Error("cannot get master server")
			return err
		}
		_, err = cli.PushProxyInfo(ctx, &pb.PushProxyInfoReq{
			Base: &pb.ServerBase{
				ServerId:     serverID,
				ServerSecret: serverSecret,
			},
			ProxyInfos: proxyInfos,
		})
		if err != nil {
			logger.Logger(context.Background()).WithError(err).Error("cannot push proxy info")
			return err
		}
	}
	return nil
}
