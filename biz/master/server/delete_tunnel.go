package server

import (
	"context"

	"fysj.net/v2/common"
	"fysj.net/v2/dao"
	"fysj.net/v2/logger"
	"fysj.net/v2/pb"
	"fysj.net/v2/rpc"
)

func RemoveFrpsHandler(c context.Context, req *pb.RemoveFRPSRequest) (*pb.RemoveFRPSResponse, error) {
	logger.Logger(c).Infof("remove frps, req: [%+v]", req)

	var (
		serverID = req.GetServerId()
		userInfo = common.GetUserInfo(c)
	)

	srv, err := dao.GetServerByServerID(userInfo, serverID)
	if srv == nil || err != nil {
		logger.Logger(context.Background()).WithError(err).Errorf("cannot get server, id: [%s]", serverID)
		return nil, err
	}

	if err = dao.DeleteServer(userInfo, serverID); err != nil {
		logger.Logger(context.Background()).WithError(err).Errorf("cannot delete server, id: [%s]", serverID)
		return nil, err
	}

	go func() {
		resp, err := rpc.CallClient(context.Background(), req.GetServerId(), pb.Event_EVENT_REMOVE_FRPS, req)
		if err != nil {
			logger.Logger(context.Background()).WithError(err).Errorf("remove event send to server error, server id: [%s]", req.GetServerId())
		}

		if resp == nil {
			logger.Logger(c).Errorf("cannot get response, server id: [%s]", req.GetServerId())
		}
	}()

	return &pb.RemoveFRPSResponse{
		Status: &pb.Status{Code: pb.RespCode_RESP_CODE_SUCCESS, Message: "ok"},
	}, nil
}
