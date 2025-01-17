package util

import (
	"math/rand"
	"time"
)

var rnd *rand.Rand

func init() {
	rnd = rand.New(rand.NewSource(time.Now().UnixNano()))
}

func RandomUInt32() uint32 {
	return rnd.Uint32()
}

func RandomUInt32n(n uint32) uint32 {
	return uint32(rnd.Int63n(int64(n)) >> 31)
}

func RandomUInt64() uint64 {
	return rnd.Uint64()
}

const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
const letterIdxBits = 6                    // 6 bits to represent a letter index
const letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
const letterIdxMax = 63 / letterIdxBits    // # of letter indices fitting in 63 bits

func RandString(n int) string {
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, c, remain := n-1, rnd.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			c, remain = rnd.Int63(), letterIdxMax
		}
		if idx := int(c & letterIdxMask); idx < len(chars) {
			b[i] = chars[idx]
			i--
		}
		c >>= letterIdxBits
		remain--
	}

	return string(b)
}
