package database

type Encoder struct {
	ID          int     `gorm:"primary_key;column:encoderId"`
	Hostname    string  `gorm:"column:hostname"`
	ActiveTasks int     `gorm:"column:activeTasks"`
	MaxTasks    int     `gorm:"column:maxTasks"`
	Load1       float32 `gorm:"column:load1"`
	UpdatedAt   string  `gorm:"column:updatedAt"`
}
