package server

import (
	"context"

	"fysj.net/v2/models"
	"fysj.net/v2/pb"
	"github.com/samber/lo"
)

func RPCPullConfig(ctx context.Context, req *pb.PullServerConfigReq) (*pb.PullServerConfigResp, error) {
	var cli *models.ServerEntity
	var err error

	if cli, err = ValidateServerRequest(req.GetBase()); err != nil {
		return nil, err
	}

	return &pb.PullServerConfigResp{
		Server: &pb.Server{
			Id:     lo.ToPtr(cli.ServerID),
			Config: lo.ToPtr(string(cli.ConfigContent)),
		},
	}, nil
}
