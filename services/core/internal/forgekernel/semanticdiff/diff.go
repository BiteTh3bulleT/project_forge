// Package semanticdiff implements the production deterministic semantic
// difference operator. It is deliberately pure: durable authority remains at
// the FORGE-K syscall boundary that resolves and seals admitted inputs.
package semanticdiff

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/cases"
	"golang.org/x/text/unicode/norm"
)

const (
	OperatorVersion = "semantic.diff.v1"
	MaxInputBytes   = 64 * 1024
	MaxTokens       = 4096
	MaxTokenRunes   = 256
)

var (
	ErrInvalidUTF8 = errors.New("semantic diff input must be valid UTF-8")
	ErrInputBound  = errors.New("semantic diff input exceeds deterministic bounds")
)

type Input struct {
	Left  string `json:"left"`
	Right string `json:"right"`
}

type Result struct {
	OperatorVersion string   `json:"operatorVersion"`
	Tokens          []string `json:"tokens"`
	Content         string   `json:"content"`
	ContentHash     string   `json:"contentHash"`
}

// Compute returns the sorted set difference left-minus-right after NFKC
// normalization, Unicode case folding, and Unicode letter/number tokenizing.
// Marks are retained only inside an existing token. All other runes delimit.
func Compute(input Input) (Result, error) {
	left, err := tokenSet(input.Left)
	if err != nil {
		return Result{}, err
	}
	right, err := tokenSet(input.Right)
	if err != nil {
		return Result{}, err
	}
	tokens := make([]string, 0, len(left))
	for token := range left {
		if _, found := right[token]; !found {
			tokens = append(tokens, token)
		}
	}
	sort.Strings(tokens)
	content := strings.Join(tokens, " ")
	digest := sha256.Sum256([]byte(content))
	return Result{
		OperatorVersion: OperatorVersion,
		Tokens:          tokens,
		Content:         content,
		ContentHash:     "sha256:" + hex.EncodeToString(digest[:]),
	}, nil
}

func Fingerprint(value any) (string, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(digest[:]), nil
}

func tokenSet(value string) (map[string]struct{}, error) {
	if !utf8.ValidString(value) {
		return nil, ErrInvalidUTF8
	}
	if len(value) > MaxInputBytes {
		return nil, ErrInputBound
	}
	normalized := cases.Fold().String(norm.NFKC.String(value))
	tokens := make(map[string]struct{})
	current := make([]rune, 0, 32)
	count := 0
	flush := func() error {
		if len(current) == 0 {
			return nil
		}
		count++
		if count > MaxTokens || len(current) > MaxTokenRunes {
			return ErrInputBound
		}
		tokens[string(current)] = struct{}{}
		current = current[:0]
		return nil
	}
	for _, r := range normalized {
		if unicode.IsLetter(r) || unicode.IsNumber(r) || (len(current) > 0 && unicode.IsMark(r)) {
			current = append(current, r)
			if len(current) > MaxTokenRunes {
				return nil, ErrInputBound
			}
			continue
		}
		if err := flush(); err != nil {
			return nil, err
		}
	}
	if err := flush(); err != nil {
		return nil, err
	}
	return tokens, nil
}
