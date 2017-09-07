package database

import (
	"time"
)

type Preset struct {
	ID                 int       `gorm:"primary_key;column:presetId"`
	ProfileId          int       `gorm:"column:profileId"`
	PresetIdDependance string    `gorm:"column:presetIdDependance"`
	Name               string    `gorm:"column:name"`
	Type               string    `gorm:"column:type"`
	DoAnalyze          string    `gorm:"column:doAnalyze"`
	CmdLine            string    `gorm:"column:cmdLine"`
	CreatedAt          time.Time `gorm:"column:createdAt" sql:"DEFAULT:CURRENT_TIMESTAMP"`
	UpdatedAt          time.Time `gorm:"column:updatedAt" sql:"DEFAULT:CURRENT_TIMESTAMP"`
}

func (Preset) TableName() string {
	return "presets"
}
