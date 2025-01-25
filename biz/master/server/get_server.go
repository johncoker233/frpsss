package server

import (
	"context"

	"fysj.net/v2/common"
	"fysj.net/v2/dao"
	"fysj.net/v2/pb"
	"github.com/samber/lo"
)

func GetServerHandler(c context.Context, req *pb.GetServerRequest) (*pb.GetServerResponse, error) {
	var (
		userServerID = req.GetServerId()
		userInfo     = common.GetUserInfo(c)
	)

	if !userInfo.Valid() {
		return &pb.GetServerResponse{
			Status: &pb.Status{Code: pb.RespCode_RESP_CODE_INVALID, Message: "invalid user"},
		}, nil
	}
	if !userInfo.IsAdmin() {
		return &pb.GetServerResponse{
			Status: &pb.Status{Code: pb.RespCode_RESP_CODE_UNAUTHORIZED, Message: "permission denied: admin role required"},
		}, nil
	}
	if len(userServerID) == 0 {
		return &pb.GetServerResponse{
			Status: &pb.Status{Code: pb.RespCode_RESP_CODE_INVALID, Message: "invalid client id"},
		}, nil
	}

	serverEntity, err := dao.GetServerByServerID(userInfo, userServerID)
	if err != nil {
		return nil, err
	}

	return &pb.GetServerResponse{
		Status: &pb.Status{Code: pb.RespCode_RESP_CODE_SUCCESS, Message: "ok"},
		Server: &pb.Server{
			Id:      lo.ToPtr(serverEntity.ServerID),
			Config:  lo.ToPtr(string(serverEntity.ConfigContent)),
			Secret:  lo.ToPtr(serverEntity.ConnectSecret),
			Comment: lo.ToPtr(serverEntity.Comment),
			Ip:      lo.ToPtr(serverEntity.ServerIP),
		},
	}, nil
}
