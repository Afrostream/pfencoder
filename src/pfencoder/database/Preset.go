package database

type Preset struct {
	ID                 uint   `gorm:"primary_key;column:presetId"`
	ProfileId          int    `gorm:"column:profileId"`
	PresetIdDependance string `gorm:"column:presetIdDependance"`
	Type               string `gorm:"column:type"`
	DoAnaylse          bool   `gorm:"column:doAnaylse"`
	CmdLine            string `gorm:"column:cmdLine"`
	CreatedAt          string `gorm:"column:createdAt"`
	UpdatedAt          string `gorm:"column:updatedAt"`
}
