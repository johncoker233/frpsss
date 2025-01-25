package client

import (
	"context"

	"fysj.net/v2/common"
	"fysj.net/v2/dao"
	"fysj.net/v2/logger"
	"fysj.net/v2/pb"
	"fysj.net/v2/rpc"
)

func StartFRPCHandler(ctx context.Context, req *pb.StartFRPCRequest) (*pb.StartFRPCResponse, error) {
	logger.Logger(ctx).Infof("master get a start client request, origin is: [%+v]", req)

	userInfo := common.GetUserInfo(ctx)
	clientID := req.GetClientId()

	if !userInfo.Valid() {
		return &pb.StartFRPCResponse{
			Status: &pb.Status{Code: pb.RespCode_RESP_CODE_INVALID, Message: "invalid user"},
		}, nil
	}

	if len(clientID) == 0 {
		return &pb.StartFRPCResponse{
			Status: &pb.Status{Code: pb.RespCode_RESP_CODE_INVALID, Message: "invalid client id"},
		}, nil
	}

	client, err := dao.GetClientByClientID(userInfo, clientID)
	if err != nil {
		return nil, err
	}

	client.Stopped = false

	if err = dao.UpdateClient(userInfo, client); err != nil {
		return nil, err
	}

	go func() {
		resp, err := rpc.CallClient(context.Background(), req.GetClientId(), pb.Event_EVENT_START_FRPC, req)
		if err != nil {
			logger.Logger(context.Background()).WithError(err).Errorf("start client event send to client error, client id: [%s]", req.GetClientId())
		}

		if resp == nil {
			logger.Logger(ctx).Errorf("cannot get response, client id: [%s]", req.GetClientId())
		}
	}()

	return &pb.StartFRPCResponse{
		Status: &pb.Status{Code: pb.RespCode_RESP_CODE_SUCCESS, Message: "ok"},
	}, nil
}
