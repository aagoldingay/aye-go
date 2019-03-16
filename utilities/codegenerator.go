package utilities

import (
	"math/rand"
	"time"
)

const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// Setup initiates a random seed, or, providing -1, is randomly selected
func Setup(seed int) {
	if seed > -1 {
		rand.Seed(int64(seed))
		return
	}
	rand.Seed(time.Now().UnixNano())
}

// GenerateCode creates a random string using the constant: letters
// l : length of code
func GenerateCode(l int) string {
	b := make([]byte, l)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
