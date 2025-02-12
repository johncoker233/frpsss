package server

import (
	"context"

	"fysj.net/v2/common"
	"fysj.net/v2/dao"
	"fysj.net/v2/pb"
)

func DeleteServerHandler(c context.Context, req *pb.DeleteServerRequest) (*pb.DeleteServerResponse, error) {
	var (
		userServerID = req.GetServerId()
		userInfo     = common.GetUserInfo(c)
	)

	if !userInfo.Valid() {
		return &pb.DeleteServerResponse{
			Status: &pb.Status{Code: pb.RespCode_RESP_CODE_INVALID, Message: "invalid user"},
		}, nil
	}
	if !userInfo.IsAdmin() {
		return &pb.DeleteServerResponse{
			Status: &pb.Status{Code: pb.RespCode_RESP_CODE_UNAUTHORIZED, Message: "permission denied: admin role required"},
		}, nil
	}
	if len(userServerID) == 0 {
		return &pb.DeleteServerResponse{
			Status: &pb.Status{Code: pb.RespCode_RESP_CODE_INVALID, Message: "invalid client id"},
		}, nil
	}

	if err := dao.DeleteServer(userInfo, userServerID); err != nil {
		return nil, err
	}

	return &pb.DeleteServerResponse{
		Status: &pb.Status{Code: pb.RespCode_RESP_CODE_SUCCESS, Message: "ok"},
	}, nil
}
