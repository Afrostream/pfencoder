package database

type Encoder struct {
	ID          int     `gorm:"primary_key;column:encoderId"`
	Hostname    string  `gorm:"column:hostname"`
	ActiveTasks int     `gorm:"column:activeTasks;default:0"`
	MaxTasks    int     `gorm:"column:maxTasks;default:1"`
	Load1       float32 `gorm:"column:load1;default:1"`
	UpdatedAt   string  `gorm:"column:updatedAt;default:CURRENT_TIMESTAMP"`
}

func (Encoder) TableName() string {
	return "encoders"
}
