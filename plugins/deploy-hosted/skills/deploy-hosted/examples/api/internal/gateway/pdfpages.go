package gateway

import (
	"bytes"
	"errors"
	"regexp"
	"strconv"
)

// ErrPagesUnknown means the PDF's page count could not be positively read from its bytes.
var ErrPagesUnknown = errors.New("gateway: cannot count PDF pages")

var (
	rootRef  = regexp.MustCompile(`/Root\s+(\d+)\s+\d+\s+R`)
	pagesRef = regexp.MustCompile(`/Pages\s+(\d+)\s+\d+\s+R`)
	countKey = regexp.MustCompile(`/Count\s+(\d+)`)
)

// PDFPageCount reads the page count of a simple PDF without a library: trailer /Root, that
// catalog's /Pages, that node's own top-level /Count. A WRONG count is worse than none, so it
// returns ErrPagesUnknown unless it can positively identify the root: a file with object
// streams (/ObjStm), cross-reference streams (/XRef), an incremental update (/Prev),
// encryption, or a root it cannot follow is unknown. Textract bills and reserves by page, so
// measure before you submit: refuse a file over the app's own page cap, tell the person the
// count, and treat ErrPagesUnknown as "use a PDF library or ask for a re-saved file", never as
// "send it and see".
func PDFPageCount(pdf []byte) (int, error) {
	if !bytes.HasPrefix(pdf, []byte("%PDF-")) {
		return 0, ErrPagesUnknown
	}
	for _, marker := range []string{"/ObjStm", "/XRef", "/Prev", "/Encrypt"} {
		if bytes.Contains(pdf, []byte(marker)) {
			return 0, ErrPagesUnknown
		}
	}
	i := bytes.LastIndex(pdf, []byte("trailer"))
	if i < 0 {
		return 0, ErrPagesUnknown
	}
	m := rootRef.FindSubmatch(pdf[i:])
	if m == nil {
		return 0, ErrPagesUnknown
	}
	catalog, ok := objectBody(pdf, string(m[1]))
	if !ok {
		return 0, ErrPagesUnknown
	}
	pm := pagesRef.FindSubmatch(topLevel(catalog))
	if pm == nil {
		return 0, ErrPagesUnknown
	}
	pages, ok := objectBody(pdf, string(pm[1]))
	if !ok {
		return 0, ErrPagesUnknown
	}
	cm := countKey.FindSubmatch(topLevel(pages))
	if cm == nil {
		return 0, ErrPagesUnknown
	}
	n, err := strconv.Atoi(string(cm[1]))
	if err != nil || n <= 0 || n > 1<<20 {
		return 0, ErrPagesUnknown
	}
	return n, nil
}

// objectBody returns the bytes between "N 0 obj" and the next "endobj" for object number num.
func objectBody(pdf []byte, num string) ([]byte, bool) {
	re := regexp.MustCompile(`(?:^|[\r\n\s])` + num + `\s+\d+\s+obj\b`)
	loc := re.FindIndex(pdf)
	if loc == nil {
		return nil, false
	}
	rest := pdf[loc[1]:]
	end := bytes.Index(rest, []byte("endobj"))
	if end < 0 {
		return nil, false
	}
	return rest[:end], true
}

// topLevel blanks everything inside nested << >> dictionaries so a key is only matched on the
// object's own dictionary (a /Resources<<...>> or a child's /Count must not be read as the node's).
func topLevel(obj []byte) []byte {
	out := make([]byte, 0, len(obj))
	depth := 0
	for i := 0; i < len(obj); i++ {
		switch {
		case i+1 < len(obj) && obj[i] == '<' && obj[i+1] == '<':
			depth++
			i++
			if depth == 1 {
				out = append(out, ' ')
			}
		case i+1 < len(obj) && obj[i] == '>' && obj[i+1] == '>':
			if depth > 0 {
				depth--
			}
			i++
			out = append(out, ' ')
		default:
			if depth <= 1 {
				out = append(out, obj[i])
			}
		}
	}
	return out
}
