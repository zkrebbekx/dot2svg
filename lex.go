package dot2svg

import (
	"fmt"
	"strings"
)

type tokenKind int

const (
	tokEOF tokenKind = iota
	tokIdent
	tokString
	tokLBrace
	tokRBrace
	tokLBracket
	tokRBracket
	tokEquals
	tokArrow  // ->
	tokEdgeOp // --
	tokComma
	tokSemi
	tokColon
)

type token struct {
	kind tokenKind
	val  string
	line int
}

// lexer tokenizes a practical subset of DOT: digraph/graph/subgraph/node/
// edge statements, quoted and bare identifiers, attr lists, and both edge
// operators. It skips //, #, and /* */ comments.
type lexer struct {
	src  []rune
	pos  int
	line int
}

func newLexer(src string) *lexer {
	return &lexer{src: []rune(src), line: 1}
}

func (l *lexer) tokens() ([]token, error) {
	var toks []token
	for {
		t, err := l.next()
		if err != nil {
			return nil, err
		}
		toks = append(toks, t)
		if t.kind == tokEOF {
			return toks, nil
		}
	}
}

func (l *lexer) at(off int) rune {
	if l.pos+off >= len(l.src) {
		return 0
	}
	return l.src[l.pos+off]
}

func (l *lexer) next() (token, error) {
	l.skipSpaceAndComments()
	if l.pos >= len(l.src) {
		return token{kind: tokEOF, line: l.line}, nil
	}

	line := l.line
	c := l.src[l.pos]

	switch c {
	case '{':
		l.pos++
		return token{kind: tokLBrace, val: "{", line: line}, nil
	case '}':
		l.pos++
		return token{kind: tokRBrace, val: "}", line: line}, nil
	case '[':
		l.pos++
		return token{kind: tokLBracket, val: "[", line: line}, nil
	case ']':
		l.pos++
		return token{kind: tokRBracket, val: "]", line: line}, nil
	case '=':
		l.pos++
		return token{kind: tokEquals, val: "=", line: line}, nil
	case ',':
		l.pos++
		return token{kind: tokComma, val: ",", line: line}, nil
	case ';':
		l.pos++
		return token{kind: tokSemi, val: ";", line: line}, nil
	case ':':
		l.pos++
		return token{kind: tokColon, val: ":", line: line}, nil
	case '-':
		if l.at(1) == '>' {
			l.pos += 2
			return token{kind: tokArrow, val: "->", line: line}, nil
		}
		if l.at(1) == '-' {
			l.pos += 2
			return token{kind: tokEdgeOp, val: "--", line: line}, nil
		}
		return token{}, fmt.Errorf("dot2svg: line %d: unexpected '-'", line)
	case '"':
		return l.lexString()
	default:
		if isIdentStart(c) {
			return l.lexIdent(), nil
		}
		return token{}, fmt.Errorf("dot2svg: line %d: unexpected character %q", line, c)
	}
}

func (l *lexer) skipSpaceAndComments() {
	for l.pos < len(l.src) {
		c := l.src[l.pos]
		switch {
		case c == '\n':
			l.line++
			l.pos++
		case c == ' ' || c == '\t' || c == '\r':
			l.pos++
		case c == '/' && l.at(1) == '/':
			for l.pos < len(l.src) && l.src[l.pos] != '\n' {
				l.pos++
			}
		case c == '#':
			for l.pos < len(l.src) && l.src[l.pos] != '\n' {
				l.pos++
			}
		case c == '/' && l.at(1) == '*':
			l.pos += 2
			for l.pos < len(l.src) && (l.src[l.pos] != '*' || l.at(1) != '/') {
				if l.src[l.pos] == '\n' {
					l.line++
				}
				l.pos++
			}
			l.pos += 2
		default:
			return
		}
	}
}

// lexString reads a quoted DOT string, unescaping \" and \\, and leaving
// other backslash sequences (\n, \l, \r used by Graphviz for label
// line-breaks) intact for the caller to interpret.
func (l *lexer) lexString() (token, error) {
	line := l.line
	l.pos++ // opening quote
	var b strings.Builder
	for {
		if l.pos >= len(l.src) {
			return token{}, fmt.Errorf("dot2svg: line %d: unterminated string", line)
		}
		c := l.src[l.pos]
		if c == '"' {
			l.pos++
			return token{kind: tokString, val: b.String(), line: line}, nil
		}
		if c == '\\' && l.pos+1 < len(l.src) {
			nc := l.src[l.pos+1]
			switch nc {
			case '"':
				b.WriteRune('"')
				l.pos += 2
				continue
			case '\\':
				b.WriteRune('\\')
				l.pos += 2
				continue
			default:
				// Keep e.g. \n, \l literally; the renderer decides how to
				// display them.
				b.WriteRune(c)
				b.WriteRune(nc)
				l.pos += 2
				continue
			}
		}
		if c == '\n' {
			l.line++
		}
		b.WriteRune(c)
		l.pos++
	}
}

func (l *lexer) lexIdent() token {
	line := l.line
	start := l.pos
	for l.pos < len(l.src) && isIdentPart(l.src[l.pos]) {
		l.pos++
	}
	return token{kind: tokIdent, val: string(l.src[start:l.pos]), line: line}
}

func isIdentStart(r rune) bool {
	return r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.'
}

func isIdentPart(r rune) bool {
	return isIdentStart(r)
}
