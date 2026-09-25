package utils

import (
	"crypto/rand"
	"fmt"
	"time"
)

// 6-digit numeric token, e.g. "482917". 10^6 = 1M combinations; fine for dev.
func GenerateResetToken() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	n := uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])
	return fmt.Sprintf("%06d", int(n%1000000))
}

func NowUnix() int64 {
	return time.Now().Unix()
}