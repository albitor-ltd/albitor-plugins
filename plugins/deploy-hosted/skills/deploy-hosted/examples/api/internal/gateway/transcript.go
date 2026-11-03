package gateway

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode"
)

// TranscriptResult is the subset of a Transcribe output file (the GetTranscriptionJob
// Result, or the JSON fetched from Transcript.TranscriptFileUri) the app needs. The field
// names follow AWS's documented output.
type TranscriptResult struct {
	Results struct {
		// Transcripts holds the whole recognised text (AWS documents one entry). Use it as
		// the text to translate or summarise: never rebuild the text from Items.
		Transcripts []struct {
			Transcript string `json:"transcript"`
		} `json:"transcripts"`
		// Items is one entry per word or punctuation mark, in order. It is for display
		// and timing, and it is NOT a list of sentences. Speaker labels are NOT on these
		// items: see SpeakerLabels.
		Items []TranscriptItem `json:"items"`
		// SpeakerLabels is present only when the job ran with ShowSpeakerLabels.
		SpeakerLabels struct {
			Segments []struct {
				SpeakerLabel string `json:"speaker_label"`
				StartTime    string `json:"start_time"`
				EndTime      string `json:"end_time"`
				Items        []struct {
					SpeakerLabel string `json:"speaker_label"`
					StartTime    string `json:"start_time"`
					EndTime      string `json:"end_time"`
				} `json:"items"`
			} `json:"segments"`
		} `json:"speaker_labels"`
	} `json:"results"`
}

// TranscriptItem is one word ("pronunciation") or punctuation mark. Punctuation has no times.
type TranscriptItem struct {
	Type         string `json:"type"` // "pronunciation" or "punctuation"
	StartTime    string `json:"start_time,omitempty"`
	EndTime      string `json:"end_time,omitempty"`
	Alternatives []struct {
		Content    string `json:"content"`
		Confidence string `json:"confidence"`
	} `json:"alternatives"`
}

// Sentence is a display segment: a run of words ending at sentence punctuation or a
// change of speaker.
type Sentence struct {
	Speaker string // empty without speaker labels
	Text    string
	Start   string // start_time of the first word, seconds, as Transcribe wrote it
}

// ParseTranscript decodes a Transcribe output file.
func ParseTranscript(raw []byte) (*TranscriptResult, error) {
	var t TranscriptResult
	if err := json.Unmarshal(raw, &t); err != nil {
		return nil, fmt.Errorf("gateway: transcript: %w", err)
	}
	return &t, nil
}

// FullText joins the entries of results.transcripts in order. Translate or summarise this,
// never the Items joined by newlines: that reads as one word per line.
func (t *TranscriptResult) FullText() string {
	parts := make([]string, 0, len(t.Results.Transcripts))
	for _, tr := range t.Results.Transcripts {
		if s := strings.TrimSpace(tr.Transcript); s != "" {
			parts = append(parts, s)
		}
	}
	return strings.Join(parts, " ")
}

// speakerByStart maps a word's start_time to its speaker, from results.speaker_labels.
func (t *TranscriptResult) speakerByStart() map[string]string {
	m := map[string]string{}
	for _, seg := range t.Results.SpeakerLabels.Segments {
		for _, it := range seg.Items {
			if it.StartTime != "" {
				m[it.StartTime] = it.SpeakerLabel
			}
		}
	}
	return m
}

// noSpaceScript reports a rune from a script that is not written with spaces between words
// (Han, Hiragana, Katakana, Thai, Lao, Khmer, Myanmar).
func noSpaceScript(r rune) bool {
	return unicode.In(r, unicode.Han, unicode.Hiragana, unicode.Katakana, unicode.Thai,
		unicode.Lao, unicode.Khmer, unicode.Myanmar)
}

// Sentences groups Items for display. Each item's first alternative is its text. A
// punctuation item attaches to the word before it with no space, and the sentence ends at
// ".", "?", "!" or their CJK forms. With speaker labels on, each word takes its speaker from
// results.speaker_labels (joined by start_time) and a change of speaker ends the sentence.
// Words are joined with a space unless either neighbour is in a script that is not
// space-delimited (Japanese, Chinese, Thai and others), which join with none.
func (t *TranscriptResult) Sentences() []Sentence {
	speakers := t.speakerByStart()
	var out []Sentence
	var cur Sentence
	var b strings.Builder
	flush := func() {
		if s := strings.TrimSpace(b.String()); s != "" {
			cur.Text = s
			out = append(out, cur)
		}
		b.Reset()
		cur = Sentence{}
	}
	for _, it := range t.Results.Items {
		if len(it.Alternatives) == 0 {
			continue
		}
		content := it.Alternatives[0].Content
		if it.Type == "punctuation" {
			b.WriteString(content)
			if strings.ContainsAny(content, ".?!。？！") {
				flush()
			}
			continue
		}
		spk := speakers[it.StartTime]
		if spk != "" && cur.Speaker != "" && spk != cur.Speaker {
			flush()
		}
		if b.Len() == 0 {
			cur.Speaker, cur.Start = spk, it.StartTime
		} else if prev, _ := lastRune(b.String()); content != "" && !noSpaceScript(prev) && !noSpaceScript([]rune(content)[0]) {
			b.WriteByte(' ')
		}
		b.WriteString(content)
	}
	flush()
	return out
}

func lastRune(s string) (rune, bool) {
	r := []rune(s)
	if len(r) == 0 {
		return 0, false
	}
	return r[len(r)-1], true
}
