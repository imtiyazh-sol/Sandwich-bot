package utils

import (
	"math/rand"
	"time"
)

func GenerateTwoUniqueRandomNumbers(min, max int) (int, int) {
	rand.Seed(time.Now().UnixNano())
	first := rand.Intn(max-min+1) + min
	second := rand.Intn(max-min+1) + min

	// Ensure second is different from first
	for second == first {
		second = rand.Intn(max-min+1) + min
	}

	return first, second
}
