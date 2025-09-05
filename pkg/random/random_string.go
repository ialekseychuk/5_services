package random

import (
	"math/rand"
	"time"
)

var timeSeeder *rand.Rand = rand.New(
	rand.NewSource(
		time.Now().UnixNano(),
	),
)
const charset = "abcdefghijklmnopqrstuvwxyz" +
  "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

 func RandomString(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[timeSeeder.Intn(len(charset))]
	}
	return string(b)
} 