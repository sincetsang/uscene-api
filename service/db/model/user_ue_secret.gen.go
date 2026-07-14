package model

import (
	"time"
)

const TableNameUserUeSecret = "user_ue_secret"

// UserUeSecret mapped from table <user_ue_secret>
type UserUeSecret struct {
	ID           int64     `gorm:"column:id;primaryKey;autoIncrement:true" json:"id"`
	UserID       int64     `gorm:"column:user_id;not null" json:"user_id"`
	Opennodecode string    `gorm:"column:opennodecode;not null" json:"opennodecode"`
	Uenodecode   string    `gorm:"column:uenodecode;not null" json:"uenodecode"`
	Status       string    `gorm:"column:status;not null;default:active" json:"status"`
	CreatedAt    time.Time `gorm:"column:created_at;not null" json:"created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at;not null" json:"updated_at"`
}

// TableName UserUeSecret's table name
func (*UserUeSecret) TableName() string {
	return TableNameUserUeSecret
}
