package routestempl

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"webproject/database"
	"webproject/spacedrepetition"
	"webproject/views/learning"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterTemplLearningRoutes(r *gin.Engine, gormDB *database.GormDB) {
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
}
