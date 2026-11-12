package ownedscope

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// parseYAML reads the YAML (or JSON) subset an OpenAPI contract is written in into maps, lists and
// scalars: block and flow mappings and sequences, plain, single- and double-quoted scalars (also
// over several lines), block scalars (| and >, read as empty text: nothing here needs their words),
// comments and one leading "---". The module is standard library only, so this is not a general
// parser, and it FAILS CLOSED: anchors, aliases, tags, merge keys, complex keys, directives,
// several documents, tab indentation, a duplicate key and any line it cannot place are an error
// naming the line, never a smaller document. Write the contract in plain block or flow style.
func parseYAML(doc string) (any, error) {
	var lines []yline
	for n, raw := range strings.Split(strings.ReplaceAll(doc, "\r\n", "\n"), "\n") {
		trimmed := strings.TrimLeft(raw, " ")
		if strings.TrimSpace(raw) == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(trimmed, "\t") {
			return nil, fmt.Errorf("line %d: indented with a tab, which YAML does not allow", n+1)
		}
		text := strings.TrimSpace(stripComment(trimmed))
		indent := len(raw) - len(trimmed)
		if indent == 0 && (text == "---" || text == "...") {
			if len(lines) == 0 && text == "---" {
				continue
			}
			return nil, fmt.Errorf("line %d: more than one YAML document, or a document end marker, is not read", n+1)
		}
		if indent == 0 && strings.HasPrefix(text, "%") {
			return nil, fmt.Errorf("line %d: a YAML directive is not read", n+1)
		}
		lines = append(lines, yline{no: n + 1, indent: indent, text: text})
	}
	if len(lines) == 0 {
		return nil, fmt.Errorf("the document is empty")
	}
	if t := lines[0].text; t[0] == '{' || t[0] == '[' {
		// A whole document in flow style, JSON included.
		var all []string
		for _, l := range lines {
			all = append(all, l.text)
		}
		f := &flow{s: strings.Join(all, " ")}
		v, err := f.value()
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", lines[0].no, err)
		}
		if f.skip(); f.i != len(f.s) {
			return nil, fmt.Errorf("unexpected text after the document: %q", f.s[f.i:])
		}
		return v, nil
	}
	p := &yparser{lines: lines}
	v, err := p.block()
	if err != nil {
		return nil, err
	}
	if l := p.peek(); l != nil {
		return nil, fmt.Errorf("line %d: cannot place %q (wrong indentation?)", l.no, l.text)
	}
	return v, nil
}

type yline struct {
	no     int
	indent int
	text   string
}

type yparser struct {
	lines []yline
	i     int
}

func (p *yparser) peek() *yline {
	if p.i < len(p.lines) {
		return &p.lines[p.i]
	}
	return nil
}

var blockIndicator = regexp.MustCompile(`^[|>][-+0-9]*$`)

func isItem(text string) bool { return text == "-" || strings.HasPrefix(text, "- ") }

// stripComment removes a trailing " # comment" that is not inside a quoted scalar.
func stripComment(s string) string {
	var quote byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case quote != 0:
			if c == '\\' && quote == '"' {
				i++
			} else if c == quote {
				quote = 0
			}
		case (c == '"' || c == '\'') && (i == 0 || strings.IndexByte(" \t[{,:", s[i-1]) >= 0):
			quote = c
		case c == '#' && (i == 0 || s[i-1] == ' ' || s[i-1] == '\t'):
			return s[:i]
		}
	}
	return s
}

// block reads the node that starts at the current line.
func (p *yparser) block() (any, error) {
	l := p.peek()
	if l == nil {
		return nil, nil
	}
	if isItem(l.text) {
		return p.list(l.indent)
	}
	return p.mapping(l.indent)
}

func (p *yparser) mapping(ind int) (any, error) {
	m := map[string]any{}
	for {
		l := p.peek()
		if l == nil || l.indent < ind {
			return m, nil
		}
		if l.indent > ind {
			return nil, fmt.Errorf("line %d: unexpected indentation before %q", l.no, l.text)
		}
		key, rest, ok, err := splitKey(l.text)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", l.no, err)
		}
		if !ok {
			return nil, fmt.Errorf("line %d: cannot read %q as a key and a value", l.no, l.text)
		}
		if key == "<<" {
			return nil, fmt.Errorf("line %d: a merge key (<<) is not read", l.no)
		}
		if _, dup := m[key]; dup {
			return nil, fmt.Errorf("line %d: the key %q is given twice", l.no, key)
		}
		p.i++
		v, err := p.value(rest, ind, l.no, true)
		if err != nil {
			return nil, err
		}
		m[key] = v
	}
}

