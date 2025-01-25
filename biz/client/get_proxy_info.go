package client

import (
	"context"
	"fmt"

	"fysj.net/v2/logger"
	"fysj.net/v2/pb"
	"fysj.net/v2/tunnel"
	"github.com/samber/lo"
)

func GetProxyConfig(c context.Context, req *pb.GetProxyConfigRequest) (*pb.GetProxyConfigResponse, error) {
	var (
		clientID  = req.GetClientId()
		serverID  = req.GetServerId()
		proxyName = req.GetName()
	)

	ctrl := tunnel.GetClientController()
	cli := ctrl.Get(clientID, serverID)
	if cli == nil {
		logger.Logger(c).Errorf("cannot get client, clientID: [%s], serverID: [%s]", clientID, serverID)
		return nil, fmt.Errorf("cannot get client")
	}
	workingStatus, ok := cli.GetProxyStatus(proxyName)
	if !ok {
		logger.Logger(c).Errorf("cannot get proxy status, client: [%s], server: [%s], proxy name: [%s]", clientID, serverID, proxyName)
		return nil, fmt.Errorf("cannot get proxy status")
	}

	return &pb.GetProxyConfigResponse{
		Status: &pb.Status{Code: pb.RespCode_RESP_CODE_SUCCESS, Message: "success"},
		WorkingStatus: &pb.ProxyWorkingStatus{
			Name:       lo.ToPtr(workingStatus.Name),
			Type:       lo.ToPtr(workingStatus.Type),
			Status:     lo.ToPtr(workingStatus.Phase),
			Err:        lo.ToPtr(workingStatus.Err),
			RemoteAddr: lo.ToPtr(workingStatus.RemoteAddr),
		},
	}, nil
}
