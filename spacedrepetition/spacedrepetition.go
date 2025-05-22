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

func CreateNextReviewDueDate(ease int) time.Time {

	t := time.Now().UTC()
	hours := ease * 24
	duration := time.Duration(hours) * time.Hour

	return t.Add(duration)

}
