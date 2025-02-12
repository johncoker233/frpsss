package client

import (
	"context"

	"fysj.net/v2/common"
	"fysj.net/v2/dao"
	"fysj.net/v2/logger"
	"fysj.net/v2/models"
	"fysj.net/v2/pb"
	"github.com/samber/lo"
)

func ListClientsHandler(ctx context.Context, req *pb.ListClientsRequest) (*pb.ListClientsResponse, error) {
	logger.Logger(ctx).Infof("list client, req: [%+v]", req)

	var (
		userInfo = common.GetUserInfo(ctx)
	)

	if !userInfo.Valid() {
		return &pb.ListClientsResponse{
			Status: &pb.Status{Code: pb.RespCode_RESP_CODE_INVALID, Message: "invalid user"},
		}, nil
	}

	var (
		page         = int(req.GetPage())
		pageSize     = int(req.GetPageSize())
		keyword      = req.GetKeyword()
		clients      []*models.ClientEntity
		err          error
		clientCounts int64
		hasKeyword   = len(keyword) > 0
	)

	if hasKeyword {
		clients, err = dao.ListClientsWithKeyword(userInfo, page, pageSize, keyword)
	} else {
		clients, err = dao.ListClients(userInfo, page, pageSize)
	}

	if err != nil {
		return nil, err
	}

	if hasKeyword {
		clientCounts, err = dao.CountClientsWithKeyword(userInfo, keyword)
	} else {
		clientCounts, err = dao.CountClients(userInfo)
	}

	if err != nil {
		return nil, err
	}

	respClients := lo.Map(clients, func(c *models.ClientEntity, _ int) *pb.Client {
		clientIDs, err := dao.GetClientIDsInShadowByClientID(userInfo, c.ClientID)
		if err != nil {
			logger.Logger(ctx).Errorf("get client ids in shadow by client id error: %v", err)
		}

		return &pb.Client{
			Id:        lo.ToPtr(c.ClientID),
			Secret:    lo.ToPtr(c.ConnectSecret),
			Config:    lo.ToPtr(string(c.ConfigContent)),
			ServerId:  lo.ToPtr(c.ServerID),
			Stopped:   lo.ToPtr(c.Stopped),
			Comment:   lo.ToPtr(c.Comment),
			ClientIds: clientIDs,
		}
	})

	return &pb.ListClientsResponse{
		Status:  &pb.Status{Code: pb.RespCode_RESP_CODE_SUCCESS, Message: "ok"},
		Clients: respClients,
		Total:   lo.ToPtr(int32(clientCounts)),
	}, nil
}
