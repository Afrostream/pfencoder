package database

type Content struct {
	ID         int    `gorm:"primary_key;column:contentId"`
	Uuid       string `gorm:"column:uuid"`
	Md5hash    string `gorm:"column:md5Hash"`
	Filename   string `gorm:"column:filename"`
	State      string `gorm:"column:state"` /* initialized, scheduled, processing, packaging, ready, failed */
	Size       int64  `gorm:"column:size"`
	Duration   string `gorm:"column:duration"`
	UspPackage string `gorm:"column:uspPackage"` /* enabled, disabled */
	Drm        string `gorm:"column:drm"`        /* enabled, disabled */
	CreatedAt  string `gorm:"column:createdAt"`
	UpdatedAt  string `gorm:"column:updatedAt"`
}

func (Content) TableName() string {
  return "contents"
}