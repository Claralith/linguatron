package main

import (
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
	"webproject/views/createcard"
	"webproject/views/createdeck"
	"webproject/views/deck"
	"webproject/views/decks"
	"webproject/views/home"
	"webproject/views/learning"

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

type GormDB struct {
	db *gorm.DB
}

func (g *GormDB) createDeck(name string) error {
	var deck models.Deck
	deck.Name = name
	return g.db.Create(&deck).Error
}

func (g *GormDB) createCard(card models.Card) error {
	return g.db.Create(&card).Error
}

func (g *GormDB) getCardByID(id uint) (models.Card, error) {
	var card models.Card
	err := g.db.First(&card, id).Error
	return card, err
}

func (g *GormDB) getAllCardsByDeckID(id uint) ([]models.Card, error) {
	var cards []models.Card
	err := g.db.Where("deck_id = ?", id).Find(&cards).Error
	return cards, err
}

func (g *GormDB) getRandomCardsByDeckID(deckID uint, cardID uint) ([]models.Card, error) {
	var count int64
	cardCountError := g.db.Model(&models.Card{}).Where("deck_id = ?", deckID).Count(&count).Error
	if cardCountError != nil {
		return nil, cardCountError
	}

	var limit int

	if count >= 3 && count < 5 {
		limit = 3
	} else if count >= 5 {
		limit = 5
	} else {
		limit = 0
	}

	var cards []models.Card
	err := g.db.Where("deck_id = ? AND id != ?", deckID, cardID).Order("RANDOM()").Limit(limit).Find(&cards).Error

	return cards, err
}

func (g *GormDB) getLearningCardsByDeckID(id uint) ([]models.Card, error) {
	var cards []models.Card
	err := g.db.Where("deck_id = ? AND stage = ?", id, "learning").Find(&cards).Error
	return cards, err
}
func (g *GormDB) getReviewCardsByDeckID(id uint) ([]models.Card, error) {
	var cards []models.Card
	err := g.db.Where("deck_id = ? AND stage = ?", id, "review").Find(&cards).Error
	return cards, err
}
func (g *GormDB) getDueReviewCardsByDeckID(id uint) ([]models.Card, error) {
	now := time.Now().UTC()

	var cards []models.Card
	err := g.db.Where("deck_id = ? AND stage = ? AND review_due_date <= ?", id, "review", now).Find(&cards).Error
	return cards, err
}

func (g *GormDB) getDeckByID(id uint) (models.Deck, error) {
	var deck models.Deck
	err := g.db.First(&deck, id).Error
	return deck, err
}

func (g *GormDB) selectAllDecks() ([]models.Deck, error) {
	var decks []models.Deck
	err := g.db.Find(&decks).Error
	return decks, err
}

func (g *GormDB) updateLearningCardByID(id uint, correct bool) error {
	card, _ := g.getCardByID(id)

	now := time.Now().UTC()

	minuteAfter := now.Add(1 * time.Minute)

	dayAfter := now.Add(24 * time.Hour)

	card.LastReviewDate = now
	if correct {
		card.Correct++
		if card.Ease > 1 {
			card.Ease = uint(getNextEaseLevel(int(card.Ease), 1))
			card.Stage = "review"
			card.ReviewDueDate = dayAfter

		} else {
			card.Ease = uint(getNextEaseLevel(int(card.Ease), 2))
			card.ReviewDueDate = minuteAfter
		}
	} else {
		card.Incorrect++
		card.Ease = 1
		card.ReviewDueDate = minuteAfter
	}
	return g.db.Save(&card).Error
}

func (g *GormDB) updateReviewCardByID(id uint, correct bool) error {
	card, _ := g.getCardByID(id)

	now := time.Now().UTC()

	minuteAfter := now.Add(time.Minute * time.Duration(1))

	card.LastReviewDate = now
	if correct {
		card.Correct++
		card.ReviewDueDate = createNextReviewDueDate(int(card.Ease))
		card.Ease = uint(getNextEaseLevel(int(card.Ease), 2))
	} else {
		card.Incorrect++
		card.ReviewDueDate = minuteAfter
		if card.Ease != 1 {
			card.Lapses++
			card.Ease = 1
		}
	}

	return g.db.Save(&card).Error
}

func (g *GormDB) updateCardByID(id uint, question string, answer string) error {
	card, _ := g.getCardByID(id)
	card.Question = question
	card.Answer = answer

	return g.db.Save(&card).Error
}

func (g *GormDB) deleteCardByID(id uint) error {
	card, err := g.getCardByID(id)
	if err != nil {
		return err
	}

	return g.db.Delete(card).Error
}

func IsAnswerCorrectInLowerCase(userAnswer string, databaseAnswer string) bool {
	return strings.EqualFold(strings.TrimSpace(userAnswer), (strings.TrimSpace(databaseAnswer)))
}

func getMostDueCard(cards []models.Card) (models.Card, error) {
	if len(cards) == 0 {
		return models.Card{}, fmt.Errorf("no cards")
	}

	mostDueCard := cards[0]

	for i := 1; i < len(cards); i++ {
		if cards[i].ReviewDueDate.Before(mostDueCard.ReviewDueDate) {
			mostDueCard = cards[i]
		}
	}

	return mostDueCard, nil
}
func isCardsNotEmpty(cards []models.Card) bool {
	if len(cards) > 0 {
		return true
	} else {
		return false
	}
}

func getNextEaseLevel(currentEase int, growthfactor float64) int {
	nextEase := int(math.Ceil(float64(currentEase) * growthfactor))

	return nextEase
}

func createNextReviewDueDate(ease int) time.Time {

	t := time.Now().UTC()
	hours := ease * 24
	duration := time.Duration(hours) * time.Hour

	return t.Add(duration)

}

func main() {
	db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	gormDB := &GormDB{db: db}
	db.AutoMigrate(&models.Deck{}, &models.Card{})

	r := gin.Default()

	r.GET("/", func(c *gin.Context) {

		c.Header("Content-Type", "text/html; charset=utf-8")

		home.HomePage("World").Render(c.Request.Context(), c.Writer)

	})

	r.GET("/update", func(c *gin.Context) {
		home.UpdatedContent().Render(c.Request.Context(), c.Writer)
	})

	r.GET("/createdeck", func(c *gin.Context) {
		createdeck.Load().Render(c.Request.Context(), c.Writer)
	})

	r.POST("/createdeck", func(c *gin.Context) {
		deckname := c.PostForm("deckname")
		(*gormDB).createDeck(deckname)
	})

	r.GET("/deck/:deckID/createcard", func(c *gin.Context) {

		deckIdStr := c.Param("deckID")
		deckId, err := strconv.ParseUint(deckIdStr, 10, 32)
		if err != nil {
			c.String(http.StatusBadRequest, "Invalid deck ID")
			log.Print(err)
			return
		}

		deck, err := (*gormDB).getDeckByID(uint(deckId))
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.String(http.StatusNotFound, "deck not found")
			} else {
				c.String(http.StatusInternalServerError, "error fetching deck")
			}
		}

		cards, err := gormDB.getAllCardsByDeckID(uint(deckId))
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.String(http.StatusNotFound, "cards not found")
			} else {
				c.String(http.StatusInternalServerError, "error fetching cards")
			}
		}

		c.Header("Content-Type", "text/html; charset=utf-8")
		createcard.Load(cards, deck).Render(c.Request.Context(), c.Writer)
	})

	r.POST("/deck/:deckID/createcard", func(c *gin.Context) {

		deckIdStr := c.Param("deckID")
		deckId, err := strconv.ParseUint(deckIdStr, 10, 32)
		if err != nil {
			c.String(http.StatusBadRequest, "Invalid deck ID")
			log.Print(err)
			return
		}

		var card models.Card
		card.DeckID = uint(deckId)
		card.Question = c.PostForm("question")
		card.Answer = c.PostForm("answer")
		card.CardCreated = time.Now().UTC()
		card.ReviewDueDate = time.Now().UTC()

		(*gormDB).createCard(card)

		cards, err := gormDB.getAllCardsByDeckID(uint(deckId))
		if err != nil {
			c.String(http.StatusBadRequest, "Invalid deck ID")
			log.Print(err)
			return
		}

		createcard.RenderTable(cards).Render(c.Request.Context(), c.Writer)

	})

	r.DELETE("/card/:cardID/delete", func(c *gin.Context) {

		cardIdStr := c.Param("cardID")
		cardId, err := strconv.ParseUint(cardIdStr, 10, 32)
		if err != nil {
			c.String(http.StatusBadRequest, "Invalid card ID")
			log.Print(err)
			return
		}

		card, err := gormDB.getCardByID(uint(cardId))
		if err != nil {
			c.String(http.StatusBadRequest, "Invalid card ID")
			log.Print(err)
			return
		}

		gormDB.deleteCardByID(card.ID)

		cards, err := gormDB.getAllCardsByDeckID(uint(card.DeckID))
		if err != nil {
			c.String(http.StatusBadRequest, "Invalid deck ID")
			log.Print(err)
			return
		}

		createcard.RenderTable(cards).Render(c.Request.Context(), c.Writer)

	})

	r.PUT("/card/:cardID/edit", func(c *gin.Context) {
		cardIdStr := c.Param("cardID")
		cardId, err := strconv.ParseUint(cardIdStr, 10, 32)
		if err != nil {
			c.String(http.StatusBadRequest, "Invalid card ID")
			log.Print(err)
			return
		}

		gormDB.updateCardByID(uint(cardId), c.PostForm("question"), c.PostForm("answer"))

		card, err := gormDB.getCardByID(uint(cardId))
		if err != nil {
			c.String(http.StatusBadRequest, "card not found")
			log.Print(err)
			return
		}

		createcard.UpdateRow(card).Render(c.Request.Context(), c.Writer)
	})

	r.GET("/deck/:deckID", func(c *gin.Context) {
		deckIdStr := c.Param("deckID")
		deckId, err := strconv.ParseUint(deckIdStr, 10, 32)
		if err != nil {
			c.String(http.StatusBadRequest, "Invalid deck ID")
			log.Print(err)
			return
		}

		deckData, err := gormDB.getDeckByID(uint(deckId))
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.String(http.StatusNotFound, "Deck not found")
			} else {
				c.String(http.StatusInternalServerError, "Error fetching deck")
			}
			return
		}

		c.Header("Content-Type", "text/html; charset=utf-8")
		deck.DeckView(deckData).Render(c.Request.Context(), c.Writer)
	})

	r.GET("/decks", func(c *gin.Context) {

		selectedDecks, err := (*gormDB).selectAllDecks()
		log.Print("after selectAllDecks")
		if err != nil {
			log.Print(err)
		}

		log.Print("err not nil")
		decks.LoadDecks(selectedDecks).Render(c.Request.Context(), c.Writer)
	})

	r.GET("/learning", func(c *gin.Context) {
		learning.InitialContent("test").Render(c.Request.Context(), c.Writer)
	})

	log.Println("Server starting on http://localhost:3030")
	if err := r.Run(":3030"); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}

}
