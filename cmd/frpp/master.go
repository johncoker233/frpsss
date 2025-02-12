package main

import (
	"context"
	"embed"
	"path/filepath"

	bizmaster "fysj.net/v2/biz/master"
	"fysj.net/v2/biz/master/auth"
	"fysj.net/v2/biz/master/proxy"
	"fysj.net/v2/biz/master/streamlog"
	bizserver "fysj.net/v2/biz/server"
	"fysj.net/v2/cache"
	"fysj.net/v2/common"
	"fysj.net/v2/conf"
	"fysj.net/v2/dao"
	"fysj.net/v2/logger"
	"fysj.net/v2/models"
	"fysj.net/v2/pb"
	"fysj.net/v2/rpc"
	"fysj.net/v2/services/api"
	"fysj.net/v2/services/master"
	"fysj.net/v2/services/rpcclient"
	"fysj.net/v2/utils"
	"fysj.net/v2/watcher"
	"github.com/fatedier/golib/crypto"
	"github.com/glebarez/sqlite"
	"github.com/sirupsen/logrus"
	"github.com/sourcegraph/conc"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var fs embed.FS

func runMaster() {
	c := context.Background()
	crypto.DefaultSalt = conf.MasterDefaultSalt()

	streamlog.Init()

	initDatabase(c)
	cache.InitCache()
	auth.InitAuth()
	creds := dao.InitCert(conf.GetCertTemplate())

	master.MustInitMasterService(creds)
	router := bizmaster.NewRouter(fs)
	api.MustInitApiService(conf.MasterAPIListenAddr(), router)

	logger.Logger(c).Infof("start to run master")
	m := master.GetMasterSerivce()
	a := api.GetAPIService()

	r, w := initDefaultInternalServer()
	defer w.Stop()
	defer r.Stop()

	tasks := watcher.NewClient()
	tasks.AddCronTask("@every 5s", proxy.CollectDailyStats)
	defer tasks.Stop()

	var wg conc.WaitGroup
	wg.Go(w.Run)
	wg.Go(r.Run)
	wg.Go(m.Run)
	wg.Go(a.Run)
	wg.Go(tasks.Run)
	wg.Wait()
}

func initDatabase(c context.Context) {
	logger.Logger(c).Infof("start to init database, type: %s", conf.Get().DB.Type)
	models.MustInitDBManager(nil, conf.Get().DB.Type)

	if conf.Get().IsDebug {
		models.GetDBManager().SetDebug(true)
	}

	switch conf.Get().DB.Type {
	case "sqlite3":
		if err := utils.EnsureDirectoryExists(conf.Get().DB.DSN); err != nil {
			logrus.WithError(err).Warnf("ensure directory failed, data location: [%s], keep data in current directory", conf.Get().DB.DSN)
			conf.Get().DB.DSN = filepath.Base(conf.Get().DB.DSN)
			logrus.Infof("new data location: [%s]", conf.Get().DB.DSN)
		}

		if sqlitedb, err := gorm.Open(sqlite.Open(conf.Get().DB.DSN), &gorm.Config{}); err != nil {
			logrus.Panic(err)
		} else {
			models.GetDBManager().SetDB("sqlite3", sqlitedb)
			logger.Logger(c).Infof("init database success, data location: [%s]", conf.Get().DB.DSN)
		}
	case "mysql":
		if mysqlDB, err := gorm.Open(mysql.Open(conf.Get().DB.DSN), &gorm.Config{}); err != nil {
			logrus.Panic(err)
		} else {
			models.GetDBManager().SetDB("mysql", mysqlDB)
			logger.Logger(c).Infof("init database success, data type: [%s]", "mysql")
		}
	case "postgres":
		if postgresDB, err := gorm.Open(postgres.Open(conf.Get().DB.DSN), &gorm.Config{}); err != nil {
			logrus.Panic(err)
		} else {
			models.GetDBManager().SetDB("postgres", postgresDB)
			logger.Logger(c).Infof("init database success, data type: [%s]", "postgres")
		}
	default:
		logrus.Panicf("currently unsupported database type: %s", conf.Get().DB.Type)
	}

	models.GetDBManager().Init()
}

func initDefaultInternalServer() (rpcclient.ClientRPCHandler, watcher.Client) {
	dao.InitDefaultServer(conf.Get().Master.APIHost)
	defaultServer, err := dao.GetDefaultServer()
	if err != nil {
		logrus.Fatal(err)
	}

	cred, err := utils.TLSClientCertNoValidate(rpc.GetClientCert(
		defaultServer.ServerID, defaultServer.ConnectSecret, pb.ClientType_CLIENT_TYPE_FRPS))
	if err != nil {
		logrus.Fatal(err)
	}
	conf.ClientCred = cred
	rpcclient.MustInitClientRPCSerivce(
		defaultServer.ServerID, defaultServer.ConnectSecret,
		pb.Event_EVENT_REGISTER_SERVER,
		bizserver.HandleServerMessage,
	)

	conf.Get().Client.ID = defaultServer.ServerID
	conf.Get().Client.Secret = defaultServer.ConnectSecret

	r := rpcclient.GetClientRPCSerivce()

	w := watcher.NewClient()
	w.AddDurationTask(common.PullConfigDuration, bizserver.PullConfig, defaultServer.ServerID, defaultServer.ConnectSecret)
	w.AddDurationTask(common.PushProxyInfoDuration, bizserver.PushProxyInfo, defaultServer.ServerID, defaultServer.ConnectSecret)

	go initServerOnce(defaultServer.ServerID, defaultServer.ConnectSecret)
	return r, w
}
