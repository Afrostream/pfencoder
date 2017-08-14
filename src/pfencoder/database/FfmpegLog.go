package database

type FfmpegLog struct {
	AssetId int    `gorm:"column:assetId"`
	Log     string `gorm:"column:log"`
}

func (FfmpegLog) TableName() string {
  return "ffmpegLogs"
}