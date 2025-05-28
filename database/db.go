package database

import (
	"math/rand/v2"
	"time"
	"webproject/models"
	"webproject/spacedrepetition"

	"gorm.io/gorm"
)

type GormDB struct {
	DB *gorm.DB
}

func (g *GormDB) CreateDeck(name string) error {
	var deck models.Deck
	deck.Name = name
	return g.DB.Create(&deck).Error
}

func (g *GormDB) CreateCard(card models.Card) error {
	return g.DB.Create(&card).Error
}

func (g *GormDB) GetCardByID(id uint) (models.Card, error) {
	var card models.Card
	err := g.DB.First(&card, id).Error
	return card, err
}

func (g *GormDB) GetAllCardsByDeckID(id uint) ([]models.Card, error) {
	var cards []models.Card
	err := g.DB.Where("deck_id = ?", id).Find(&cards).Error
	return cards, err
}

func (g *GormDB) GetShuffledChoicesForCard(deckID uint, mostDueCard models.Card) ([]models.Card, error) {
	var count int64
	cardCountError := g.DB.Model(&models.Card{}).Where("deck_id = ?", deckID).Count(&count).Error
	if cardCountError != nil {
		return nil, cardCountError
	}

	var limit int

	if count > 1 && count < 5 {
		limit = int(count - 1) //as many false answers as possible
	} else if count >= 5 {
		limit = 5
	} else {
		limit = 0
	}

	var falseAnswers []models.Card
	err := g.DB.Where("deck_id = ? AND id != ?", deckID, mostDueCard.ID).
		Order("RANDOM()").
		Limit(limit).
		Find(&falseAnswers).Error
	if err != nil {
		return nil, err
	}

	falseandCorrect := append(falseAnswers, mostDueCard)
	rand.Shuffle(len(falseandCorrect), func(i, j int) {
		falseandCorrect[i], falseandCorrect[j] = falseandCorrect[j], falseandCorrect[i]
	})

	return falseandCorrect, err
}

func (g *GormDB) GetLearningCardsByDeckID(id uint) ([]models.Card, error) {
	var cards []models.Card
	err := g.DB.Where("deck_id = ? AND stage = ?", id, "learning").Find(&cards).Error
	return cards, err
}
func (g *GormDB) GetReviewCardsByDeckID(id uint) ([]models.Card, error) {
	var cards []models.Card
	err := g.DB.Where("deck_id = ? AND stage = ?", id, "review").Find(&cards).Error
	return cards, err
}
func (g *GormDB) GetDueReviewCardsByDeckID(id uint) ([]models.Card, error) {
	now := time.Now().UTC()

	var cards []models.Card
	err := g.DB.Where("deck_id = ? AND stage = ? AND review_due_date <= ?", id, "review", now).Find(&cards).Error
	return cards, err
}

func (g *GormDB) GetFirstXCards(deckID uint, limit int, cardStage string) ([]models.Card, error) {
	var cards []models.Card
	err := g.DB.
		Where("deck_id = ? AND stage = ?", deckID, cardStage).
		Order("review_due_date ASC").
		Limit(limit).
		Find(&cards).Error
	return cards, err
}

func (g *GormDB) GetDeckByID(id uint) (models.Deck, error) {
	var deck models.Deck
	err := g.DB.First(&deck, id).Error
	return deck, err
}

func (g *GormDB) SelectAllDecks() ([]models.Deck, error) {
	var decks []models.Deck
	err := g.DB.Find(&decks).Error
	return decks, err
}

func (g *GormDB) UpdateLearningCardByID(id uint, correct bool) error {
	card, _ := g.GetCardByID(id)

	now := time.Now().UTC()
	shortDelay := now.Add(1 * time.Minute)
	initialReviewDelay := now.Add(4 * time.Hour)

	card.LastReviewDate = now

	if correct {
		card.Correct++
		if card.Ease > 1 {
			card.Ease = uint(spacedrepetition.GetNextEaseLevel(int(card.Ease), 1))
			card.Stage = "review"
			card.ReviewDueDate = initialReviewDelay
		} else {
			card.Ease = uint(spacedrepetition.GetNextEaseLevel(int(card.Ease), 2))
			card.ReviewDueDate = shortDelay
		}
	} else {
		card.Incorrect++
		card.Ease = 1
		card.ReviewDueDate = shortDelay
	}

	return g.DB.Save(&card).Error
}

func (g *GormDB) UpdateReviewCardByID(id uint, correct bool) error {
	card, _ := g.GetCardByID(id)
	now := time.Now().UTC()
	shortDelay := now.Add(1 * time.Minute)

	card.LastReviewDate = now

	if correct {
		card.Correct++
		card.Ease = uint(spacedrepetition.GetNextEaseLevel(int(card.Ease), 2))
		card.ReviewDueDate = spacedrepetition.CreateNextReviewDueDate(int(card.Ease))
	} else {
		card.Incorrect++
		card.ReviewDueDate = shortDelay
		if card.Ease != 1 {
			card.Lapses++
			card.Ease = 1
		}
	}

	return g.DB.Save(&card).Error
}

func (g *GormDB) UpdateCardByID(id uint, question string, answer string, extra string) error {
	card, _ := g.GetCardByID(id)
	card.Question = question
	card.Answer = answer
	card.Extra = extra

	return g.DB.Save(&card).Error
}

func (g *GormDB) DeleteCardByID(id uint) error {
	card, err := g.GetCardByID(id)
	if err != nil {
		return err
	}

	return g.DB.Delete(card).Error
}

func (g *GormDB) DeleteDeckByID(id uint) error {
	card, err := g.GetDeckByID(id)
	if err != nil {
		return err
	}

	return g.DB.Delete(card).Error
}
