package database

type ProfilesParameter struct {
	ID        int    `gorm:"primary_key;column:profileParameterId"`
	ProfileId int    `gorm:"column:profileId"`
	AssetId   int    `gorm:"column:assetId"` /* TODO : NCO : ? Why here ? */
	Parameter string `gorm:"column:parameter"`
	Value     string `gorm:"column:value"`
	CreatedAt string `gorm:"column:createdAt"`
	UpdatedAt string `gorm:"column:updatedAt"`
}
