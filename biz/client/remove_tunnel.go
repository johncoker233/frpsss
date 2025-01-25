package client

import (
	"context"
	"os"
	"time"

	"fysj.net/v2/logger"
	"fysj.net/v2/pb"
)

func RemoveFrpcHandler(ctx context.Context, req *pb.RemoveFRPCRequest) (*pb.RemoveFRPCResponse, error) {
	logger.Logger(ctx).Infof("remove frpc, req: [%+v], will exit in 10s", req)

	go func() {
		time.Sleep(10 * time.Second)
		os.Exit(0)
	}()

	return &pb.RemoveFRPCResponse{
		Status: &pb.Status{Code: pb.RespCode_RESP_CODE_SUCCESS, Message: "ok"},
	}, nil
}
