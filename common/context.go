package common

import (
	"context"

	"fysj.net/v2/models"
)

func GetUserInfo(c context.Context) models.UserInfo {
	val := c.Value(UserInfoKey)
	if val == nil {
		return nil
	}

	u, ok := val.(*models.UserEntity)
	if !ok {
		return nil
	}

	return u
}
