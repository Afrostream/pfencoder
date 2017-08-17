package database

type Preset struct {
	ID                 int    `gorm:"primary_key;column:presetId"`
	ProfileId          int    `gorm:"column:profileId"`
	PresetIdDependance string `gorm:"column:presetIdDependance"`
	Name               string `gorm:"column:name"`
	Type               string `gorm:"column:type"`
	DoAnalyze          bool   `gorm:"column:doAnalyze"`
	CmdLine            string `gorm:"column:cmdLine"`
	CreatedAt          string `gorm:"column:createdAt"`
	UpdatedAt          string `gorm:"column:updatedAt"`
}

func (Preset) TableName() string {
	return "presets"
}
