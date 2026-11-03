package gateway

import (
	"strings"
	"unicode/utf8"
)

// TranslateChunkBytes is the budget for one TranslateText call. The gateway's limit is
// 10,000 bytes of UTF-8, not characters: CJK, Hindi and Thai take 3 bytes a character, so
// a 4,000-character chunk can be 12,000 bytes and is a 400 that is never retried.
const TranslateChunkBytes = 9000

// ChunkByBytes splits text into pieces of at most max bytes, preferring sentence
// boundaries (". ", "? ", "! " followed by whitespace or the end, CJK full stops, newlines),
// then whitespace, and only then, for a run with no break at all, a rune boundary. Invalid
// UTF-8 is replaced first, so the pieces are valid and always make progress. Joining the
// pieces gives back the (sanitised) text. Translate the pieces one call each and join the
// answers.
func ChunkByBytes(text string, max int) []string {
	if max < utf8.UTFMax {
		max = utf8.UTFMax
	}
	text = strings.ToValidUTF8(text, "�")
	var out []string
	for len(text) > max {
		cut := cutPoint(text, max)
		if cut <= 0 {
			cut = max
		}
		out = append(out, text[:cut])
		text = text[cut:]
	}
	if text != "" {
		out = append(out, text)
	}
	return out
}

func cutPoint(text string, max int) int {
	end := max
	for end > 0 && !utf8.RuneStart(text[end]) { // never end the window inside a rune
		end--
	}
	window := text[:end]
	next := text[end:]
	best := -1
	for i := len(window); i > 0; {
		r, size := utf8.DecodeLastRuneInString(window[:i])
		i -= size
		switch r {
		case '。', '？', '！', '\n':
			best = i + size
		case '.', '?', '!':
			// A sentence end only when whitespace (or the end of text) follows.
			after := window[i+size:]
			if after == "" {
				after = next
			}
			if after == "" || after[0] == ' ' || after[0] == '\t' || after[0] == '\n' || after[0] == '\r' {
				best = i + size
			}
		}
		if best > 0 {
			break
		}
	}
	if best > 0 {
		for best < len(window) && (window[best] == ' ' || window[best] == '\t') {
			best++ // keep the separating space with the sentence it follows
		}
	}
	if best <= 0 {
		if i := strings.LastIndexAny(window, " \t"); i > 0 {
			best = i + 1
		}
	}
	if best <= 0 {
		best = len(window)
	}
	return best
}
