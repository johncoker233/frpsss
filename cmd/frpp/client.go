package main

import (
	"context"

	bizclient "fysj.net/v2/biz/client"
	"fysj.net/v2/common"
	"fysj.net/v2/conf"
	"fysj.net/v2/logger"
	"fysj.net/v2/pb"
	"fysj.net/v2/rpc"
	"fysj.net/v2/services/rpcclient"
	"fysj.net/v2/utils"
	"fysj.net/v2/watcher"
	"github.com/fatedier/golib/crypto"
	"github.com/sirupsen/logrus"
	"github.com/sourcegraph/conc"
)

func runClient() {
	var (
		c            = context.Background()
		clientID     = conf.Get().Client.ID
		clientSecret = conf.Get().Client.Secret
	)
	crypto.DefaultSalt = conf.Get().App.Secret
	logger.Logger(c).Infof("start to run client")
	if len(clientSecret) == 0 {
		logrus.Fatal("client secret cannot be empty")
	}

	if len(clientID) == 0 {
		logrus.Fatal("client id cannot be empty")
	}

	cred, err := utils.TLSClientCertNoValidate(rpc.GetClientCert(clientID, clientSecret, pb.ClientType_CLIENT_TYPE_FRPC))
	if err != nil {
		logrus.Fatal(err)
	}
	conf.ClientCred = cred

	rpcclient.MustInitClientRPCSerivce(
		clientID,
		clientSecret,
		pb.Event_EVENT_REGISTER_CLIENT,
		bizclient.HandleServerMessage,
	)
	r := rpcclient.GetClientRPCSerivce()
	defer r.Stop()

	w := watcher.NewClient()
	w.AddDurationTask(common.PullConfigDuration, bizclient.PullConfig, clientID, clientSecret)
	defer w.Stop()

	initClientOnce(clientID, clientSecret)

	var wg conc.WaitGroup
	wg.Go(r.Run)
	wg.Go(w.Run)
	wg.Wait()
}

func initClientOnce(clientID, clientSecret string) {
	err := bizclient.PullConfig(clientID, clientSecret)
	if err != nil {
		logger.Logger(context.Background()).WithError(err).Errorf("cannot pull client config, wait for retry")
	}
}
