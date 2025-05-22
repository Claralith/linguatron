package api

import (
	"log"
	"net/http"
	"strconv"
	"webproject/database"

	"github.com/gin-gonic/gin"
)

func RegisterDecksRoutes(r *gin.Engine, gormDB *database.GormDB) {
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
}
