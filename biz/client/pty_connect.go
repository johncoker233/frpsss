package client

import (
	"context"

	"fysj.net/v2/biz/common"
	"fysj.net/v2/conf"
	"fysj.net/v2/pb"
)

func StartPTYConnect(c context.Context, req *pb.CommonRequest) (*pb.CommonResponse, error) {
	return common.StartPTYConnect(c, req, &pb.PTYClientMessage{Base: &pb.PTYClientMessage_ClientBase{
		ClientBase: &pb.ClientBase{
			ClientId:     conf.Get().Client.ID,
			ClientSecret: conf.Get().Client.Secret,
		},
	}})
}
