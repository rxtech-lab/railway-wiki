package models

import (
	"time"
)

// Example represents an example entity in the database
type Example struct {
	ID          string    `gorm:"primaryKey;type:uuid"`
	Name        string    `gorm:"not null;size:255"`
	Description *string   `gorm:"size:1000"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`
}

// TableName returns the table name for the Example model
func (Example) TableName() string {
	return "examples"
}
