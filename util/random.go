package util

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

const alphabet = "abcdefghijklmnopqrstuvwxyz"

func init() {
	rand.Seed(time.Now().UnixNano())
}

// RandomInt generates a random integer between min and max
func RandomInt(min, max int32) int32 {
	return min + rand.Int31n(max-min+1)
}

// RandomString generates a random string of length n
func RandomString(n int) string {
	var sb strings.Builder
	k := len(alphabet)

	for i := 0; i < n; i++ {
		c := alphabet[rand.Intn(k)]
		sb.WriteByte(c)
	}

	return sb.String()
}

// RandomUser generates a random username
func RandomUser() string {
	return RandomString(6)
}

// RandomCurrency generates a random currency code
func RandomStock() string {
	// currencies := []string{"AAPL", "META", "MSFT", "TSLA", "GOOG", "NVDA", "AVGO", "QCOM", "INTC", "AMZN"}
	// n := len(currencies)
	// return currencies[rand.Intn(n)]
	return strings.ToUpper(RandomString(4))
}

// RandomEmail generates a random email
func RandomEmail() string {
	return fmt.Sprintf("%s@email.com", RandomString(6))
}

func RandomFloats() float64 {
	return rand.Float64()
}
