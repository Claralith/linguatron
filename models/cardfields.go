package models

type CardFields struct {
	ID          uint `gorm:"primaryKey"`
	Content     string
	ShowOnFront bool
	ShowOnBack  bool
	CardID      uint
	Card        Card `gorm:"constraint:OnDelete:CASCADE;foreignKey:CardID"`
}
