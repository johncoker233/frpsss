package client

import (
	"context"

	"fysj.net/v2/common"
	"fysj.net/v2/dao"
	"fysj.net/v2/logger"
	"fysj.net/v2/models"
	"github.com/samber/lo"
)

func SyncTunnel(ctx context.Context, userInfo models.UserInfo) error {
	clis, err := dao.GetAllClients(userInfo)
	if err != nil {
		return err
	}
	lo.ForEach(clis, func(cli *models.ClientEntity, _ int) {
		cfg, err := cli.GetConfigContent()
		if err != nil {
			logger.Logger(context.Background()).WithError(err).Errorf("cannot get client config content, id: [%s]", cli.ClientID)
			return
		}

		cfg.User = userInfo.GetUserName()
		cfg.Metadatas = map[string]string{
			common.FRPAuthTokenKey: userInfo.GetToken(),
		}
		if err := cli.SetConfigContent(*cfg); err != nil {
			logger.Logger(context.Background()).WithError(err).Errorf("cannot set client config content, id: [%s]", cli.ClientID)
			return
		}

		if err := dao.UpdateClient(userInfo, cli); err != nil {
			logger.Logger(context.Background()).WithError(err).Errorf("cannot update client, id: [%s]", cli.ClientID)
			return
		}
		logger.Logger(ctx).Infof("update client success, id: [%s]", cli.ClientID)
	})
	return nil
}
