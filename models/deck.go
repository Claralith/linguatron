package models

type Deck struct {
	ID    uint `gorm:"primaryKey"`
	Name  string
	Cards []Card `gorm:"foreignKey:DeckID"`
}
