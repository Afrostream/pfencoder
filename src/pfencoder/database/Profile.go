package database

type Profile struct {
	ID              int    `gorm:"primary_key;column:profileId"`
	Name            string `gorm:"column:name"`
	Broadcaster     string `gorm:"column:broadcaster"`
	AcceptSubtitles string `gorm:"column:acceptSubtitles"` /* yes, no */
	CreatedAt       string `gorm:"column:createdAt"`
	UpdatedAt       string `gorm:"column:updatedAt"`
}

func (Profile) TableName() string {
	return "profiles"
}
