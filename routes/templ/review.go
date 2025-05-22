package routestempl

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"webproject/database"
	"webproject/spacedrepetition"
	"webproject/views/review"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterTemplReviewRoutes(r *gin.Engine, gormDB *database.GormDB) {
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
}
