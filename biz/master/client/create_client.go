package client

import (
	"context"

	"fysj.net/v2/common"
	"fysj.net/v2/dao"
	"fysj.net/v2/models"
	"fysj.net/v2/pb"
	"fysj.net/v2/utils"
	"github.com/google/uuid"
)

func InitClientHandler(c context.Context, req *pb.InitClientRequest) (*pb.InitClientResponse, error) {
	userClientID := req.GetClientId()
	userInfo := common.GetUserInfo(c)

	if !userInfo.Valid() {
		return &pb.InitClientResponse{
			Status: &pb.Status{Code: pb.RespCode_RESP_CODE_INVALID, Message: "invalid user"},
		}, nil
	}

	if len(userClientID) == 0 || !utils.IsClientIDPermited(userClientID) {
		return &pb.InitClientResponse{
			Status: &pb.Status{Code: pb.RespCode_RESP_CODE_INVALID, Message: "invalid client id"},
		}, nil
	}

	globalClientID := common.GlobalClientID(userInfo.GetUserName(), "c", userClientID)

	if err := dao.CreateClient(userInfo,
		&models.ClientEntity{
			ClientID:      globalClientID,
			TenantID:      userInfo.GetTenantID(),
			UserID:        userInfo.GetUserID(),
			ConnectSecret: uuid.New().String(),
			IsShadow:      true,
		}); err != nil {
		return &pb.InitClientResponse{Status: &pb.Status{Code: pb.RespCode_RESP_CODE_INVALID, Message: err.Error()}}, err
	}

	return &pb.InitClientResponse{
		Status:   &pb.Status{Code: pb.RespCode_RESP_CODE_SUCCESS, Message: "ok"},
		ClientId: &globalClientID,
	}, nil
}
