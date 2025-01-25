package client

import (
	"context"

	"fysj.net/v2/biz/common"
	"fysj.net/v2/conf"
	"fysj.net/v2/pb"
	"fysj.net/v2/services/rpcclient"
	"fysj.net/v2/utils"
	"github.com/sirupsen/logrus"
)

func StartSteamLogHandler(ctx context.Context, req *pb.CommonRequest) (*pb.CommonResponse, error) {
	return common.StartSteamLogHandler(ctx, req, initStreamLog)
}

func StopSteamLogHandler(ctx context.Context, req *pb.CommonRequest) (*pb.CommonResponse, error) {
	return common.StopSteamLogHandler(ctx, req)
}

func initStreamLog(h *common.HookMgr) {
	clientID := conf.Get().Client.ID
	clientSecret := conf.Get().Client.Secret

	handler, err := rpcclient.GetClientRPCSerivce().GetCli().PushClientStreamLog(
		context.Background())
	if err != nil {
		logrus.Error(err)
	}

	h.AddStream(func(msg string) {
		handler.Send(&pb.PushClientStreamLogReq{
			Log: []byte(utils.EncodeBase64(msg)),
			Base: &pb.ClientBase{
				ClientId:     clientID,
				ClientSecret: clientSecret,
			},
		})
	}, func() { handler.CloseSend() })
}
