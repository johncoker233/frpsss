package client

import (
	"context"

	"fysj.net/v2/common"
	"fysj.net/v2/dao"
	"fysj.net/v2/logger"
	"fysj.net/v2/pb"
	"fysj.net/v2/rpc"
)

func DeleteClientHandler(ctx context.Context, req *pb.DeleteClientRequest) (*pb.DeleteClientResponse, error) {
	logger.Logger(ctx).Infof("delete client, req: [%+v]", req)

	userInfo := common.GetUserInfo(ctx)
	clientID := req.GetClientId()

	if !userInfo.Valid() {
		return &pb.DeleteClientResponse{
			Status: &pb.Status{Code: pb.RespCode_RESP_CODE_INVALID, Message: "invalid user"},
		}, nil
	}

	if len(clientID) == 0 {
		return &pb.DeleteClientResponse{
			Status: &pb.Status{Code: pb.RespCode_RESP_CODE_INVALID, Message: "invalid client id"},
		}, nil
	}

	if err := dao.DeleteClient(userInfo, clientID); err != nil {
		return nil, err
	}

	if err := dao.DeleteProxyConfigsByClientIDOrOriginClientID(userInfo, clientID); err != nil {
		return nil, err
	}

	go func() {
		resp, err := rpc.CallClient(context.Background(), req.GetClientId(), pb.Event_EVENT_REMOVE_FRPC, req)
		if err != nil {
			logger.Logger(context.Background()).WithError(err).Errorf("remove event send to client error, client id: [%s]", req.GetClientId())
		}

		if resp == nil {
			logger.Logger(ctx).Errorf("cannot get response, client id: [%s]", req.GetClientId())
		}
	}()

	return &pb.DeleteClientResponse{
		Status: &pb.Status{Code: pb.RespCode_RESP_CODE_SUCCESS, Message: "ok"},
	}, nil
}
