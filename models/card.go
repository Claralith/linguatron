package models

type Card struct {
	ID             uint `gorm:"primaryKey"`
	DeckID         uint
	Correct        uint   `gorm:"default:0"`
	Incorrect      uint   `gorm:"default:0"`
	CardCreated    string `gorm:"default:''"`
	LastReviewDate string `gorm:"default:''"`
	Stage          string `gorm:"default:'learning'"`
	Lapses         uint   `gorm:"default:0"`
	Ease           uint   `gorm:"default:1"`
	ReviewDueDate  string `gorm:"default:''"`
	Question       string
	Answer         string
}
