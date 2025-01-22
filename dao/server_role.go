package dao

func CheckServerRole(userInfo models.UserInfo, serverID string) error {
    var serverRole models.ServerRole
    result := DB.Where("server_id = ?", serverID).First(&serverRole)
    if result.Error != nil {
        return result.Error
    }
    
    // 如果是管理员，直接放行
    if userInfo.IsAdmin() {
        return nil
    }
    
    // 检查用户角色是否在允许的角色列表中
    if !serverRole.HasRole(userInfo.GetRole()) {
        return fmt.Errorf("permission denied: required roles %s", serverRole.Roles)
    }
    
    return nil
}