func (p *yparser) list(ind int) (any, error) {
	out := []any{}
	for {
		l := p.peek()
		if l == nil || l.indent < ind {
			return out, nil
		}
		if l.indent > ind {
			return nil, fmt.Errorf("line %d: unexpected indentation before %q", l.no, l.text)
		}
		if !isItem(l.text) {
			return out, nil
		}
		rest := strings.TrimLeft(l.text[1:], " ")
		if rest != "" {
			_, _, isKey, err := splitKey(rest)
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", l.no, err)
			}
			if isKey || isItem(rest) {
				// "- key: v" and "- - x": the item's own block starts where its text does.
				l.indent += len(l.text) - len(rest)
				l.text = rest
				v, err := p.block()
				if err != nil {
					return nil, err
				}
				out = append(out, v)
				continue
			}
		}
		p.i++
		v, err := p.value(rest, ind, l.no, false)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
}

// value reads what follows a "key:" (inMap) or a "- " on a line at indent parent, whose own text
// after it is rest.
func (p *yparser) value(rest string, parent, no int, inMap bool) (any, error) {
	switch {
	case rest == "":
		next := p.peek()
		switch {
		case next == nil:
			return nil, nil
		case next.indent > parent:
			return p.block()
		case inMap && next.indent == parent && isItem(next.text):
			return p.list(parent)
		}
		return nil, nil
	case blockIndicator.MatchString(rest):
		for l := p.peek(); l != nil && l.indent > parent; l = p.peek() {
			p.i++
		}
		return "", nil
	case strings.IndexByte("&*!@`%", rest[0]) >= 0:
		return nil, fmt.Errorf("line %d: an anchor, alias, tag or reserved indicator (%q) is not read", no, rest[:1])
	case rest[0] == '[' || rest[0] == '{':
		text := rest
		for !flowClosed(text) {
			l := p.peek()
			if l == nil {
				return nil, fmt.Errorf("line %d: a flow collection is never closed", no)
			}
			p.i++
			text += " " + l.text
		}
		f := &flow{s: text}
		v, err := f.value()
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", no, err)
		}
		if f.skip(); f.i != len(f.s) {
			return nil, fmt.Errorf("line %d: unexpected text after a flow collection: %q", no, f.s[f.i:])
		}
		return v, nil
	case rest[0] == '"' || rest[0] == '\'':
		text := rest
		for !quoteClosed(text) {
			l := p.peek()
			if l == nil {
				return nil, fmt.Errorf("line %d: a quoted scalar is never closed", no)
			}
			p.i++
			text += " " + l.text
		}
		v, err := scalar(text)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", no, err)
		}
		return v, nil
	}
	text := rest
	for l := p.peek(); l != nil && l.indent > parent; l = p.peek() {
		p.i++
		text += " " + l.text
	}
	v, err := scalar(text)
	if err != nil {
		return nil, fmt.Errorf("line %d: %w", no, err)
	}
	return v, nil
}

// splitKey splits "key: value" (the key plain or quoted). ok is false when text is no such line.
func splitKey(text string) (key, rest string, ok bool, err error) {
	if text != "" && strings.IndexByte("&*!", text[0]) >= 0 {
		// An anchor, alias or tag on a key is not read; failing closed beats a key that reads wrong.
		return "", "", false, fmt.Errorf("an anchor, alias or tag (%q) on a key is not read", text[:1])
	}
	if text == "" || text[0] == '[' || text[0] == '{' || text[0] == '?' && (len(text) == 1 || text[1] == ' ') {
		return "", "", false, nil
	}
	if q := text[0]; q == '"' || q == '\'' {
		end := closingQuote(text)
		if end < 0 {
			return "", "", false, nil
		}
		after := strings.TrimLeft(text[end+1:], " ")
		if !strings.HasPrefix(after, ":") || (len(after) > 1 && after[1] != ' ') {
			return "", "", false, nil
		}
		k, err := scalarString(text[:end+1])
		if err != nil {
			return "", "", false, err
		}
		return k, strings.TrimSpace(after[1:]), true, nil
	}
	idx := strings.Index(text, ": ")
	if idx < 0 && strings.HasSuffix(text, ":") {
		idx = len(text) - 1
	}
	if idx <= 0 {
		return "", "", false, nil
	}
	return strings.TrimSpace(text[:idx]), strings.TrimSpace(text[idx+1:]), true, nil
}

// closingQuote is the index of the quote that closes the quoted scalar starting text, or -1.
func closingQuote(text string) int {
	q := text[0]
	for i := 1; i < len(text); i++ {
		switch {
		case q == '"' && text[i] == '\\':
			i++
		case text[i] == q && q == '\'' && i+1 < len(text) && text[i+1] == '\'':
			i++
		case text[i] == q:
			return i
		}
	}
	return -1
}

func quoteClosed(text string) bool { return closingQuote(text) >= 0 }

// flowClosed reports whether every bracket of a flow collection that starts text is closed.
func flowClosed(text string) bool {
	depth := 0
	var quote byte
	for i := 0; i < len(text); i++ {
		c := text[i]
		switch {
		case quote != 0:
			if c == '\\' && quote == '"' {
				i++
			} else if c == quote {
				quote = 0
			}
		case c == '"' || c == '\'':
			quote = c
		case c == '[' || c == '{':
			depth++
		case c == ']' || c == '}':
			depth--
		}
	}
	return depth <= 0 && quote == 0
}

