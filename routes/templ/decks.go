package routestempl

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"webproject/database"
	"webproject/views/deck"
	"webproject/views/decks"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterTemplDecksRoutes(r *gin.Engine, gormDB *database.GormDB) {
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
}
