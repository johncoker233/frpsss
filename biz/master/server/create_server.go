package server

import (
	"context"

	"fysj.net/v2/common"
	"fysj.net/v2/dao"
	"fysj.net/v2/models"
	"fysj.net/v2/pb"
	"fysj.net/v2/utils"
	"github.com/google/uuid"
)

func InitServerHandler(c context.Context, req *pb.InitServerRequest) (*pb.InitServerResponse, error) {
	var (
		userServerID = req.GetServerId()
		serverIP     = req.GetServerIp()
		userInfo     = common.GetUserInfo(c)
	)

	if !userInfo.Valid() {
		return &pb.InitServerResponse{
			Status: &pb.Status{Code: pb.RespCode_RESP_CODE_INVALID, Message: "invalid user"},
		}, nil
	}
	if !userInfo.IsAdmin() {
		return &pb.InitServerResponse{
			Status: &pb.Status{Code: pb.RespCode_RESP_CODE_UNAUTHORIZED, Message: "permission denied: admin role required"},
		}, nil
	}
	if len(userServerID) == 0 || len(serverIP) == 0 || !utils.IsClientIDPermited(userServerID) {
		return &pb.InitServerResponse{
			Status: &pb.Status{Code: pb.RespCode_RESP_CODE_INVALID, Message: "request invalid"},
		}, nil
	}

	globalServerID := common.GlobalClientID(userInfo.GetUserName(), "s", userServerID)

	if err := dao.CreateServer(userInfo,
		&models.ServerEntity{
			ServerID:      globalServerID,
			TenantID:      userInfo.GetTenantID(),
			UserID:        userInfo.GetUserID(),
			ConnectSecret: uuid.New().String(),
			ServerIP:      serverIP,
		}); err != nil {
		return nil, err
	}

	return &pb.InitServerResponse{
		Status:   &pb.Status{Code: pb.RespCode_RESP_CODE_SUCCESS, Message: "ok"},
		ServerId: &globalServerID,
	}, nil
}
