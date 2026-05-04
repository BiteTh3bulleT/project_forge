package contextcompiler

import (
	"crypto/sha256"
	"encoding/hex"
	"unicode"
)

func SHA256Text(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}

func ContentHash(block ContextBlock) string {
	return SHA256Text(SerializeBlock(block))
}

// TokenInputHash is tokenizer-neutral in Phase 7: it hashes the exact text that
// would be sent to a tokenizer. Phase 8 can add model/tokenizer-specific
// identity gates before any deterministic KV cache reuse.
func TokenInputHash(canonicalText string) string {
	return SHA256Text(canonicalText)
}

func EstimateTokens(text string) int {
	if text == "" {
		return 0
	}
	words := 0
	inWord := false
	punctuation := 0
	for _, r := range text {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			if !inWord {
				words++
				inWord = true
			}
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			inWord = false
			punctuation++
		default:
			inWord = false
		}
	}
	byChars := (len([]rune(text)) + 3) / 4
	estimate := words + (punctuation+3)/4
	if estimate < byChars {
		estimate = byChars
	}
	if estimate < 1 {
		return 1
	}
	return estimate
}
