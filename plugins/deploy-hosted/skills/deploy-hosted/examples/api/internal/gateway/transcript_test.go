package gateway

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// Shaped like AWS's documented output with ShowSpeakerLabels: speakers live in
// results.speaker_labels.segments[].items[] (joined to results.items by start_time), and
// punctuation items carry no times.
const transcriptJSON = `{"jobName":"j","accountId":"1","status":"COMPLETED","results":{
 "transcripts":[{"transcript":"Hello there. How are you? Fine, ok"}],
 "speaker_labels":{"speakers":2,"segments":[
  {"start_time":"0.1","speaker_label":"spk_0","end_time":"1.5","items":[
    {"start_time":"0.1","speaker_label":"spk_0","end_time":"0.5"},
    {"start_time":"0.6","speaker_label":"spk_0","end_time":"1.0"}]},
  {"start_time":"2.0","speaker_label":"spk_1","end_time":"3.5","items":[
    {"start_time":"2.0","speaker_label":"spk_1","end_time":"2.2"},
    {"start_time":"2.3","speaker_label":"spk_1","end_time":"2.5"},
    {"start_time":"2.6","speaker_label":"spk_1","end_time":"3.0"}]},
  {"start_time":"4.0","speaker_label":"spk_0","end_time":"4.5","items":[
    {"start_time":"4.0","speaker_label":"spk_0","end_time":"4.4"}]},
  {"start_time":"5.0","speaker_label":"spk_1","end_time":"5.3","items":[
    {"start_time":"5.0","speaker_label":"spk_1","end_time":"5.2"}]}]},
 "items":[
  {"start_time":"0.1","end_time":"0.5","type":"pronunciation","alternatives":[{"confidence":"0.99","content":"Hello"},{"confidence":"0.1","content":"Hallo"}]},
  {"start_time":"0.6","end_time":"1.0","type":"pronunciation","alternatives":[{"confidence":"0.9","content":"there"}]},
  {"type":"punctuation","alternatives":[{"confidence":"0.0","content":"."}]},
  {"start_time":"2.0","end_time":"2.2","type":"pronunciation","alternatives":[{"confidence":"0.9","content":"How"}]},
  {"start_time":"2.3","end_time":"2.5","type":"pronunciation","alternatives":[{"confidence":"0.9","content":"are"}]},
  {"start_time":"2.6","end_time":"3.0","type":"pronunciation","alternatives":[{"confidence":"0.9","content":"you"}]},
  {"type":"punctuation","alternatives":[{"confidence":"0.0","content":"?"}]},
  {"start_time":"4.0","end_time":"4.4","type":"pronunciation","alternatives":[{"confidence":"0.9","content":"Fine"}]},
  {"type":"punctuation","alternatives":[{"confidence":"0.0","content":","}]},
  {"start_time":"5.0","end_time":"5.2","type":"pronunciation","alternatives":[{"confidence":"0.9","content":"ok"}]}
 ]}}`

func TestTranscriptFullText(t *testing.T) {
	tr, err := ParseTranscript([]byte(transcriptJSON))
	if err != nil {
		t.Fatal(err)
	}
	if got := tr.FullText(); got != "Hello there. How are you? Fine, ok" {
		t.Fatalf("FullText = %q", got)
	}
}

