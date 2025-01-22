package models

import "strings"

type ServerRole struct {
    ID       int    `json:"id" gorm:"primaryKey"`
    ServerID string `json:"server_id"`
    Roles    string `json:"roles"` // 逗号分割的角色列表，如 "admin,operator,viewer"
}

func (s *ServerRole) TableName() string {
    return "server_roles"
}

func (s *ServerRole) HasRole(role string) bool {
    if role == "" {
        return false
    }
    roleList := strings.Split(s.Roles, ",")
    for _, r := range roleList {
        if strings.TrimSpace(r) == role {
            return true
        }
    }
    return false
}