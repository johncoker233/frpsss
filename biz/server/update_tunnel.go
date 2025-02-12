package server

import (
	"context"
	"reflect"

	"fysj.net/v2/logger"
	"fysj.net/v2/pb"
	"fysj.net/v2/services/server"
	"fysj.net/v2/tunnel"
	"fysj.net/v2/utils"
)

func UpdateFrpsHander(ctx context.Context, req *pb.UpdateFRPSRequest) (*pb.UpdateFRPSResponse, error) {
	logger.Logger(ctx).Infof("update frps, req: [%+v]", req)

	content := req.GetConfig()

	s, err := utils.LoadServerConfig(content, true)
	if err != nil {
		logger.Logger(context.Background()).WithError(err).Errorf("cannot load config")
		return &pb.UpdateFRPSResponse{
			Status: &pb.Status{Code: pb.RespCode_RESP_CODE_INVALID, Message: err.Error()},
		}, err
	}

	serverID := req.GetServerId()
	if cli := tunnel.GetServerController().Get(serverID); cli != nil {
		if !reflect.DeepEqual(cli.GetCommonCfg(), s) {
			cli.Stop()
			tunnel.GetServerController().Delete(serverID)
			logger.Logger(ctx).Infof("server %s config changed, will recreate it", serverID)
		} else {
			logger.Logger(ctx).Infof("server %s config not changed", serverID)
			return &pb.UpdateFRPSResponse{
				Status: &pb.Status{Code: pb.RespCode_RESP_CODE_SUCCESS, Message: "ok"},
			}, nil
		}
	}
	tunnel.GetServerController().Add(serverID, server.NewServerHandler(s))
	tunnel.GetServerController().Run(serverID)

	return &pb.UpdateFRPSResponse{
		Status: &pb.Status{Code: pb.RespCode_RESP_CODE_SUCCESS, Message: "ok"},
	}, nil
}
