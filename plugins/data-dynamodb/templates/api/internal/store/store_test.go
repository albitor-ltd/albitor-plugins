package store

import (
	"errors"
	"strings"
	"testing"
)

func TestKey(t *testing.T) {
	got, err := Key("OWNER", "alice", "ITEM")
	if err != nil || got != "OWNER#alice#ITEM" {
		t.Fatalf("Key = %q, %v", got, err)
	}
	for name, parts := range map[string][]string{
		"no parts":         {},
		"empty part":       {"OWNER", "", "ITEM"},
		"separator":        {"OWNER", "a#b"},
		"control":          {"OWNER", "a\nb"},
		"nul":              {"OWNER", "a\x00"},
		"del":              {"OWNER", "a\x7f"},
		"invalid utf-8":    {"OWNER", "\xff"},
		"part too long":    {"OWNER", strings.Repeat("a", MaxKeyBytes+1)},
		"joined too long":  {strings.Repeat("a", MaxKeyBytes), "b"},
		"owner climbs out": {"OWNER", "bob#ITEM"},
	} {
		if _, err := Key(parts...); !errors.Is(err, ErrInvalid) {
			t.Errorf("%s: Key = %v, want ErrInvalid", name, err)
		}
	}
	// Non-ASCII text is a legitimate key.
	if _, err := Key("OWNER", "zoë"); err != nil {
		t.Errorf("Key refused valid UTF-8: %v", err)
	}
}

func TestCursorRoundTripAndBinding(t *testing.T) {
	q := Query{PK: "p#1", SKPrefix: "inv#", Limit: 5}
	q.Cursor = encodeCursor(q.PK, "inv#007")
	if sk, err := decodeCursor(q); err != nil || sk != "inv#007" {
		t.Fatalf("round trip = %q, %v", sk, err)
	}
	// A partition key that contains the length separator and digits cannot be
	// split differently on the way back.
	tricky := Query{PK: "3:abc", Limit: 5}
	tricky.Cursor = encodeCursor(tricky.PK, "x")
	if sk, err := decodeCursor(tricky); err != nil || sk != "x" {
		t.Fatalf("tricky partition = %q, %v", sk, err)
	}
	other := q
	other.PK = "p#2"
	if _, err := decodeCursor(other); !errors.Is(err, ErrInvalidCursor) {
		t.Errorf("cursor for another partition = %v, want ErrInvalidCursor", err)
	}
	other = q
	other.SKPrefix = "ord#"
	if _, err := decodeCursor(other); !errors.Is(err, ErrInvalidCursor) {
		t.Errorf("cursor outside the prefix = %v, want ErrInvalidCursor", err)
	}
	if sk, err := decodeCursor(Query{PK: "p"}); err != nil || sk != "" {
		t.Errorf("empty cursor = %q, %v, want the start", sk, err)
	}
}

func TestCheckData(t *testing.T) {
	for data, ok := range map[string]bool{`{}`: true, `[1]`: true, `"s"`: true, `null`: true, ``: false, `{`: false, `{} {}`: false, "{\"a\":\"\xff\"}": false, `{"a":"é"}`: true} {
		if err := checkData([]byte(data)); (err == nil) != ok {
			t.Errorf("checkData(%q) = %v, want ok=%v", data, err, ok)
		}
	}
}
