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
	AssetId int     `gorm:"column:assetId"`
	Frame   int     `gorm:"column:frame"`
	Fps     int     `gorm:"column:fps"`
	Q       int     `gorm:"column:q"`
	Size    int     `gorm:"column:size"`
	Elapsed string  `gorm:"column:elapsed"`
	Bitrate float32 `gorm:"column:bitrate"`
}