func TestTranscriptSentencesUseSpeakerSegments(t *testing.T) {
	tr, _ := ParseTranscript([]byte(transcriptJSON))
	got := tr.Sentences()
	want := []Sentence{
		{"spk_0", "Hello there.", "0.1"},
		{"spk_1", "How are you?", "2.0"},
		{"spk_0", "Fine,", "4.0"}, // speaker change ends it without a full stop
		{"spk_1", "ok", "5.0"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d sentences %+v", len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("sentence %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestSentencesWithoutSpeakersGroupByPunctuation(t *testing.T) {
	tr, _ := ParseTranscript([]byte(`{"results":{"items":[
	 {"type":"pronunciation","alternatives":[{"content":"a"}]},{"type":"pronunciation","alternatives":[{"content":"b"}]},
	 {"type":"punctuation","alternatives":[{"content":"."}]},
	 {"type":"pronunciation","alternatives":[{"content":"c"}]}]}}`))
	got := tr.Sentences()
	if len(got) != 2 || got[0].Text != "a b." || got[1].Text != "c" {
		t.Fatalf("got %+v", got)
	}
}

func TestSentencesJoinWithoutSpacesForUnspacedScripts(t *testing.T) {
	word := func(s string) string {
		return `{"type":"pronunciation","alternatives":[{"content":"` + s + `"}]}`
	}
	tr, _ := ParseTranscript([]byte(`{"results":{"items":[` + word("今日") + `,` + word("は") + `,` + word("晴れ") +
		`,{"type":"punctuation","alternatives":[{"content":"。"}]},` + word("Hi") + `,` + word("there") + `]}}`))
	got := tr.Sentences()
	if len(got) != 2 || got[0].Text != "今日は晴れ。" || got[1].Text != "Hi there" {
		t.Fatalf("got %+v", got)
	}
}

func TestChunkByBytesRespectsByteLimitAndRoundTrips(t *testing.T) {
	for name, text := range map[string]string{
		"ascii":     strings.Repeat("Hello world. ", 200),
		"cjk":       strings.Repeat("今日は天気がいいです。", 300),
		"no breaks": strings.Repeat("日", 1000),
		"newlines":  strings.Repeat("line one\n", 400),
	} {
		t.Run(name, func(t *testing.T) {
			chunks := ChunkByBytes(text, 500)
			if strings.Join(chunks, "") != text {
				t.Fatal("chunks do not rejoin to the input")
			}
			for i, c := range chunks {
				if len(c) > 500 {
					t.Errorf("chunk %d is %d bytes", i, len(c))
				}
				if !utf8.ValidString(c) {
					t.Errorf("chunk %d splits a rune", i)
				}
			}
		})
	}
}

func TestChunkByBytesSplitsAtSentenceEnd(t *testing.T) {
	text := "One two three. Four five six. Seven eight nine."
	chunks := ChunkByBytes(text, 35)
	if len(chunks) != 2 || chunks[0] != "One two three. Four five six. " || chunks[1] != "Seven eight nine." {
		t.Fatalf("chunks %q", chunks)
	}
}

func TestChunkByBytesDoesNotSplitAtADecimalPoint(t *testing.T) {
	// Sentence punctuation not followed by whitespace is not a boundary; whitespace is used.
	chunks := ChunkByBytes("aaaa 3.14159 bbbb", 10)
	for _, c := range chunks {
		if strings.HasSuffix(c, "3.") {
			t.Fatalf("split inside a number: %q", chunks)
		}
	}
	if strings.Join(chunks, "") != "aaaa 3.14159 bbbb" {
		t.Fatalf("chunks %q", chunks)
	}
}

func TestChunkByBytesTerminatesOnInvalidUTF8(t *testing.T) {
	bad := strings.Repeat("\x80", 5000) + "ok" + strings.Repeat("\xbf\xbf", 3000)
	chunks := ChunkByBytes(bad, 100)
	if len(chunks) == 0 {
		t.Fatal("no chunks")
	}
	for _, c := range chunks {
		if len(c) > 100 || !utf8.ValidString(c) {
			t.Fatalf("bad chunk of %d bytes", len(c))
		}
	}
}

func TestChunkCharactersAreNotBytes(t *testing.T) {
	text := strings.Repeat("あ", 4000) // 4,000 characters, 12,000 bytes
	if len(text) <= 10000 {
		t.Fatal("fixture must exceed the byte limit")
	}
	for _, c := range ChunkByBytes(text, TranslateChunkBytes) {
		if len(c) > 10000 {
			t.Fatalf("chunk of %d bytes", len(c))
		}
	}
}

func TestPDFPageCount(t *testing.T) {
	simple := "%PDF-1.4\n1 0 obj\n<</Type /Catalog /Pages 2 0 R>>\nendobj\n" +
		"2 0 obj\n<</Type /Pages /Resources <</Font <</F1 9 0 R>> /Count 99>> /Kids [3 0 R 4 0 R] /Count 2>>\nendobj\n" +
		"3 0 obj\n<</Type /Page /Parent 2 0 R>>\nendobj\n4 0 obj\n<</Type /Page /Parent 2 0 R>>\nendobj\n" +
		"xref\n0 5\ntrailer\n<</Root 1 0 R /Size 5>>\nstartxref\n0\n%%EOF"
	if n, err := PDFPageCount([]byte(simple)); err != nil || n != 2 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	// A nested tree: the root /Count is the total, not a child's.
	nested := "%PDF-1.4\n1 0 obj<</Pages 2 0 R /Type /Catalog>>endobj\n" +
		"2 0 obj<</Type /Pages /Kids [3 0 R] /Count 5>>endobj\n3 0 obj<</Type /Pages /Parent 2 0 R /Kids [] /Count 3>>endobj\n" +
		"trailer\n<</Root 1 0 R>>\n%%EOF"
	if n, err := PDFPageCount([]byte(nested)); err != nil || n != 5 {
		t.Fatalf("nested n=%d err=%v", n, err)
	}
	for name, bad := range map[string]string{
		"not a pdf":          "hello",
		"encrypted":          simple + "/Encrypt 5 0 R",
		"incremental update": simple + "\ntrailer\n<</Root 1 0 R /Prev 100>>",
		"object stream":      simple + "\n7 0 obj<</Type /ObjStm /N 3>>endobj",
		"xref stream":        simple + "\n8 0 obj<</Type /XRef>>endobj",
		"no trailer":         "%PDF-1.4\n1 0 obj<</Type /Pages /Count 3>>endobj",
		"root not found":     "%PDF-1.4\ntrailer\n<</Root 9 0 R>>",
	} {
		if n, err := PDFPageCount([]byte(bad)); err != ErrPagesUnknown {
			t.Errorf("%s: n=%d err=%v", name, n, err)
		}
	}
}
