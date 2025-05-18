package models

type Media struct {
	ID      uint `gorm:"primaryKey"`
	URL     string
	Type    string //image, audio
	FieldID int
	Field   CardFields `gorm:"constraint:OnDelete:CASCADE;foreignKey:FieldID"`
}
