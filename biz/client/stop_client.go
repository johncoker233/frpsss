package client

import (
	"context"

	"fysj.net/v2/logger"
	"fysj.net/v2/pb"
	"fysj.net/v2/tunnel"
)

func StopFRPCHandler(ctx context.Context, req *pb.StopFRPCRequest) (*pb.StopFRPCResponse, error) {
	logger.Logger(ctx).Infof("client get a stop client request, origin is: [%+v]", req)

	tunnel.GetClientController().StopAll()
	tunnel.GetClientController().DeleteAll()

	return &pb.StopFRPCResponse{
		Status: &pb.Status{Code: pb.RespCode_RESP_CODE_SUCCESS, Message: "ok"},
	}, nil
}
