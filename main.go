package main

import (
	"log"
	"webproject/database"
	"webproject/routes"
	"webproject/views/home"

	"webproject/models"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Database interface {
	createDeck(name string) error
	createCard(card models.Card) error
	getCardByID(id uint) (models.Card, error)
	getAllCardsByDeckID(id uint) ([]models.Card, error)
	getRandomCardsByDeckID(id uint) ([]models.Card, error)
	getLearningCardsByDeckID(id uint) ([]models.Card, error)
	getReviewCardsByDeckID(id uint) ([]models.Card, error)
	getDueReviewCardsByDeckID(id uint) ([]models.Card, error)
	getDeckByID(id uint) (models.Deck, error)
	selectAllDecks() ([]models.Deck, error)
	updateLearningCardByID(card models.Card) error
	updateReviewCardByID(card models.Card) error
	deleteCardByID(card models.Card) error
}

func main() {
	db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	gormDB := &database.GormDB{DB: db}
	db.AutoMigrate(&models.Deck{}, &models.Card{}, &models.CardFields{}, &models.Media{})

	r := gin.Default()

	r.Static("/static", "./static")

	r.GET("/", func(c *gin.Context) {

		c.Header("Content-Type", "text/html; charset=utf-8")

		home.HomePage().Render(c.Request.Context(), c.Writer)

	})

	routes.RegisterAll(r, gormDB)

	log.Println("Server starting on http://localhost:3030")
	if err := r.Run(":3030"); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}

}
