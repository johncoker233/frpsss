package dao

import (
	"context"
	"fmt"
	"time"

	"fysj.net/v2/logger"
	"fysj.net/v2/models"
	"fysj.net/v2/pb"
	"fysj.net/v2/utils"
	"github.com/samber/lo"
	"github.com/sourcegraph/conc/pool"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func GetProxyStatsByClientID(userInfo models.UserInfo, clientID string) ([]*models.ProxyStatsEntity, error) {
	if clientID == "" {
		return nil, fmt.Errorf("invalid client id")
	}
	db := models.GetDBManager().GetDefaultDB()
	list := []*models.ProxyStats{}
	err := db.
		Where(&models.ProxyStats{ProxyStatsEntity: &models.ProxyStatsEntity{
			UserID:   userInfo.GetUserID(),
			TenantID: userInfo.GetTenantID(),
			ClientID: clientID,
		}}).
		Or(&models.ProxyStats{ProxyStatsEntity: &models.ProxyStatsEntity{
			UserID:   0,
			TenantID: userInfo.GetTenantID(),
			ClientID: clientID,
		}}).
		Or(&models.ProxyStats{ProxyStatsEntity: &models.ProxyStatsEntity{
			UserID:         userInfo.GetUserID(),
			TenantID:       userInfo.GetTenantID(),
			OriginClientID: clientID,
		}}).
		Or(&models.ProxyStats{ProxyStatsEntity: &models.ProxyStatsEntity{
			UserID:         0,
			TenantID:       userInfo.GetTenantID(),
			OriginClientID: clientID,
		}}).
		Find(&list).Error
	if err != nil {
		return nil, err
	}
	return lo.Map(list, func(item *models.ProxyStats, _ int) *models.ProxyStatsEntity {
		return item.ProxyStatsEntity
	}), nil
}

func GetProxyStatsByServerID(userInfo models.UserInfo, serverID string) ([]*models.ProxyStatsEntity, error) {
	if serverID == "" {
		return nil, fmt.Errorf("invalid server id")
	}
	db := models.GetDBManager().GetDefaultDB()
	list := []*models.ProxyStats{}
	err := db.
		Where(&models.ProxyStats{ProxyStatsEntity: &models.ProxyStatsEntity{
			UserID:   userInfo.GetUserID(),
			TenantID: userInfo.GetTenantID(),
			ServerID: serverID,
		}}).Or(&models.ProxyStats{ProxyStatsEntity: &models.ProxyStatsEntity{
		UserID:   0,
		TenantID: userInfo.GetTenantID(),
		ServerID: serverID,
	}}).
		Find(&list).Error
	if err != nil {
		return nil, err
	}
	return lo.Map(list, func(item *models.ProxyStats, _ int) *models.ProxyStatsEntity {
		return item.ProxyStatsEntity
	}), nil
}

func AdminUpdateProxyStats(srv *models.ServerEntity, inputs []*pb.ProxyInfo) error {
    if srv.ServerID == "" {
        return fmt.Errorf("invalid server id")
    }

    // 构建输入数据map
    inputMap := lo.SliceToMap(inputs, func(p *pb.ProxyInfo) (string, *pb.ProxyInfo) {
		return *p.Name, p
    })

    // 构建代理实体map 
    proxyMap := make(map[string]*models.ProxyStatsEntity)
	for _, input := range inputs {
		proxyMap[input.GetName()] = &models.ProxyStatsEntity{
			Name:            input.GetName(),
			UserID:          srv.UserID,
			ServerID:        srv.ServerID,
			TodayTrafficIn:  input.GetTodayTrafficIn(),
			TodayTrafficOut: input.GetTodayTrafficOut(),
		}
	}

    db := models.GetDBManager().GetDefaultDB()
    return db.Transaction(func(tx *gorm.DB) error {
        // 并发查询数据
        queryResults := make([]interface{}, 3)
        p := pool.New().WithErrors()
        
        // 查询用户信息
        p.Go(func() error {
            user := models.User{}
            if err := tx.Where(&models.User{
                UserEntity: &models.UserEntity{
                    UserID: srv.UserID,
                },
            }).First(&user).Error; err != nil {
                return err
            }
            queryResults[0] = user
            return nil
        })

        // 查询客户端信息
        p.Go(func() error {
            clients := []*models.Client{}
            if err := tx.Where(&models.Client{
                ClientEntity: &models.ClientEntity{
                    UserID:   srv.UserID,
                    ServerID: srv.ServerID,
                },
            }).Find(&clients).Error; err != nil {
                return err
            }
            queryResults[1] = clients
            return nil
        })

        // 查询历史代理统计信息
        p.Go(func() error {
            oldProxy := []*models.ProxyStats{}
            if err := tx.Where(&models.ProxyStats{
                ProxyStatsEntity: &models.ProxyStatsEntity{
                    UserID:   srv.UserID,
                    ServerID: srv.ServerID,
                },
            }).Find(&oldProxy).Error; err != nil {
                return err
            }
            oldProxyMap := lo.SliceToMap(oldProxy, func(p *models.ProxyStats) (string, *models.ProxyStats) {
                return p.Name, p
            })
            queryResults[2] = oldProxyMap
            return nil
        })

        if err := p.Wait(); err != nil {
            return err
        }

        user := queryResults[0].(models.User)
        clients := queryResults[1].([]*models.Client)
        oldProxyMap := queryResults[2].(map[string]*models.ProxyStats)

        // 获取角色流量百分比
        rolePercentage := models.RolePercentage{}
        if err := tx.Where(&models.RolePercentage{
            Role: user.Role,
        }).First(&rolePercentage).Error; err != nil {
            return err
        }

        // 关联客户端与代理
        proxyEntityMap := map[string]*models.ProxyStatsEntity{}
        for _, client := range clients {
            cliCfg, err := client.GetConfigContent()
            if err != nil || cliCfg == nil {
                continue
            }
            for _, cfg := range cliCfg.Proxies {
                if proxy, ok := proxyMap[cfg.GetBaseConfig().Name]; ok {
                    proxy.ClientID = client.ClientID
                    proxy.OriginClientID = client.OriginClientID
                    proxyEntityMap[proxy.Name] = proxy
                }
            }
        }

        // 更新统计数据
        nowTime := time.Now()
        results := lo.Values(lo.MapValues(proxyEntityMap, func(p *models.ProxyStatsEntity, name string) *models.ProxyStats {
            item := &models.ProxyStats{
                ProxyStatsEntity: p,
            }
			if oldProxy, ok := oldProxyMap[name]; ok {
				item.ProxyID = oldProxy.ProxyID
				firstSync := inputMap[name].GetFirstSync()
				isSameDay := utils.IsSameDay(nowTime, oldProxy.UpdatedAt)
		
				// 基础历史流量
				item.HistoryTrafficIn = oldProxy.HistoryTrafficIn
				item.HistoryTrafficOut = oldProxy.HistoryTrafficOut
				
				// 设置今日流量为当前上报值
				newTodayIn := inputMap[name].GetTodayTrafficIn()
				newTodayOut := inputMap[name].GetTodayTrafficOut()
				
				if !isSameDay || firstSync {
					// 不同天或首次同步,累计昨日流量到历史
					item.HistoryTrafficIn += oldProxy.TodayTrafficIn
					item.HistoryTrafficOut += oldProxy.TodayTrafficOut
					
					// 记录历史
					historyStats := &models.HistoryProxyStats{
						ProxyID:        oldProxy.ProxyID,
						ServerID:       oldProxy.ServerID,
						ClientID:       oldProxy.ClientID,
						OriginClientID: oldProxy.OriginClientID,
						Name:          oldProxy.Name,
						Type:          oldProxy.Type,
						UserID:        oldProxy.UserID,
						TenantID:      oldProxy.TenantID,
						TrafficIn:     oldProxy.TodayTrafficIn,
						TrafficOut:    oldProxy.TodayTrafficOut,
					}
					if err := tx.Create(historyStats).Error; err != nil {
						return nil
					}
					
					// 重置今日流量
					item.TodayTrafficIn = newTodayIn
					item.TodayTrafficOut = newTodayOut
				} else {
					// 同一天,累加今日流量
					item.TodayTrafficIn = oldProxy.TodayTrafficIn + newTodayIn
					item.TodayTrafficOut = oldProxy.TodayTrafficOut + newTodayOut
				}
		
				// 计算需要扣除的流量
				trafficIn := newTodayIn * int64(rolePercentage.Percentage) / 100
				trafficOut := newTodayOut * int64(rolePercentage.Percentage) / 100
				user.Bandwidth = int(int64(user.Bandwidth) - (trafficIn + trafficOut))
				if user.Bandwidth < 0 {
					user.Bandwidth = 0
				}
				
				if err := tx.Save(&user).Error; err != nil {
					return nil
				}
			} else {
				// 新代理,直接使用上报流量
				item.TodayTrafficIn = inputMap[name].GetTodayTrafficIn()
				item.TodayTrafficOut = inputMap[name].GetTodayTrafficOut()
			}
			return item
		}))

		// 批量保存更新
		if len(results) > 0 {
			return tx.Save(results).Error
		}
		return nil
	})
}

func AdminGetTenantProxyStats(tenantID int) ([]*models.ProxyStatsEntity, error) {
	db := models.GetDBManager().GetDefaultDB()
	list := []*models.ProxyStats{}
	err := db.
		Where(&models.ProxyStats{ProxyStatsEntity: &models.ProxyStatsEntity{
			TenantID: tenantID,
		}}).
		Find(&list).Error
	if err != nil {
		return nil, err
	}
	return lo.Map(list, func(item *models.ProxyStats, _ int) *models.ProxyStatsEntity {
		return item.ProxyStatsEntity
	}), nil
}

func AdminGetAllProxyStats(tx *gorm.DB) ([]*models.ProxyStatsEntity, error) {
	db := tx
	list := []*models.ProxyStats{}
	err := db.Clauses(clause.Locking{Strength: "UPDATE"}).
		Find(&list).Error
	if err != nil {
		return nil, err
	}
	return lo.Map(list, func(item *models.ProxyStats, _ int) *models.ProxyStatsEntity {
		return item.ProxyStatsEntity
	}), nil
}

func AdminCreateProxyConfig(proxyCfg *models.ProxyConfig) error {
	db := models.GetDBManager().GetDefaultDB()
	return db.Create(proxyCfg).Error
}

// RebuildProxyConfigFromClient rebuild proxy from client
func RebuildProxyConfigFromClient(userInfo models.UserInfo, client *models.Client) error {
	db := models.GetDBManager().GetDefaultDB()

	pxyCfgs, err := utils.LoadProxiesFromContent(client.ConfigContent)
	if err != nil {
		return err
	}

	proxyConfigEntities := []*models.ProxyConfig{}

	for _, pxyCfg := range pxyCfgs {
		proxyCfg := &models.ProxyConfig{
			ProxyConfigEntity: &models.ProxyConfigEntity{},
		}
		if oldProxyCfg, err := GetProxyConfigByOriginClientIDAndName(userInfo, client.ClientID, pxyCfg.GetBaseConfig().Name); err == nil {
			logger.Logger(context.Background()).WithError(err).Warnf("proxy config already exist, will be override, clientID: [%s], name: [%s]",
				client.ClientID, pxyCfg.GetBaseConfig().Name)
			proxyCfg.Model = oldProxyCfg.Model
		}

		if err := proxyCfg.FillClientConfig(client.ClientEntity); err != nil {
			return err
		}

		if err := proxyCfg.FillTypedProxyConfig(pxyCfg); err != nil {
			return err
		}

		proxyConfigEntities = append(proxyConfigEntities, proxyCfg)
	}

	if err := DeleteProxyConfigsByClientIDOrOriginClientID(userInfo, client.ClientID); err != nil {
		return err
	}

	if len(proxyConfigEntities) == 0 {
		return nil
	}

	return db.Save(proxyConfigEntities).Error
}

func AdminGetProxyConfigByClientIDAndName(clientID string, name string) (*models.ProxyConfig, error) {
	db := models.GetDBManager().GetDefaultDB()
	proxyCfg := &models.ProxyConfig{}
	err := db.
		Where(&models.ProxyConfig{ProxyConfigEntity: &models.ProxyConfigEntity{
			ClientID: clientID,
			Name:     name,
		}}).
		First(proxyCfg).Error
	if err != nil {
		return nil, err
	}
	return proxyCfg, nil
}

func GetProxyConfigsByClientID(userInfo models.UserInfo, clientID string) ([]*models.ProxyConfigEntity, error) {
	if clientID == "" {
		return nil, fmt.Errorf("invalid client id")
	}
	db := models.GetDBManager().GetDefaultDB()
	list := []*models.ProxyConfig{}
	err := db.
		Where(&models.ProxyConfig{ProxyConfigEntity: &models.ProxyConfigEntity{
			UserID:   userInfo.GetUserID(),
			TenantID: userInfo.GetTenantID(),
			ClientID: clientID,
		}}).
		Find(&list).Error
	if err != nil {
		return nil, err
	}
	return lo.Map(list, func(item *models.ProxyConfig, _ int) *models.ProxyConfigEntity {
		return item.ProxyConfigEntity
	}), nil
}

func GetProxyConfigByFilter(userInfo models.UserInfo, proxyConfig *models.ProxyConfigEntity) (*models.ProxyConfig, error) {
	db := models.GetDBManager().GetDefaultDB()
	filter := &models.ProxyConfigEntity{}

	if len(proxyConfig.ClientID) != 0 {
		filter.ClientID = proxyConfig.ClientID
	}
	if len(proxyConfig.OriginClientID) != 0 {
		filter.OriginClientID = proxyConfig.OriginClientID
	}
	if len(proxyConfig.Name) != 0 {
		filter.Name = proxyConfig.Name
	}
	if len(proxyConfig.Type) != 0 {
		filter.Type = proxyConfig.Type
	}
	if len(proxyConfig.ServerID) != 0 {
		filter.ServerID = proxyConfig.ServerID
	}

	filter.UserID = userInfo.GetUserID()
	filter.TenantID = userInfo.GetTenantID()

	respProxyCfg := &models.ProxyConfig{}
	err := db.
		Where(&models.ProxyConfig{ProxyConfigEntity: filter}).
		First(respProxyCfg).Error
	if err != nil {
		return nil, err
	}
	return respProxyCfg, nil
}

func ListProxyConfigsWithFilters(userInfo models.UserInfo, page, pageSize int, filters *models.ProxyConfigEntity) ([]*models.ProxyConfig, error) {
	if page < 1 || pageSize < 1 {
		return nil, fmt.Errorf("invalid page or page size")
	}

	db := models.GetDBManager().GetDefaultDB()
	offset := (page - 1) * pageSize

	filters.UserID = userInfo.GetUserID()
	filters.TenantID = userInfo.GetTenantID()

	var proxyConfigs []*models.ProxyConfig
	err := db.Where(&models.ProxyConfig{
		ProxyConfigEntity: filters,
	}).Where(filters).Offset(offset).Limit(pageSize).Find(&proxyConfigs).Error
	if err != nil {
		return nil, err
	}

	return proxyConfigs, nil
}

func AdminListProxyConfigsWithFilters(filters *models.ProxyConfigEntity) ([]*models.ProxyConfig, error) {
	db := models.GetDBManager().GetDefaultDB()

	var proxyConfigs []*models.ProxyConfig
	err := db.Where(&models.ProxyConfig{
		ProxyConfigEntity: filters,
	}).Where(filters).Find(&proxyConfigs).Error
	if err != nil {
		return nil, err
	}

	return proxyConfigs, nil
}

func ListProxyConfigsWithFiltersAndKeyword(userInfo models.UserInfo, page, pageSize int, filters *models.ProxyConfigEntity, keyword string) ([]*models.ProxyConfig, error) {
	if page < 1 || pageSize < 1 || len(keyword) == 0 {
		return nil, fmt.Errorf("invalid page or page size or keyword")
	}

	db := models.GetDBManager().GetDefaultDB()
	offset := (page - 1) * pageSize

	filters.UserID = userInfo.GetUserID()
	filters.TenantID = userInfo.GetTenantID()

	var proxyConfigs []*models.ProxyConfig
	err := db.Where(&models.ProxyConfig{
		ProxyConfigEntity: filters,
	}).Where(filters).Where("name like ?", "%"+keyword+"%").Offset(offset).Limit(pageSize).Find(&proxyConfigs).Error
	if err != nil {
		return nil, err
	}

	return proxyConfigs, nil
}

func ListProxyConfigsWithKeyword(userInfo models.UserInfo, page, pageSize int, keyword string) ([]*models.ProxyConfig, error) {
	return ListProxyConfigsWithFiltersAndKeyword(userInfo, page, pageSize, &models.ProxyConfigEntity{}, keyword)
}

func ListProxyConfigs(userInfo models.UserInfo, page, pageSize int) ([]*models.ProxyConfig, error) {
	return ListProxyConfigsWithFilters(userInfo, page, pageSize, &models.ProxyConfigEntity{})
}

func CreateProxyConfig(userInfo models.UserInfo, proxyCfg *models.ProxyConfigEntity) error {
	db := models.GetDBManager().GetDefaultDB()
	proxyCfg.UserID = userInfo.GetUserID()
	proxyCfg.TenantID = userInfo.GetTenantID()
	return db.Create(&models.ProxyConfig{ProxyConfigEntity: proxyCfg}).Error
}

func UpdateProxyConfig(userInfo models.UserInfo, proxyCfg *models.ProxyConfig) error {
	if proxyCfg.ID == 0 {
		return fmt.Errorf("invalid proxy config id")
	}
	db := models.GetDBManager().GetDefaultDB()
	proxyCfg.UserID = userInfo.GetUserID()
	proxyCfg.TenantID = userInfo.GetTenantID()
	return db.Where(&models.ProxyConfig{
		ProxyConfigEntity: &models.ProxyConfigEntity{
			UserID:   userInfo.GetUserID(),
			TenantID: userInfo.GetTenantID(),
			ClientID: proxyCfg.ClientID,
		},
		Model: &gorm.Model{
			ID: proxyCfg.ID,
		},
	}).Save(proxyCfg).Error
}

func DeleteProxyConfig(userInfo models.UserInfo, clientID, name string) error {
	if clientID == "" || name == "" {
		return fmt.Errorf("invalid client id or name")
	}
	db := models.GetDBManager().GetDefaultDB()
	return db.Unscoped().
		Where(&models.ProxyConfig{ProxyConfigEntity: &models.ProxyConfigEntity{
			UserID:   userInfo.GetUserID(),
			TenantID: userInfo.GetTenantID(),
			ClientID: clientID,
			Name:     name,
		}}).
		Delete(&models.ProxyConfig{}).Error
}

func DeleteProxyConfigsByClientIDOrOriginClientID(userInfo models.UserInfo, clientID string) error {
	if clientID == "" {
		return fmt.Errorf("invalid client id")
	}
	db := models.GetDBManager().GetDefaultDB()
	return db.Unscoped().
		Where(&models.ProxyConfig{ProxyConfigEntity: &models.ProxyConfigEntity{
			UserID:   userInfo.GetUserID(),
			TenantID: userInfo.GetTenantID(),
			ClientID: clientID,
		}}).
		Or(&models.ProxyConfig{ProxyConfigEntity: &models.ProxyConfigEntity{
			UserID:         userInfo.GetUserID(),
			TenantID:       userInfo.GetTenantID(),
			OriginClientID: clientID,
		}}).
		Delete(&models.ProxyConfig{}).Error
}

func DeleteProxyConfigsByClientID(userInfo models.UserInfo, clientID string) error {
	if clientID == "" {
		return fmt.Errorf("invalid client id")
	}
	db := models.GetDBManager().GetDefaultDB()
	return db.Unscoped().
		Where(&models.ProxyConfig{ProxyConfigEntity: &models.ProxyConfigEntity{
			UserID:   userInfo.GetUserID(),
			TenantID: userInfo.GetTenantID(),
			ClientID: clientID,
		}}).
		Delete(&models.ProxyConfig{}).Error
}

func GetProxyConfigByOriginClientIDAndName(userInfo models.UserInfo, clientID string, name string) (*models.ProxyConfig, error) {
	if clientID == "" || name == "" {
		return nil, fmt.Errorf("invalid client id or name")
	}
	db := models.GetDBManager().GetDefaultDB()
	item := &models.ProxyConfig{}
	err := db.
		Where(&models.ProxyConfig{ProxyConfigEntity: &models.ProxyConfigEntity{
			UserID:         userInfo.GetUserID(),
			TenantID:       userInfo.GetTenantID(),
			OriginClientID: clientID,
			Name:           name,
		}}).
		First(&item).Error
	if err != nil {
		return nil, err
	}
	return item, nil
}

func CountProxyConfigs(userInfo models.UserInfo) (int64, error) {
	return CountProxyConfigsWithFilters(userInfo, &models.ProxyConfigEntity{})
}

func CountProxyConfigsWithFilters(userInfo models.UserInfo, filters *models.ProxyConfigEntity) (int64, error) {
	db := models.GetDBManager().GetDefaultDB()
	filters.UserID = userInfo.GetUserID()
	filters.TenantID = userInfo.GetTenantID()

	var count int64
	err := db.Model(&models.ProxyConfig{}).Where(&models.ProxyConfig{
		ProxyConfigEntity: filters,
	}).Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

func CountProxyConfigsWithFiltersAndKeyword(userInfo models.UserInfo, filters *models.ProxyConfigEntity, keyword string) (int64, error) {
	if len(keyword) == 0 {
		return CountProxyConfigsWithFilters(userInfo, filters)
	}

	db := models.GetDBManager().GetDefaultDB()
	filters.UserID = userInfo.GetUserID()
	filters.TenantID = userInfo.GetTenantID()

	var count int64
	err := db.Model(&models.ProxyConfig{}).Where(&models.ProxyConfig{
		ProxyConfigEntity: filters,
	}).Where("name like ?", "%"+keyword+"%").Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}
