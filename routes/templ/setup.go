package routestempl

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
	"webproject/database"
	"webproject/models"
	"webproject/views/batchadd"
	"webproject/views/createcard"
	"webproject/views/createdeck"
	"webproject/views/decks"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterTemplSetupRoutes(r *gin.Engine, gormDB *database.GormDB) {
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
}