func scalarString(text string) (string, error) {
	v, err := scalar(text)
	if err != nil {
		return "", err
	}
	if s, ok := v.(string); ok {
		return s, nil
	}
	return text, nil
}

// scalar reads one scalar: a quoted string, null, a boolean, a number or a plain string.
func scalar(text string) (any, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, nil
	}
	if end := closingQuote2(text); end > 0 {
		inner := text[1:end]
		if text[0] == '\'' {
			return strings.ReplaceAll(inner, "''", "'"), nil
		}
		s, err := strconv.Unquote(text[:end+1])
		if err != nil {
			// A YAML escape Go does not share (\/, say) must not read as raw text: fail closed.
			return nil, fmt.Errorf("the quoted scalar %s has an escape that is not read", text[:end+1])
		}
		return s, nil
	}
	switch text {
	case "~", "null", "Null", "NULL":
		return nil, nil
	case "true", "True", "TRUE":
		return true, nil
	case "false", "False", "FALSE":
		return false, nil
	}
	if _, err := strconv.ParseFloat(text, 64); err == nil && !strings.ContainsAny(text, "_xXnNiI") {
		f, _ := strconv.ParseFloat(text, 64)
		return f, nil
	}
	return text, nil
}

func closingQuote2(text string) int {
	if text[0] != '"' && text[0] != '\'' {
		return -1
	}
	return closingQuote(text)
}

// flow reads a flow collection ({a: b, c: [d, e]}) held in one string.
type flow struct {
	s string
	i int
}

func (f *flow) skip() {
	for f.i < len(f.s) && (f.s[f.i] == ' ' || f.s[f.i] == '\t') {
		f.i++
	}
}

func (f *flow) value() (any, error) {
	f.skip()
	if f.i >= len(f.s) {
		return nil, fmt.Errorf("a flow collection ends early")
	}
	switch f.s[f.i] {
	case '{':
		f.i++
		m := map[string]any{}
		for {
			f.skip()
			if f.i >= len(f.s) {
				return nil, fmt.Errorf("a flow mapping is never closed")
			}
			if f.s[f.i] == '}' {
				f.i++
				return m, nil
			}
			key, err := f.key()
			if err != nil {
				return nil, err
			}
			if _, dup := m[key]; dup {
				return nil, fmt.Errorf("the key %q is given twice", key)
			}
			f.skip()
			var v any
			if f.i < len(f.s) && f.s[f.i] == ':' {
				f.i++
				if v, err = f.value(); err != nil {
					return nil, err
				}
			}
			m[key] = v
			f.skip()
			if f.i < len(f.s) && f.s[f.i] == ',' {
				f.i++
			}
		}
	case '[':
		f.i++
		out := []any{}
		for {
			f.skip()
			if f.i >= len(f.s) {
				return nil, fmt.Errorf("a flow sequence is never closed")
			}
			if f.s[f.i] == ']' {
				f.i++
				return out, nil
			}
			v, err := f.value()
			if err != nil {
				return nil, err
			}
			out = append(out, v)
			f.skip()
			if f.i < len(f.s) && f.s[f.i] == ',' {
				f.i++
			}
		}
	case '&', '*', '!':
		return nil, fmt.Errorf("an anchor, alias or tag (%q) is not read", f.s[f.i:f.i+1])
	case '"', '\'':
		end := closingQuote(f.s[f.i:])
		if end < 0 {
			return nil, fmt.Errorf("a quoted scalar is never closed")
		}
		v, err := scalar(f.s[f.i : f.i+end+1])
		if err != nil {
			return nil, err
		}
		f.i += end + 1
		return v, nil
	}
	start := f.i
	for f.i < len(f.s) && strings.IndexByte(",]}", f.s[f.i]) < 0 {
		f.i++
	}
	return scalar(f.s[start:f.i])
}

// key reads a flow mapping's key: quoted, or plain up to its colon.
func (f *flow) key() (string, error) {
	f.skip()
	if f.i < len(f.s) && (f.s[f.i] == '"' || f.s[f.i] == '\'') {
		end := closingQuote(f.s[f.i:])
		if end < 0 {
			return "", fmt.Errorf("a quoted key is never closed")
		}
		k, err := scalarString(f.s[f.i : f.i+end+1])
		if err != nil {
			return "", err
		}
		f.i += end + 1
		return k, nil
	}
	start := f.i
	for f.i < len(f.s) {
		c := f.s[f.i]
		if c == ',' || c == '}' || c == ']' || (c == ':' && (f.i+1 >= len(f.s) || strings.IndexByte(" ,}]", f.s[f.i+1]) >= 0)) {
			break
		}
		f.i++
	}
	k := strings.TrimSpace(f.s[start:f.i])
	if k != "" && strings.IndexByte("&*!", k[0]) >= 0 {
		return "", fmt.Errorf("an anchor, alias or tag (%q) on a key is not read", k[:1])
	}
	if k == "" {
		return "", fmt.Errorf("a flow mapping has an empty key")
	}
	return k, nil
}
