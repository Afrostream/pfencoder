package database

type Encoder struct {
	ID          uint    `gorm:"primary_key;column:encoderId"`
	Hostname    string  `gorm:"column:hostname"`
	activeTasks int     `gorm:"column:activeTasks"`
	maxTasks    int     `gorm:"column:maxTasks"`
	load1       float32 `gorm:"column:load1"`
	updatedAt   string  `gorm:"column:updatedAt"`
}
