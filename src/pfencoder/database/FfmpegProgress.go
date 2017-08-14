package database

type FfmpegProgressV0 struct {
	Frame   string
	Fps     string
	Q       string
	Size    string
	Elapsed string
	Bitrate string
}

type FfmpegProgress struct {
	AssetId int     `gorm:"primary_key;column:assetId"` /* forcing primary_key, none yet in DB in order to prevent multiple inserts */
	Frame   int     `gorm:"column:frame"`
	Fps     int     `gorm:"column:fps"`
	Q       float32 `gorm:"column:q"`
	Size    int     `gorm:"column:size"`
	Elapsed string  `gorm:"column:elapsed"`
	Bitrate float32 `gorm:"column:bitrate"`
}

func (FfmpegProgress) TableName() string {
	return "ffmpegProgress"
}
