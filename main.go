package main

import (
	"errors"
	"fmt"
	"log"
	"math"
	"math/rand/v2"
	"net/http"
	"strconv"
	"strings"
	"time"
	"webproject/views/batchadd"
	"webproject/views/createcard"
	"webproject/views/createdeck"
	"webproject/views/deck"
	"webproject/views/decks"
	"webproject/views/home"
	"webproject/views/learning"
	"webproject/views/review"

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

func (g *GormDB) getShuffledChoicesForCard(deckID uint, mostDueCard models.Card) ([]models.Card, error) {
	var count int64
	cardCountError := g.db.Model(&models.Card{}).Where("deck_id = ?", deckID).Count(&count).Error
	if cardCountError != nil {
		return nil, cardCountError
	}

	var limit int

	if count > 1 && count < 5 {
		limit = int(count - 1) //as many false answers as possible
	} else if count >= 5 {
		limit = 5
	} else {
		limit = 0
	}

	var falseAnswers []models.Card
	err := g.db.Where("deck_id = ? AND id != ?", deckID, mostDueCard.ID).
		Order("RANDOM()").
		Limit(limit).
		Find(&falseAnswers).Error
	if err != nil {
		return nil, err
	}

	falseandCorrect := append(falseAnswers, mostDueCard)
	rand.Shuffle(len(falseandCorrect), func(i, j int) {
		falseandCorrect[i], falseandCorrect[j] = falseandCorrect[j], falseandCorrect[i]
	})

	return falseandCorrect, err
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

func (g *GormDB) deleteDeckByID(id uint) error {
	card, err := g.getDeckByID(id)
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
	db.AutoMigrate(&models.Deck{}, &models.Card{}, &models.CardFields{}, &models.Media{})

	r := gin.Default()

	r.Static("/static", "./static")

	r.GET("/", func(c *gin.Context) {

		c.Header("Content-Type", "text/html; charset=utf-8")

		home.HomePage().Render(c.Request.Context(), c.Writer)

	})

	r.GET("/createdeck", func(c *gin.Context) {
		createdeck.Load().Render(c.Request.Context(), c.Writer)
	})

	r.POST("/createdeck", func(c *gin.Context) {
		deckname := c.PostForm("deckname")

		if strings.TrimSpace(deckname) == "" {
			createdeck.Error("Deck name cannot be empty").Render(c.Request.Context(), c.Writer)
			return
		}

		var deck models.Deck

		deck.Name = deckname
		err := gormDB.db.Create(&deck).Error
		if err != nil {
			createdeck.Error("Failed to create deck. Please try again. Error: "+err.Error()).Render(c.Request.Context(), c.Writer)
			return
		}

		createdeck.Created(deck).Render(c.Request.Context(), c.Writer)
	})

	r.DELETE("/deck/:deckID/delete", func(c *gin.Context) {
		deckIdStr := c.Param("deckID")
		deckId, err := strconv.ParseUint(deckIdStr, 10, 32)
		if err != nil {
			c.String(http.StatusBadRequest, "Invalid deck ID")
			log.Print(err)
			return
		}

		gormDB.deleteDeckByID(uint(deckId))

		selectedDecks, err := (*gormDB).selectAllDecks()
		if err != nil {
			log.Print(err)
		}

		decks.Decks(selectedDecks).Render(c.Request.Context(), c.Writer)

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

	r.GET("/deck/:deckID/batchadd", func(c *gin.Context) {
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

		batchadd.Page(deck, 0).Render(c.Request.Context(), c.Writer)

	})

	r.POST("/deck/:deckID/batchadd", func(c *gin.Context) {
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

		userInput := c.PostForm("batchinput")
		lines := strings.Split(userInput, "\n")

		var NumberOfCards int

		for _, line := range lines {
			parts := strings.Split(line, "|;")
			if len(parts) != 2 {
				continue
			}
			card := models.Card{
				DeckID:        uint(deckId),
				Question:      strings.TrimSpace(parts[0]),
				Answer:        strings.TrimSpace(parts[1]),
				CardCreated:   time.Now().UTC(),
				ReviewDueDate: time.Now().UTC(),
			}
			gormDB.createCard(card)
			NumberOfCards++
		}

		batchadd.Page(deck, NumberOfCards).Render(c.Request.Context(), c.Writer)
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

	r.GET("/deck/:deckID/learning", func(c *gin.Context) {
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

		learningCards, err := gormDB.getLearningCardsByDeckID(deck.ID)
		if err != nil || len(learningCards) == 0 {
			learning.NoneLeft(deck).Render(c.Request.Context(), c.Writer)
			return
		}

		mostDueCard, err := getMostDueCard(learningCards)
		if err != nil {
			c.String(http.StatusBadRequest, "not enough learning cards?")
			log.Print(err)
		}

		randomCards, err := gormDB.getShuffledChoicesForCard(deck.ID, mostDueCard)
		if err != nil {
			c.String(http.StatusBadRequest, "failed to get random cards")
			log.Print(err)
		}

		learning.InitialContent(mostDueCard, randomCards, deck).Render(c.Request.Context(), c.Writer)
	})

	r.POST("/card/:cardID/learning", func(c *gin.Context) {
		cardIdStr := c.Param("cardID")
		cardId, err := strconv.ParseUint(cardIdStr, 10, 32)
		if err != nil {
			c.String(http.StatusBadRequest, "Invalid card ID")
			log.Print(err)
			return
		}

		card, err := gormDB.getCardByID(uint(cardId))
		if err != nil {
			c.String(http.StatusBadRequest, "card not found")
			log.Print(err)
			return
		}

		correct := IsAnswerCorrectInLowerCase(c.PostForm("textanswer"), card.Answer)

		gormDB.updateLearningCardByID(card.ID, correct)

		deck, err := (*gormDB).getDeckByID(card.DeckID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.String(http.StatusNotFound, "deck not found")
			} else {
				c.String(http.StatusInternalServerError, "error fetching deck")
			}
		}

		learningCards, err := gormDB.getLearningCardsByDeckID(card.DeckID)
		if err != nil || len(learningCards) == 0 {
			learning.NoneLeft(deck).Render(c.Request.Context(), c.Writer)
			return
		}

		mostDueCard, err := getMostDueCard(learningCards)
		if err != nil {
			c.String(http.StatusBadRequest, "invalid deck id?")
			log.Print(err)
			return
		}

		randomCards, err := gormDB.getShuffledChoicesForCard(card.DeckID, mostDueCard)
		if err != nil {
			c.String(http.StatusBadRequest, "failed to get random cards")
			log.Print(err)
		}

		fmt.Print("text answer form's text was: " + c.PostForm("textanswer"))

		learning.AnswerFeedback(correct, card.Answer, deck, mostDueCard, randomCards).Render(c.Request.Context(), c.Writer)
	})

	r.GET("/card/:cardID/nextlearning", func(c *gin.Context) {
		cardIdStr := c.Param("cardID")
		cardId, _ := strconv.ParseUint(cardIdStr, 10, 32)

		card, _ := gormDB.getCardByID(uint(cardId))
		deck, _ := gormDB.getDeckByID(card.DeckID)

		learningCards, _ := gormDB.getLearningCardsByDeckID(deck.ID)
		if len(learningCards) == 0 {
			learning.NoneLeft(deck).Render(c.Request.Context(), c.Writer)
			return
		}

		mostDueCard, _ := getMostDueCard(learningCards)
		randomCards, _ := gormDB.getShuffledChoicesForCard(deck.ID, mostDueCard)

		learning.InitialContent(mostDueCard, randomCards, deck).Render(c.Request.Context(), c.Writer)
	})

	r.GET("/deck/:deckID/review", func(c *gin.Context) {
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

		reviewCards, err := gormDB.getDueReviewCardsByDeckID(deck.ID)
		if err != nil || len(reviewCards) == 0 {
			review.NoneLeft(deck).Render(c.Request.Context(), c.Writer)
			return
		}

		mostDueCard, err := getMostDueCard(reviewCards)
		if err != nil {
			c.String(http.StatusBadRequest, "not enough review cards?")
			log.Print(err)
		}

		randomCards, err := gormDB.getShuffledChoicesForCard(deck.ID, mostDueCard)
		if err != nil {
			c.String(http.StatusBadRequest, "failed to get random cards")
			log.Print(err)
		}

		review.InitialContent(mostDueCard, randomCards, deck).Render(c.Request.Context(), c.Writer)
	})

	r.POST("/card/:cardID/review", func(c *gin.Context) {
		cardIdStr := c.Param("cardID")
		cardId, err := strconv.ParseUint(cardIdStr, 10, 32)
		if err != nil {
			c.String(http.StatusBadRequest, "Invalid card ID")
			log.Print(err)
			return
		}

		card, err := gormDB.getCardByID(uint(cardId))
		if err != nil {
			c.String(http.StatusBadRequest, "card not found")
			log.Print(err)
			return
		}

		deck, err := (*gormDB).getDeckByID(card.DeckID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.String(http.StatusNotFound, "deck not found")
			} else {
				c.String(http.StatusInternalServerError, "error fetching deck")
			}
		}

		correct := IsAnswerCorrectInLowerCase(c.PostForm("textanswer"), card.Answer)

		gormDB.updateReviewCardByID(card.ID, correct)

		reviewCards, err := gormDB.getDueReviewCardsByDeckID(card.DeckID)
		if err != nil || len(reviewCards) == 0 {
			review.NoneLeft(deck).Render(c.Request.Context(), c.Writer)
			return
		}

		mostDueCard, err := getMostDueCard(reviewCards)
		if err != nil {
			c.String(http.StatusBadRequest, "invalid deck id?")
			log.Print(err)
			return
		}

		randomCards, err := gormDB.getShuffledChoicesForCard(card.DeckID, mostDueCard)
		if err != nil {
			c.String(http.StatusBadRequest, "failed to get random cards")
			log.Print(err)
		}

		review.AnswerFeedback(correct, card.Answer, deck, mostDueCard, randomCards).Render(c.Request.Context(), c.Writer)
	})

	r.GET("/card/:cardID/nextreview", func(c *gin.Context) {
		cardIdStr := c.Param("cardID")
		cardId, _ := strconv.ParseUint(cardIdStr, 10, 32)

		card, _ := gormDB.getCardByID(uint(cardId))
		deck, _ := gormDB.getDeckByID(card.DeckID)

		reviewCards, _ := gormDB.getDueReviewCardsByDeckID(deck.ID)
		if len(reviewCards) == 0 {
			review.NoneLeft(deck).Render(c.Request.Context(), c.Writer)
			return
		}

		mostDueCard, _ := getMostDueCard(reviewCards)
		randomCards, _ := gormDB.getShuffledChoicesForCard(deck.ID, mostDueCard)

		review.InitialContent(mostDueCard, randomCards, deck).Render(c.Request.Context(), c.Writer)
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
		if err != nil {
			log.Print(err)
		}

		decks.LoadDecks(selectedDecks).Render(c.Request.Context(), c.Writer)
	})

	//JSON API

	r.GET("/api/decks", func(c *gin.Context) {

		selectedDecks, err := (*gormDB).selectAllDecks()

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to fetch decks: " + err.Error(),
			})
		}

		c.JSON(http.StatusOK, gin.H{
			"decks": selectedDecks,
		})
	})

	r.GET("/api/deck/:deckID", func(c *gin.Context) {

		deckIdStr := c.Param("deckID")
		deckId, err := strconv.ParseUint(deckIdStr, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid deck ID",
			})
			log.Println("Invalid deck ID:", err)
			return
		}

		selectedDeck, err := (*gormDB).getDeckByID(uint(deckId))

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to fetch deck: " + err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"deck": selectedDeck,
		})
	})

	r.POST("/api/createdeck", func(c *gin.Context) {
		var json struct {
			Name string `json:"name"`
		}

		if err := c.BindJSON(&json); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid JSON",
			})
			return
		}
		deckName := strings.TrimSpace(json.Name)
		if deckName == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "deck name cannot be empty",
			})
		}

		deck := models.Deck{Name: deckName}
		if err := gormDB.db.Create(&deck).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to create deck: " + err.Error(),
			})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"message": "Deck created successfully",
			"deck": gin.H{
				"id":   deck.ID,
				"name": deck.Name,
			},
		})

	})

	r.POST("/api/deck/:deckID/createcard", func(c *gin.Context) {

		deckIdStr := c.Param("deckID")
		deckId, err := strconv.ParseUint(deckIdStr, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid deck ID",
			})
			log.Println("Invalid deck ID:", err)
			return
		}

		var json struct {
			Question string `json:"question"`
			Answer   string `json:"answer"`
		}

		if err := c.ShouldBindJSON(&json); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid JSON payload",
				"details": err.Error(),
			})
			log.Println("JSON binding error:", err)
			return
		}

		json.Question = strings.TrimSpace(json.Question)
		json.Answer = strings.TrimSpace(json.Answer)

		if json.Question == "" || json.Answer == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "question or answer cannot be empty",
			})
			return
		}

		card := models.Card{
			DeckID:        uint(deckId),
			Question:      json.Question,
			Answer:        json.Answer,
			CardCreated:   time.Now().UTC(),
			ReviewDueDate: time.Now().UTC(),
		}

		if err := gormDB.db.Create(&card).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to create card",
				"details": err.Error(),
			})
			log.Println("DB CREATE ERROR: ", err)
			return
		}

		c.JSON(http.StatusCreated, card)

	})

	r.PUT("/api/card/:cardID/edit", func(c *gin.Context) {
		cardIdStr := c.Param("cardID")
		cardId, err := strconv.ParseUint(cardIdStr, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid card ID",
			})
			return
		}

		var json struct {
			Question string `json:"question"`
			Answer   string `json:"answer"`
		}

		if err := c.ShouldBindJSON(&json); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "invalid JSON payload",
				"details": err.Error(),
			})
			return

		}

		json.Question = strings.TrimSpace(json.Question)
		json.Answer = strings.TrimSpace(json.Answer)

		if json.Question == "" || json.Answer == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "question or answer can't be empty",
			})
			return

		}

		err = gormDB.updateCardByID(uint(cardId), json.Question, json.Answer)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "failed to update card",
				"details": err.Error(),
			})
			return
		}

		card, err := gormDB.getCardByID(uint(cardId))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "failed to get card after update",
				"details": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, card)

	})

	r.DELETE("/api/deck/:deckID", func(c *gin.Context) {
		deckIdStr := c.Param("deckID")
		deckId, err := strconv.ParseInt(deckIdStr, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid deck ID",
			})
			return
		}

		err = gormDB.deleteDeckByID(uint(deckId))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to delete deck",
				"details": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Deck deleted successfully",
			"deck_id": deckId,
		})
	})

	r.GET("/api/deck/:deckID/learning", func(c *gin.Context) {
		deckIdStr := c.Param("deckID")
		deckId, err := strconv.ParseInt(deckIdStr, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid deck id",
			})
			return
		}

		deck, err := gormDB.getDeckByID(uint(deckId))
		if err != nil {

			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{
					"error": "Deck not found",
				})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error":   "Error fetching deck",
					"details": err.Error(),
				})
				return
			}
		}

		learningCards, err := gormDB.getLearningCardsByDeckID(deck.ID)
		if err != nil || len(learningCards) == 0 {
			c.JSON(http.StatusOK, gin.H{
				"message": "No learning cards available in this deck",
				"deck":    deck,
				"cards":   []any{},
			})
			return
		}

		mostDueCard, err := getMostDueCard(learningCards)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Could not determine most due card",
				"details": err.Error(),
			})
			return
		}

		randomCards, err := gormDB.getShuffledChoicesForCard(deck.ID, mostDueCard)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "failed to random cards",
				"details": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"deck":                     deck,
			"most_due_card":            mostDueCard,
			"random_cards":             randomCards,
			"number_of_learning_cards": len(learningCards),
		})
	})

	r.POST("/api/card/:cardID/learning", func(c *gin.Context) {
		cardIdStr := c.Param("cardID")
		cardId, err := strconv.ParseUint(cardIdStr, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid card ID"})
			return
		}

		var payload struct {
			Answer string `json:"answer"`
		}

		if err := c.BindJSON(&payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON payload"})
			return
		}

		card, err := gormDB.getCardByID(uint(cardId))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Card not found"})
			return
		}

		deck, err := gormDB.getDeckByID(card.DeckID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Deck not found"})
			return
		}

		var correct bool
		if payload.Answer != "" {
			correct = IsAnswerCorrectInLowerCase(payload.Answer, card.Answer)
			gormDB.updateLearningCardByID(card.ID, correct)
		}

		learningCards, err := gormDB.getLearningCardsByDeckID(deck.ID)
		if err != nil || len(learningCards) == 0 {
			c.JSON(http.StatusOK, gin.H{
				"done":    true,
				"message": "No more learning cards in this deck",
			})
			return
		}

		mostDueCard, err := getMostDueCard(learningCards)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Error while trying to get most due card",
				"details": err.Error(),
			})

			return
		}

		randomCards, err := gormDB.getShuffledChoicesForCard(deck.ID, mostDueCard)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Error while trying to get multiple choice options",
				"details": err.Error(),
			})

			return
		}

		c.JSON(http.StatusOK, gin.H{
			"done":      false,
			"correct":   correct,
			"next_card": mostDueCard,
			"choices":   randomCards,
		})

	})

	r.GET("/api/deck/:deckID/review", func(c *gin.Context) {
		deckIdStr := c.Param("deckID")
		deckID, err := strconv.ParseUint(deckIdStr, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid deck ID"})
			return
		}

		deck, err := gormDB.getDeckByID(uint(deckID))
		if err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, gorm.ErrRecordNotFound) {
				status = http.StatusNotFound
			}
			c.JSON(status, gin.H{"error": "Deck not found"})
			return
		}

		reviewCards, err := gormDB.getDueReviewCardsByDeckID(deck.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Database error while fetching cards",
				"details": err.Error(),
			})
			return
		}

		if len(reviewCards) == 0 {
			c.JSON(http.StatusOK, gin.H{
				"done":  true,
				"deck":  deck,
				"cards": []any{},
				"msg":   "No review cards that are due in this deck",
			})
			return
		}

		mostDue, err := getMostDueCard(reviewCards)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Error while trying to fetch most due card",
				"details": err.Error(),
			})
			return
		}

		choices, err := gormDB.getShuffledChoicesForCard(deck.ID, mostDue)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Error while fetching multiple choice options",
				"details": err.Error(),
			})

			return
		}

		c.JSON(http.StatusFound, gin.H{
			"done":        false,
			"deck":        deck,
			"next_card":   mostDue,
			"choices":     choices,
			"cards_total": len(reviewCards),
		})

	})

	r.POST("/api/card/:cardID/review", func(c *gin.Context) {
		cardIDstr := c.Param("cardID")
		cardID, err := strconv.ParseUint(cardIDstr, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid card ID"})
			return
		}

		card, err := gormDB.getCardByID(uint(cardID))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Card not found"})
			return
		}
		deck, err := gormDB.getDeckByID(card.DeckID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Deck not found"})
			return
		}

		var payload struct {
			Answer string `json:"answer"`
		}

		if err := c.BindJSON(&payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON payload"})
			return
		}

		var correct bool
		if strings.TrimSpace(payload.Answer) != "" {
			correct = IsAnswerCorrectInLowerCase(payload.Answer, card.Answer)
			err := gormDB.updateReviewCardByID(card.ID, correct)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error":   "Error while trying to update review card",
					"details": err.Error(),
				})
				return
			}
		}

		reviewCards, err := gormDB.getDueReviewCardsByDeckID(deck.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Error while trying to get due review cards",
				"details": err.Error(),
			})

			return
		}

		if len(reviewCards) == 0 {
			c.JSON(http.StatusOK, gin.H{
				"done":    true,
				"correct": correct,
				"msg":     "No review cards due in this deck",
			})
			return
		}

		mostDue, err := getMostDueCard(reviewCards)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Error while trying to get most due card",
				"details": err.Error(),
			})

			return
		}

		choices, err := gormDB.getShuffledChoicesForCard(deck.ID, mostDue)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Error while trying to get multiple choice options",
				"details": err.Error(),
			})

			return
		}

		c.JSON(http.StatusOK, gin.H{
			"done":       false,
			"correct":    correct,
			"next_card":  mostDue,
			"choices":    choices,
			"cards_left": len(reviewCards),
		})
	})

	log.Println("Server starting on http://localhost:3030")
	if err := r.Run(":3030"); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}

}
