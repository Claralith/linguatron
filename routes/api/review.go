package api

import (
	"net/http"
	"strconv"
	"strings"
	"webproject/database"
	"webproject/models"
	"webproject/spacedrepetition"

	"github.com/gin-gonic/gin"
)

func RegisterReviewRoutes(r *gin.Engine, gormDB *database.GormDB) {

	r.GET("/api/deck/:deckID/review", func(c *gin.Context) {
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

		cards, err := gormDB.GetFirstXCards(deckID, limit, "review")
		if err != nil || len(cards) == 0 {
			c.JSON(http.StatusOK, gin.H{
				"done":  true,
				"deck":  deck,
				"cards": []any{},
				"msg":   "no review cards are due",
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

	r.POST("/api/deck/:deckID/review", func(c *gin.Context) {
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
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload or no cards provided"})
			return
		}

		currentCard := payload.Cards[0]
		remainingCards := payload.Cards

		answerSubmitted := strings.TrimSpace(payload.Answer) != ""
		isCorrect := false

		if answerSubmitted {
			isCorrect = spacedrepetition.IsAnswerCorrectInLowerCase(
				payload.Answer, currentCard.Answer)

			if err := gormDB.UpdateReviewCardByID(currentCard.ID, isCorrect); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error":   "DB update failed",
					"details": err.Error(),
				})
				return
			}
		}
		if isCorrect {
			if len(remainingCards) > 0 {
				remainingCards = remainingCards[1:]
			}
		} else {
			if len(remainingCards) > 1 {
				cardToRequeue := remainingCards[0]
				remainingCards = append(remainingCards[1:], cardToRequeue)
			}
		}

		if len(remainingCards) == 0 {
			c.JSON(http.StatusOK, gin.H{
				"done":    true,
				"correct": isCorrect,
				"cards":   []any{},
				"choices": []any{},
			})
			return
		}

		nextCardToShow := remainingCards[0]
		choices, err := gormDB.GetShuffledChoicesForCard(deckID, nextCardToShow)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Error while trying to get multiple choice options",
				"details": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"done":       false,
			"correct":    isCorrect,
			"cards":      remainingCards,
			"current":    nextCardToShow,
			"choices":    choices,
			"cards_left": len(remainingCards),
		})
	})
}
