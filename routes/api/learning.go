package api

import (
	"net/http"
	"strconv"
	"webproject/database"
	"webproject/models"
	"webproject/spacedrepetition"

	"github.com/gin-gonic/gin"
)

const defaultSessionLimit = 10

func RegisterLearningRoutes(r *gin.Engine, gormDB *database.GormDB) {

	r.GET("/api/deck/:deckID/learning", func(c *gin.Context) {
		deckIDStr, err := strconv.ParseUint(c.Param("deckID"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid deck ID"})
			return
		}
		deckID := uint(deckIDStr)

		deck, err := gormDB.GetDeckByID(deckID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "deck not found"})
			return
		}

		limit := defaultSessionLimit
		if q := c.Query("limit"); q != "" {
			if n, err := strconv.Atoi(q); err == nil && n > 0 {
				limit = n
			}
		}

		cards, err := gormDB.GetFirstXCards(deckID, limit, "learning")
		if err != nil || len(cards) == 0 {
			c.JSON(http.StatusOK, gin.H{
				"done":    true,
				"message": "no learning cards available",
			})
			return
		}

		first := cards[0]
		choices, err := gormDB.GetShuffledChoicesForCard(deckID, first)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Error while fetching multiple choice options",
				"details": err.Error(),
			})

			return
		}

		c.JSON(http.StatusOK, gin.H{
			"deck":    deck,
			"cards":   cards,
			"current": first,
			"choices": choices,
		})
	})
	r.POST("/api/deck/:deckID/learning", func(c *gin.Context) {
		deckIDStr, err := strconv.ParseUint(c.Param("deckID"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid deck ID"})
			return
		}
		deckID := uint(deckIDStr)

		var payload struct {
			Answer string        `json:"answer"`
			Cards  []models.Card `json:"cards"`
		}
		if err := c.ShouldBindJSON(&payload); err != nil || len(payload.Cards) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
			return
		}

		current := payload.Cards[0]

		correct := spacedrepetition.IsAnswerCorrectInLowerCase(
			payload.Answer, current.Answer)

		if err := gormDB.UpdateLearningCardByID(current.ID, correct); err != nil {
			c.JSON(http.StatusInternalServerError,
				gin.H{"error": "DB update failed", "details": err.Error()})
			return
		}

		remaining := payload.Cards
		if correct {
			remaining = remaining[1:]
		}

		if len(remaining) == 0 {
			c.JSON(http.StatusOK, gin.H{
				"done":    true,
				"correct": correct,
			})
			return
		}

		next := remaining[0]
		choices, err := gormDB.GetShuffledChoicesForCard(deckID, next)
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
			"cards":      remaining,
			"current":    next,
			"choices":    choices,
			"cards_left": len(remaining),
		})
	})
}
