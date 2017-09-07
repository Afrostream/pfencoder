package database

import (
	"time"
)

type Profile struct {
	ID              int       `gorm:"primary_key;column:profileId"`
	Name            string    `gorm:"column:name"`
	Broadcaster     string    `gorm:"column:broadcaster"`
	AcceptSubtitles string    `gorm:"column:acceptSubtitles"` /* yes, no */
	CreatedAt       time.Time `gorm:"column:createdAt" sql:"DEFAULT:CURRENT_TIMESTAMP"`
	UpdatedAt       time.Time `gorm:"column:updatedAt" sql:"DEFAULT:CURRENT_TIMESTAMP"`
}

func (Profile) TableName() string {
	return "profiles"
}
