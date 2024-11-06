package utils

import (
	"strings"
	"time"

	"github.com/brianvoe/gofakeit/v6"
)

var GenerateMnemonic = func(numWords int) string {
	gofakeit.Seed(time.Now().UnixNano())

	wordSet := make(map[string]struct{})
	mnemonicWords := []string{}

	for len(mnemonicWords) < numWords {
		word := gofakeit.Word()
		if _, exists := wordSet[word]; !exists {
			wordSet[word] = struct{}{}
			mnemonicWords = append(mnemonicWords, word)
		}
	}

	return strings.ToLower(strings.Join(mnemonicWords, " "))
}
