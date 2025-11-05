package game

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
)

const (
	MIN_MULTIPLIER = 1.00
	MAX_MULTIPLIER = 1000000.00
	HOUSE_EDGE     = 0.01 // 1%
)

// HashAndMapToMultiplier generates a provably fair crash multiplier
// using HMAC-SHA256 and exponential distribution
func HashAndMapToMultiplier(serverSeed, clientSeed string, nonce int) float64 {
	data := fmt.Sprintf("%s:%d", clientSeed, nonce)
	h := hmac.New(sha256.New, []byte(serverSeed))
	h.Write([]byte(data))
	hashBytes := h.Sum(nil)
	hashHex := hex.EncodeToString(hashBytes)

	// Take first 16 hex characters (64 bits)
	hexValue := hashHex[:16]
	i := new(big.Int)
	i.SetString(hexValue, 16)

	// Convert to float between 0 and 1
	const MAX_VALUE_F64 = 18446744073709551616.0
	rFloat := float64(i.Uint64()) / MAX_VALUE_F64

	// House edge: 1% instant crash
	if rFloat < HOUSE_EDGE {
		return MIN_MULTIPLIER
	}

	// Exponential distribution formula
	// This creates a fair distribution where higher multipliers are rarer
	crashValue := (100.0 - HOUSE_EDGE*100) / (100.0 - rFloat*100.0)
	
	// Round to 2 decimal places
	finalMultiplier := float64(int(crashValue*100)) / 100.0

	// Clamp to valid range
	if finalMultiplier < MIN_MULTIPLIER {
		return MIN_MULTIPLIER
	}
	if finalMultiplier > MAX_MULTIPLIER {
		return MAX_MULTIPLIER
	}

	return finalMultiplier
}

// GenerateSeed creates a cryptographically secure random seed
func GenerateSeed() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// HashCommitment creates a SHA256 hash of the seed for commitment
func HashCommitment(seed string) string {
	h := sha256.New()
	h.Write([]byte(seed))
	return hex.EncodeToString(h.Sum(nil))
}

// VerifyRound allows players to verify the fairness of a round
func VerifyRound(serverSeed, clientSeed string, nonce int, claimedMultiplier float64) bool {
	calculatedMultiplier := HashAndMapToMultiplier(serverSeed, clientSeed, nonce)
	// Allow small floating point differences
	diff := calculatedMultiplier - claimedMultiplier
	if diff < 0 {
		diff = -diff
	}
	return diff < 0.01
}
