package autonomy

import (
	"crypto/sha1"
	"encoding/hex"
	"strings"
)

func shortHash(parts ...string) string {
	sum := sha1.Sum([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(sum[:8])
}
