package server

import (
	"context"

	"fysj.net/v2/dao"
	"fysj.net/v2/models"
	"fysj.net/v2/pb"
)

func PushProxyInfo(ctx context.Context, req *pb.PushProxyInfoReq) (*pb.PushProxyInfoResp, error) {
	var srv *models.ServerEntity
	var err error

	if srv, err = ValidateServerRequest(req.GetBase()); err != nil {
		return nil, err
	}

	if err = dao.AdminUpdateProxyStats(srv, req.GetProxyInfos()); err != nil {
		return nil, err
	}
	return &pb.PushProxyInfoResp{
		Status: &pb.Status{Code: pb.RespCode_RESP_CODE_SUCCESS, Message: "ok"},
	}, nil
}
