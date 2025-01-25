package auth

import (
	"context"

	"fysj.net/v2/cache"
	"fysj.net/v2/dao"
	"fysj.net/v2/logger"
	"fysj.net/v2/models"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
)

func InitAuth() {
	c := context.Background()
	logrus.Info("start to init frp user auth token")

	u, err := dao.AdminGetAllUsers()
	if err != nil {
		logger.Logger(context.Background()).WithError(err).Fatalf("init frp user auth token failed")
	}

	lo.ForEach(u, func(user *models.UserEntity, _ int) {
		cache.Get().Set([]byte(user.GetUserName()), []byte(user.GetToken()), 0)
	})

	logger.Logger(c).Infof("init frp user auth token success, count: %d", len(u))
}
