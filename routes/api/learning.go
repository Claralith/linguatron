package api

import (
	"errors"
	"net/http"
	"strconv"
	"webproject/database"
	"webproject/spacedrepetition"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterLearningRoutes(r *gin.Engine, gormDB *database.GormDB) {

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
}
