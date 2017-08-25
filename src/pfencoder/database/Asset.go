package database

import (
	"time"
)

type Asset struct {
	ID                int       `gorm:"primary_key;column:assetId" json:"assetId"`
	ContentId         int       `gorm:"column:contentId" json:"contentId"`
	PresetId          int       `gorm:"column:presetId" json:"presetId"`
	AssetIdDependance *string   `gorm:"column:assetIdDependance" json:"assetIdDependance"` /* can be Null */
	Filename          string    `gorm:"column:filename" json:"filename"`
	DoAnalyze         string    `gorm:"column:doAnalyze" json:"doAnalyze"`                   /* yes, no */
	State             string    `gorm:"column:state" sql:"DEFAULT:'scheduled'" json:"state"` /* scheduled, processing, ready, failed */
	CreatedAt         time.Time `gorm:"column:createdAt" sql:"DEFAULT:CURRENT_TIMESTAMP" json:"createdAt"`
	UpdatedAt         time.Time `gorm:"column:updatedAt" sql:"DEFAULT:CURRENT_TIMESTAMP" json:"updatedAt"`
}

func (Asset) TableName() string {
	return "assets"
}
