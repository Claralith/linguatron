package api

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"webproject/database"
	"webproject/spacedrepetition"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterReviewRoutes(r *gin.Engine, gormDB *database.GormDB) {

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
}
