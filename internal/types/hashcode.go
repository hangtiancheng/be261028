package types

import (
	"crypto/md5"
	"math"
	"math/big"
	"unicode/utf16"
)

// JavaStringHashCode reproduces java.lang.String.hashCode() exactly, including
// 32-bit signed overflow and UTF-16 code-unit iteration. The legacy DCCService
// uses this to bucket users for traffic cutting, so byte-for-byte parity keeps
// the same users in range after migration.
func JavaStringHashCode(s string) int32 {
	units := utf16.Encode([]rune(s))
	var h int32
	for _, u := range units {
		h = 31*h + int32(u)
	}
	return h
}

// JavaAbs reproduces java.lang.Math.abs(int), including the Integer.MIN_VALUE
// edge case where the result stays negative under two's complement.
func JavaAbs(n int32) int32 {
	if n < 0 {
		return -n
	}
	return n
}

// IndexFromUserId reproduces IRedisService.getIndexFromUserId: MD5 of the UTF-8
// bytes interpreted as an unsigned big-endian integer, modulo Integer.MAX_VALUE.
// It is the Redis bitmap offset used for crowd-tag membership.
func IndexFromUserId(userID string) int64 {
	sum := md5.Sum([]byte(userID))
	n := new(big.Int).SetBytes(sum[:])
	mod := new(big.Int).Mod(n, big.NewInt(math.MaxInt32))
	return mod.Int64()
}
