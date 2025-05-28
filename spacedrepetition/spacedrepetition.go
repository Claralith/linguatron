package spacedrepetition

import (
	"fmt"
	"math"
	"strings"
	"time"
	"webproject/models"
)

func IsAnswerCorrectInLowerCase(userAnswer string, databaseAnswer string) bool {
	return strings.EqualFold(strings.TrimSpace(userAnswer), (strings.TrimSpace(databaseAnswer)))
}

func GetMostDueCard(cards []models.Card) (models.Card, error) {
	if len(cards) == 0 {
		return models.Card{}, fmt.Errorf("no cards")
	}

	mostDueCard := cards[0]

	for i := 1; i < len(cards); i++ {
		if cards[i].ReviewDueDate.Before(mostDueCard.ReviewDueDate) {
			mostDueCard = cards[i]
		}
	}

	return mostDueCard, nil
}
func IsCardsNotEmpty(cards []models.Card) bool {
	if len(cards) > 0 {
		return true
	} else {
		return false
	}
}

func GetNextEaseLevel(currentEase int, growthfactor float64) int {
	nextEase := int(math.Ceil(float64(currentEase) * growthfactor))

	return nextEase
}

// Used for cards already in review
func CreateNextReviewDueDate(ease int) time.Time {
	// Base review delay
	base := 4.0 // hours
	delay := time.Duration(base*math.Pow(float64(ease), 1.1)) * time.Hour
	return time.Now().UTC().Add(delay)
}
