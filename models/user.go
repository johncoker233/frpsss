package models

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

type User struct {
	*UserEntity
}

type UserInfo interface {
	GetUserID() int
	GetUserIDStr() string
	GetUserName() string
	GetEmail() string
	GetHashedPassword() string
	GetStatus() int
	GetRole() string
	GetToken() string
	GetTenantID() int
	GetSafeUserInfo() UserEntity
	IsAdmin() bool
	Valid() bool
}

var _ UserInfo = (*UserEntity)(nil)

type UserEntity struct {
	UserID    int    `json:"user_id" gorm:"primaryKey"`
	UserName  string `json:"user_name" gorm:"uniqueIndex;not null"`
	Password  string `json:"password"`
	Email     string `json:"email" gorm:"uniqueIndex;not null"`
	Status    int    `json:"status"`
	Role      string `json:"role"`
	TenantID  int    `json:"tenant_id"`
	Token     string `json:"token"`
	Bandwidth int    `json:"Bandwidth"`
	balance   int    `json:"balance"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (u *UserEntity) GetUserID() int {
	return u.UserID
}

func (u *UserEntity) GetUserIDStr() string {
	return fmt.Sprint(u.UserID)
}

func (u *UserEntity) GetUserName() string {
	return u.UserName
}
func (u *UserEntity) GetBandwidth() int {
	return u.Bandwidth
}
func (u *UserEntity) GetBalance() int {
	return u.balance
}
func (u *UserEntity) GetEmail() string {
	return u.Email
}

func (u *UserEntity) GetHashedPassword() string {
	return u.Password
}

func (u *UserEntity) GetStatus() int {
	return u.Status
}

func (u *UserEntity) GetRole() string {
	return u.Role
}

func (u *UserEntity) GetTenantID() int {
	return u.TenantID
}

func (u *UserEntity) GetToken() string {
	return u.Token
}

func (u *UserEntity) GetSafeUserInfo() UserEntity {
	return UserEntity{
		UserID:   u.UserID,
		UserName: u.UserName,
		Email:    u.Email,
		Status:   u.Status,
		Role:     u.Role,
		Bandwidth:u.Bandwidth
		balance:  u.balance
	}
}

func (u *UserEntity) Valid() bool {
	if u == nil {
		return false
	}
	if u.Status == STATUS_BANED {
		return false
	}
	return true
}

func (u *UserEntity) IsAdmin() bool {
	return u.Role == ROLE_ADMIN
}

func (u *User) TableName() string {
	return "users"
}
