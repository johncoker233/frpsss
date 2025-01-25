package proxy

import (
	"context"
	"fmt"

	"fysj.net/v2/common"
	"fysj.net/v2/dao"
	"fysj.net/v2/logger"
	"fysj.net/v2/pb"
)

// GetProxyStatsByClientID get proxy info by client id
func GetProxyStatsByClientID(c context.Context, req *pb.GetProxyStatsByClientIDRequest) (*pb.GetProxyStatsByClientIDResponse, error) {
	logger.Logger(c).Infof("get proxy by client id, req: [%+v]", req)
	var (
		clientID = req.GetClientId()
		userInfo = common.GetUserInfo(c)
	)

	if len(clientID) == 0 {
		return nil, fmt.Errorf("request invalid")
	}

	proxyStatsList, err := dao.GetProxyStatsByClientID(userInfo, clientID)
	if proxyStatsList == nil || err != nil {
		logger.Logger(context.Background()).WithError(err).Errorf("cannot get proxy, client id: [%s]", clientID)
		return nil, err
	}
	return &pb.GetProxyStatsByClientIDResponse{
		Status:     &pb.Status{Code: pb.RespCode_RESP_CODE_SUCCESS, Message: "ok"},
		ProxyInfos: convertProxyStatsList(proxyStatsList),
	}, nil
}
