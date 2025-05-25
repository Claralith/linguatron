package api

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
	"webproject/database"
	"webproject/models"

	"github.com/gin-gonic/gin"
)

func RegisterSetupRoutes(r *gin.Engine, gormDB *database.GormDB) {

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

	r.GET("/api/deck/:deckID/cards", func(c *gin.Context) {
		deckIdStr := c.Param("deckID")
		deckId, err := strconv.ParseInt(deckIdStr, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid deck ID",
			})
			return
		}

		deck, err := (*gormDB).GetDeckByID(uint(deckId))

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to fetch deck: " + err.Error(),
			})
			return
		}

		cards, err := gormDB.GetAllCardsByDeckID(deck.ID)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"error":   "No cards in this deck",
				"details": err.Error(),
				"cards":   []any{},
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"cards": cards,
		})

	})

	r.DELETE("/api/card/:cardID/delete", func(c *gin.Context) {
		cardIdStr := c.Param("cardID")
		cardId, err := strconv.ParseUint(cardIdStr, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid card ID",
			})
			return
		}

		err = gormDB.DeleteCardByID(uint(cardId))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to delete card",
				"details": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Card deleted successfully",
			"card_id": cardId,
		})

	})

	r.POST("/api/deck/:deckID/batchadd", func(c *gin.Context) {
		deckIdStr := c.Param("deckID")
		deckId, err := strconv.ParseUint(deckIdStr, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid deck ID"})
			return
		}

		var json struct {
			Lines []string `json:"lines"`
		}
		if err := c.ShouldBindJSON(&json); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON", "details": err.Error()})
			return
		}

		var numberOfCards int
		for _, line := range json.Lines {
			parts := strings.Split(line, ";")
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
			err := gormDB.CreateCard(card)

			if err == nil {
				numberOfCards++
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error":   "There was an error adding batch of cards",
					"details": err.Error(),
				})
				return
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"message":           "Batch add completed",
			"cards_added_count": numberOfCards,
		})
	})
}
