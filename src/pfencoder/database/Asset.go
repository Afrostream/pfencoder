package database

type Asset struct {
	ID                int   `gorm:"primary_key;column:assetId"`
	ContentId         int   `gorm:"column:contentId"`
	PresetId          int   `gorm:"column:presetId"`
	AssetIdDependance string `gorm:"column:assetIdDependance"`
	Filename          string `gorm:"column:filename"`
	DoAnalyse         string `gorm:"column:doAnalyse"`	/* yes, no */
	State             string `gorm:"column:state"`		/* scheduled, processing, ready failed */
	CreatedAt         string `gorm:"column:createdAt"`
	UpdatedAt         string `gorm:"column:updatedAt"`
}
