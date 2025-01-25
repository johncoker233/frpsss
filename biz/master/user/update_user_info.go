package user

import (
	"context"

	"fysj.net/v2/biz/master/client"
	"fysj.net/v2/common"
	"fysj.net/v2/dao"
	"fysj.net/v2/logger"
	"fysj.net/v2/models"
	"fysj.net/v2/pb"
	"fysj.net/v2/utils"
)

func UpdateUserInfoHander(c context.Context, req *pb.UpdateUserInfoRequest) (*pb.UpdateUserInfoResponse, error) {
	var (
		userInfo = common.GetUserInfo(c)
	)

	if !userInfo.Valid() {
		return &pb.UpdateUserInfoResponse{
			Status: &pb.Status{Code: pb.RespCode_RESP_CODE_INVALID, Message: "invalid user"},
		}, nil
	}
	newUserEntity := userInfo.(*models.UserEntity)
	newUserInfo := req.GetUserInfo()

	if newUserInfo.GetEmail() != "" {
		newUserEntity.Email = newUserInfo.GetEmail()
	}

	if newUserInfo.GetRawPassword() != "" {
		hashedPassword, err := utils.HashPassword(newUserInfo.GetRawPassword())
		if err != nil {
			logger.Logger(context.Background()).WithError(err).Errorf("cannot hash password")
			return nil, err
		}
		newUserEntity.Password = hashedPassword
	}

	if newUserInfo.GetUserName() != "" {
		newUserEntity.UserName = newUserInfo.GetUserName()
	}

	if newUserInfo.GetToken() != "" {
		newUserEntity.Token = newUserInfo.GetToken()
	}

	if err := dao.UpdateUser(userInfo, newUserEntity); err != nil {
		return &pb.UpdateUserInfoResponse{
			Status: &pb.Status{Code: pb.RespCode_RESP_CODE_INVALID, Message: err.Error()},
		}, err
	}

	go func() {
		newUser, err := dao.GetUserByUserID(userInfo.GetUserID())
		if err != nil {
			logger.Logger(context.Background()).WithError(err).Errorf("cannot get user")
			return
		}

		if err := client.SyncTunnel(c, newUser); err != nil {
			logger.Logger(context.Background()).WithError(err).Errorf("cannot sync tunnel, user need to retry update")
		}
	}()

	return &pb.UpdateUserInfoResponse{
		Status: &pb.Status{Code: pb.RespCode_RESP_CODE_SUCCESS, Message: "ok"},
	}, nil
}
