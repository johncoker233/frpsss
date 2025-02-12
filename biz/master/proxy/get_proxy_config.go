package proxy

import (
	"context"
	"fmt"

	"fysj.net/v2/common"
	"fysj.net/v2/dao"
	"fysj.net/v2/logger"
	"fysj.net/v2/models"
	"fysj.net/v2/pb"
	"fysj.net/v2/rpc"
	"github.com/samber/lo"
)

func GetProxyConfig(c context.Context, req *pb.GetProxyConfigRequest) (*pb.GetProxyConfigResponse, error) {
	var (
		userInfo  = common.GetUserInfo(c)
		clientID  = req.GetClientId()
		serverID  = req.GetServerId()
		proxyName = req.GetName()
	)

	proxyConfig, err := dao.GetProxyConfigByFilter(userInfo, &models.ProxyConfigEntity{
		ClientID: clientID,
		ServerID: serverID,
		Name:     proxyName,
	})
	if err != nil {
		logger.Logger(c).WithError(err).Errorf("cannot get proxy config, client: [%s], server: [%s], proxy name: [%s]", clientID, serverID, proxyName)
		return nil, err
	}

	resp := &pb.GetProxyConfigResponse{}
	if err := rpc.CallClientWrapper(c, proxyConfig.OriginClientID, pb.Event_EVENT_GET_PROXY_INFO, &pb.GetProxyConfigRequest{
		ClientId: lo.ToPtr(proxyConfig.ClientID),
		ServerId: lo.ToPtr(proxyConfig.ServerID),
		Name:     lo.ToPtr(fmt.Sprintf("%s.%s", userInfo.GetUserName(), proxyName)),
	}, resp); err != nil {
		resp.WorkingStatus = &pb.ProxyWorkingStatus{
			Status: lo.ToPtr("error"),
		}
		logger.Logger(c).WithError(err).Errorf("cannot get proxy config, client: [%s], server: [%s], proxy name: [%s]", proxyConfig.OriginClientID, proxyConfig.ServerID, proxyConfig.Name)
	}

	if len(resp.GetWorkingStatus().GetStatus()) == 0 {
		resp.WorkingStatus = &pb.ProxyWorkingStatus{
			Status: lo.ToPtr("unknown"),
		}
	}

	return &pb.GetProxyConfigResponse{
		Status: &pb.Status{Code: pb.RespCode_RESP_CODE_SUCCESS, Message: "success"},
		ProxyConfig: &pb.ProxyConfig{
			Id:             lo.ToPtr(uint32(proxyConfig.ID)),
			Name:           lo.ToPtr(proxyConfig.Name),
			Type:           lo.ToPtr(proxyConfig.Type),
			ClientId:       lo.ToPtr(proxyConfig.ClientID),
			ServerId:       lo.ToPtr(proxyConfig.ServerID),
			Config:         lo.ToPtr(string(proxyConfig.Content)),
			OriginClientId: lo.ToPtr(proxyConfig.OriginClientID),
		},
		WorkingStatus: resp.GetWorkingStatus(),
	}, nil
}
