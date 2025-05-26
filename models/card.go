package models

import "time"

type Card struct {
	ID             uint `gorm:"primaryKey"`
	DeckID         uint
	Correct        uint      `gorm:"default:0"`
	Incorrect      uint      `gorm:"default:0"`
	CardCreated    time.Time `gorm:"autoCreateTime"`
	LastReviewDate time.Time
	Stage          string `gorm:"default:'learning'"`
	Lapses         uint   `gorm:"default:0"`
	Ease           uint   `gorm:"default:1"`
	ReviewDueDate  time.Time
	Question       string
	Answer         string
	Extra          string
	Audio          string
	Image          string
}
