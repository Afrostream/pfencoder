package database

import (
	"time"
)

type ProfilesParameter struct {
	ID        int       `gorm:"primary_key;column:profileParameterId" json:"profileParameterId"`
	ProfileId int       `gorm:"column:profileId" json:"profileId"`
	AssetId   int       `gorm:"column:assetId" json:"assetId"` /* TODO : NCO : ? Why here ? */
	Parameter string    `gorm:"column:parameter" json:"parameter"`
	Value     string    `gorm:"column:value" json:"value"`
	CreatedAt time.Time `gorm:"column:createdAt" sql:"DEFAULT:CURRENT_TIMESTAMP" json:"createdAt"`
	UpdatedAt time.Time `gorm:"column:updatedAt" sql:"DEFAULT:CURRENT_TIMESTAMP" json:"updatedAt"`
}

func (ProfilesParameter) TableName() string {
	return "profilesParameters"
}
