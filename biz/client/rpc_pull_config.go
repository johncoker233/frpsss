package client

import (
	"context"
	"reflect"

	"fysj.net/v2/logger"
	"fysj.net/v2/pb"
	"fysj.net/v2/rpc"
	"fysj.net/v2/services/client"
	"fysj.net/v2/tunnel"
	"fysj.net/v2/utils"
	"github.com/samber/lo"
)

func PullConfig(clientID, clientSecret string) error {
	ctx := context.Background()
	ctrl := tunnel.GetClientController()

	logger.Logger(ctx).Infof("start to pull client config, clientID: [%s]", clientID)
	cli, err := rpc.MasterCli(ctx)
	if err != nil {
		logger.Logger(context.Background()).WithError(err).Error("cannot get master client")
		return err
	}
	resp, err := cli.PullClientConfig(ctx, &pb.PullClientConfigReq{
		Base: &pb.ClientBase{
			ClientId:     clientID,
			ClientSecret: clientSecret,
		},
	})
	if err != nil {
		logger.Logger(context.Background()).WithError(err).Error("cannot pull client config")
		return err
	}

	if resp.GetClient().GetStopped() {
		logger.Logger(ctx).Infof("client [%s] is stopped, stop client", clientID)
		ctrl.StopByClient(clientID)
		return nil
	}

	if len(resp.GetClient().GetOriginClientId()) == 0 {
		currentClientIDs := ctrl.List()
		if idsToRemove, _ := lo.Difference(resp.GetClient().GetClientIds(), currentClientIDs); len(idsToRemove) > 0 {
			logger.Logger(ctx).Infof("client [%s] has %d expired child clients, remove clientIDs: [%+v]", clientID, len(idsToRemove), idsToRemove)
			for _, id := range idsToRemove {
				ctrl.StopByClient(id)
				ctrl.DeleteByClient(id)
			}
		}
	}

	// this client is shadow client, has no config
	// pull child client config
	if len(resp.GetClient().GetClientIds()) > 0 {
		for _, id := range resp.GetClient().GetClientIds() {
			if id == clientID {
				logger.Logger(ctx).Infof("client [%s] is shadow client, skip", clientID)
				continue
			}
			if err := PullConfig(id, clientSecret); err != nil {
				logger.Logger(context.Background()).WithError(err).Errorf("cannot pull child client config, id: [%s]", id)
			}
		}
		return nil
	}

	if len(resp.GetClient().GetConfig()) == 0 {
		logger.Logger(ctx).Infof("client [%s] config is empty, wait for server init", clientID)
		return nil
	}

	c, p, v, err := utils.LoadClientConfig([]byte(resp.GetClient().GetConfig()), true)
	if err != nil {
		logger.Logger(context.Background()).WithError(err).Error("cannot load client config")
		return err
	}

	serverID := resp.GetClient().GetServerId()

	if t := ctrl.Get(clientID, serverID); t == nil {
		logger.Logger(ctx).Infof("client [%s] for server [%s] not exists, create it", clientID, serverID)
		ctrl.Add(clientID, serverID, client.NewClientHandler(c, p, v))
		ctrl.Run(clientID, serverID)
	} else {
		if !reflect.DeepEqual(t.GetCommonCfg(), c) {
			logger.Logger(ctx).Infof("client [%s] for server [%s] config changed, will recreate it", clientID, serverID)
			tcli := ctrl.Get(clientID, serverID)
			if tcli != nil {
				tcli.Stop()
				ctrl.Delete(clientID, serverID)
			}
			ctrl.Add(clientID, serverID, client.NewClientHandler(c, p, v))
			ctrl.Run(clientID, serverID)
		} else {
			logger.Logger(ctx).Infof("client [%s] for server [%s] already exists, update if need", clientID, serverID)
			tcli := ctrl.Get(clientID, serverID)
			if tcli == nil || !tcli.Running() {
				if tcli != nil {
					tcli.Stop()
					ctrl.Delete(clientID, serverID)
				}
				ctrl.Add(clientID, serverID, client.NewClientHandler(c, p, v))
				ctrl.Run(clientID, serverID)
			} else {
				tcli.Update(p, v)
			}
		}
	}

	logger.Logger(ctx).Infof("pull client config success, clientID: [%s], serverID: [%s]", clientID, serverID)
	return nil
}
