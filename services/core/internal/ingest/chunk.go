package ingest

import (
	"strings"
	"unicode/utf8"
)

const maxChunkRunes = 4000
const minChunkRunes = 400

func ChunkText(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	if strings.TrimSpace(s) == "" {
		return nil
	}
	paras := strings.Split(s, "\n\n")
	var chunks []string
	var b strings.Builder
	runesInBuf := 0
	flush := func() {
		t := strings.TrimSpace(b.String())
		if t == "" {
			b.Reset()
			runesInBuf = 0
			return
		}
		chunks = append(chunks, t)
		b.Reset()
		runesInBuf = 0
	}
	for _, p := range paras {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		pr := utf8.RuneCountInString(p)
		if runesInBuf+pr > maxChunkRunes && runesInBuf >= minChunkRunes {
			flush()
		}
		if b.Len() > 0 {
			b.WriteString("\n\n")
			runesInBuf += 2
		}
		b.WriteString(p)
		runesInBuf += pr
		if runesInBuf >= maxChunkRunes {
			flush()
		}
	}
	flush()
	if len(chunks) == 0 && strings.TrimSpace(s) != "" {
		return []string{strings.TrimSpace(s)}
	}
	return chunks
}
