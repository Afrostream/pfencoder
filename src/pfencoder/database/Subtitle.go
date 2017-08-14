package database

type Subtitle struct {
	ID        int    `gorm:"primary_key;column:id"`
	ContentId int    `gorm:"column:contentId"`
	Lang      string `gorm:"column:lang"`
	Url       string `gorm:"column:url"`
}

func (Subtitle) TableName() string {
  return "subtitles"
}