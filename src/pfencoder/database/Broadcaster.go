package database

type Broadcaster struct {
	ID        int    `gorm:"primary_key;column:id"`
	Name      string `gorm:"column:name"`
	ProfileId int    `gorm:"column:profileId"`
}

func (Broadcaster) TableName() string {
	return "broadcasters"
}
