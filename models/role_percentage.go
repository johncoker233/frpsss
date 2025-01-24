package models

type RolePercentage struct {
    ID         int    `json:"id" gorm:"primaryKey"`
    Role       string `json:"role" gorm:"unique"`
    Percentage int    `json:"percentage"`
}

func (*RolePercentage) TableName() string {
    return "role_percentage"
}