package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
	"webproject/database"
	"webproject/spacedrepetition"
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
		err := gormDB.DB.Create(&deck).Error
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

		gormDB.DeleteDeckByID(uint(deckId))

		selectedDecks, err := (*gormDB).SelectAllDecks()
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

		deck, err := (*gormDB).GetDeckByID(uint(deckId))
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.String(http.StatusNotFound, "deck not found")
			} else {
				c.String(http.StatusInternalServerError, "error fetching deck")
			}
		}

		cards, err := gormDB.GetAllCardsByDeckID(uint(deckId))
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

		(*gormDB).CreateCard(card)

		cards, err := gormDB.GetAllCardsByDeckID(uint(deckId))
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

		deck, err := (*gormDB).GetDeckByID(uint(deckId))
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

		deck, err := (*gormDB).GetDeckByID(uint(deckId))
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
			gormDB.CreateCard(card)
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

		card, err := gormDB.GetCardByID(uint(cardId))
		if err != nil {
			c.String(http.StatusBadRequest, "Invalid card ID")
			log.Print(err)
			return
		}

		gormDB.DeleteCardByID(card.ID)

		cards, err := gormDB.GetAllCardsByDeckID(uint(card.DeckID))
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

		gormDB.UpdateCardByID(uint(cardId), c.PostForm("question"), c.PostForm("answer"))

		card, err := gormDB.GetCardByID(uint(cardId))
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

		deck, err := (*gormDB).GetDeckByID(uint(deckId))
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.String(http.StatusNotFound, "deck not found")
			} else {
				c.String(http.StatusInternalServerError, "error fetching deck")
			}
		}

		learningCards, err := gormDB.GetLearningCardsByDeckID(deck.ID)
		if err != nil || len(learningCards) == 0 {
			learning.NoneLeft(deck).Render(c.Request.Context(), c.Writer)
			return
		}

		mostDueCard, err := spacedrepetition.GetMostDueCard(learningCards)
		if err != nil {
			c.String(http.StatusBadRequest, "not enough learning cards?")
			log.Print(err)
		}

		randomCards, err := gormDB.GetShuffledChoicesForCard(deck.ID, mostDueCard)
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

		card, err := gormDB.GetCardByID(uint(cardId))
		if err != nil {
			c.String(http.StatusBadRequest, "card not found")
			log.Print(err)
			return
		}

		correct := spacedrepetition.IsAnswerCorrectInLowerCase(c.PostForm("textanswer"), card.Answer)

		gormDB.UpdateLearningCardByID(card.ID, correct)

		deck, err := (*gormDB).GetDeckByID(card.DeckID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.String(http.StatusNotFound, "deck not found")
			} else {
				c.String(http.StatusInternalServerError, "error fetching deck")
			}
		}

		learningCards, err := gormDB.GetLearningCardsByDeckID(card.DeckID)
		if err != nil || len(learningCards) == 0 {
			learning.NoneLeft(deck).Render(c.Request.Context(), c.Writer)
			return
		}

		mostDueCard, err := spacedrepetition.GetMostDueCard(learningCards)
		if err != nil {
			c.String(http.StatusBadRequest, "invalid deck id?")
			log.Print(err)
			return
		}

		randomCards, err := gormDB.GetShuffledChoicesForCard(card.DeckID, mostDueCard)
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

		card, _ := gormDB.GetCardByID(uint(cardId))
		deck, _ := gormDB.GetDeckByID(card.DeckID)

		learningCards, _ := gormDB.GetLearningCardsByDeckID(deck.ID)
		if len(learningCards) == 0 {
			learning.NoneLeft(deck).Render(c.Request.Context(), c.Writer)
			return
		}

		mostDueCard, _ := spacedrepetition.GetMostDueCard(learningCards)
		randomCards, _ := gormDB.GetShuffledChoicesForCard(deck.ID, mostDueCard)

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

		deck, err := (*gormDB).GetDeckByID(uint(deckId))
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.String(http.StatusNotFound, "deck not found")
			} else {
				c.String(http.StatusInternalServerError, "error fetching deck")
			}
		}

		reviewCards, err := gormDB.GetDueReviewCardsByDeckID(deck.ID)
		if err != nil || len(reviewCards) == 0 {
			review.NoneLeft(deck).Render(c.Request.Context(), c.Writer)
			return
		}

		mostDueCard, err := spacedrepetition.GetMostDueCard(reviewCards)
		if err != nil {
			c.String(http.StatusBadRequest, "not enough review cards?")
			log.Print(err)
		}

		randomCards, err := gormDB.GetShuffledChoicesForCard(deck.ID, mostDueCard)
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

		card, err := gormDB.GetCardByID(uint(cardId))
		if err != nil {
			c.String(http.StatusBadRequest, "card not found")
			log.Print(err)
			return
		}

		deck, err := (*gormDB).GetDeckByID(card.DeckID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.String(http.StatusNotFound, "deck not found")
			} else {
				c.String(http.StatusInternalServerError, "error fetching deck")
			}
		}

		correct := spacedrepetition.IsAnswerCorrectInLowerCase(c.PostForm("textanswer"), card.Answer)

		gormDB.UpdateReviewCardByID(card.ID, correct)

		reviewCards, err := gormDB.GetDueReviewCardsByDeckID(card.DeckID)
		if err != nil || len(reviewCards) == 0 {
			review.NoneLeft(deck).Render(c.Request.Context(), c.Writer)
			return
		}

		mostDueCard, err := spacedrepetition.GetMostDueCard(reviewCards)
		if err != nil {
			c.String(http.StatusBadRequest, "invalid deck id?")
			log.Print(err)
			return
		}

		randomCards, err := gormDB.GetShuffledChoicesForCard(card.DeckID, mostDueCard)
		if err != nil {
			c.String(http.StatusBadRequest, "failed to get random cards")
			log.Print(err)
		}

		review.AnswerFeedback(correct, card.Answer, deck, mostDueCard, randomCards).Render(c.Request.Context(), c.Writer)
	})

	r.GET("/card/:cardID/nextreview", func(c *gin.Context) {
		cardIdStr := c.Param("cardID")
		cardId, _ := strconv.ParseUint(cardIdStr, 10, 32)

		card, _ := gormDB.GetCardByID(uint(cardId))
		deck, _ := gormDB.GetDeckByID(card.DeckID)

		reviewCards, _ := gormDB.GetDueReviewCardsByDeckID(deck.ID)
		if len(reviewCards) == 0 {
			review.NoneLeft(deck).Render(c.Request.Context(), c.Writer)
			return
		}

		mostDueCard, _ := spacedrepetition.GetMostDueCard(reviewCards)
		randomCards, _ := gormDB.GetShuffledChoicesForCard(deck.ID, mostDueCard)

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

		deckData, err := gormDB.GetDeckByID(uint(deckId))
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

		selectedDecks, err := (*gormDB).SelectAllDecks()
		if err != nil {
			log.Print(err)
		}

		decks.LoadDecks(selectedDecks).Render(c.Request.Context(), c.Writer)
	})

	//JSON API

	r.GET("/api/decks", func(c *gin.Context) {

		selectedDecks, err := (*gormDB).SelectAllDecks()

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

		selectedDeck, err := (*gormDB).GetDeckByID(uint(deckId))

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
		if err := gormDB.DB.Create(&deck).Error; err != nil {
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

		if err := gormDB.DB.Create(&card).Error; err != nil {
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

		err = gormDB.UpdateCardByID(uint(cardId), json.Question, json.Answer)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "failed to update card",
				"details": err.Error(),
			})
			return
		}

		card, err := gormDB.GetCardByID(uint(cardId))
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

		err = gormDB.DeleteDeckByID(uint(deckId))
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

		deck, err := gormDB.GetDeckByID(uint(deckId))
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

		learningCards, err := gormDB.GetLearningCardsByDeckID(deck.ID)
		if err != nil || len(learningCards) == 0 {
			c.JSON(http.StatusOK, gin.H{
				"message": "No learning cards available in this deck",
				"deck":    deck,
				"cards":   []any{},
			})
			return
		}

		mostDueCard, err := spacedrepetition.GetMostDueCard(learningCards)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Could not determine most due card",
				"details": err.Error(),
			})
			return
		}

		randomCards, err := gormDB.GetShuffledChoicesForCard(deck.ID, mostDueCard)
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

		card, err := gormDB.GetCardByID(uint(cardId))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Card not found"})
			return
		}

		deck, err := gormDB.GetDeckByID(card.DeckID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Deck not found"})
			return
		}

		var correct bool
		if payload.Answer != "" {
			correct = spacedrepetition.IsAnswerCorrectInLowerCase(payload.Answer, card.Answer)
			gormDB.UpdateLearningCardByID(card.ID, correct)
		}

		learningCards, err := gormDB.GetLearningCardsByDeckID(deck.ID)
		if err != nil || len(learningCards) == 0 {
			c.JSON(http.StatusOK, gin.H{
				"done":    true,
				"message": "No more learning cards in this deck",
			})
			return
		}

		mostDueCard, err := spacedrepetition.GetMostDueCard(learningCards)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Error while trying to get most due card",
				"details": err.Error(),
			})

			return
		}

		randomCards, err := gormDB.GetShuffledChoicesForCard(deck.ID, mostDueCard)
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

		deck, err := gormDB.GetDeckByID(uint(deckID))
		if err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, gorm.ErrRecordNotFound) {
				status = http.StatusNotFound
			}
			c.JSON(status, gin.H{"error": "Deck not found"})
			return
		}

		reviewCards, err := gormDB.GetDueReviewCardsByDeckID(deck.ID)
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

		mostDue, err := spacedrepetition.GetMostDueCard(reviewCards)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Error while trying to fetch most due card",
				"details": err.Error(),
			})
			return
		}

		choices, err := gormDB.GetShuffledChoicesForCard(deck.ID, mostDue)
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

		card, err := gormDB.GetCardByID(uint(cardID))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Card not found"})
			return
		}
		deck, err := gormDB.GetDeckByID(card.DeckID)
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
			correct = spacedrepetition.IsAnswerCorrectInLowerCase(payload.Answer, card.Answer)
			err := gormDB.UpdateReviewCardByID(card.ID, correct)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error":   "Error while trying to update review card",
					"details": err.Error(),
				})
				return
			}
		}

		reviewCards, err := gormDB.GetDueReviewCardsByDeckID(deck.ID)
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

		mostDue, err := spacedrepetition.GetMostDueCard(reviewCards)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Error while trying to get most due card",
				"details": err.Error(),
			})

			return
		}

		choices, err := gormDB.GetShuffledChoicesForCard(deck.ID, mostDue)
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
