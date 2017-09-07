package database

import (
	"time"
)

type Content struct {
	ID         int       `gorm:"primary_key;column:contentId"`
	Uuid       string    `gorm:"column:uuid"`
	Md5Hash    string    `gorm:"column:md5Hash"`
	Filename   string    `gorm:"column:filename"`
	State      string    `gorm:"column:state"` /* initialized, scheduled, processing, packaging, ready, failed */
	Size       int64     `gorm:"column:size"`
	Duration   string    `gorm:"column:duration"`
	UspPackage string    `gorm:"column:uspPackage"` /* enabled, disabled */
	Drm        string    `gorm:"column:drm"`        /* enabled, disabled */
	CreatedAt  time.Time `gorm:"column:createdAt" sql:"DEFAULT:CURRENT_TIMESTAMP"`
	UpdatedAt  time.Time `gorm:"column:updatedAt" sql:"DEFAULT:CURRENT_TIMESTAMP"`
	ProfileIds []int     `gorm:"-" json:"profilesIds"`
}

func (Content) TableName() string {
	return "contents"
}
