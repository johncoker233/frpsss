package server

import (
	"fmt"

	"fysj.net/v2/dao"
	"fysj.net/v2/models"
)

type ValidateableServerRequest interface {
	GetServerSecret() string
	GetServerId() string
}

func ValidateServerRequest(req ValidateableServerRequest) (*models.ServerEntity, error) {
	if req == nil {
		return nil, fmt.Errorf("invalid request")
	}

	if req.GetServerId() == "" || req.GetServerSecret() == "" {
		return nil, fmt.Errorf("invalid request")
	}

	var (
		cli *models.ServerEntity
		err error
	)

	if cli, err = dao.ValidateServerSecret(req.GetServerId(), req.GetServerSecret()); err != nil {
		return nil, err
	}

	return cli, nil
}
