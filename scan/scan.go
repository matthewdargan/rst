// Copyright 2023 Matthew P. Dargan. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package scan lexically analyzes reStructuredText.
package scan

import (
	"fmt"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Token represents a token or text string returned from the scanner.
type Token struct {
	Type Type   // The type of this item.
	Line int    // The line number on which this token appears
	Text string // The text of this item.
}

//go:generate stringer -type Type

// Type identifies the type of lex items.
type Type int

const (
	EOF                  Type = iota // EOF indicates an end-of-file character
	Error                            // Error occurred; value is text of error
	BlankLine                        // BlankLine separates elements
	Space                            // Space indicates a run of whitespace
	Text                             // Text indicates plaintext
	Comment                          // Comment marker
	HyperlinkStart                   // HyperlinkStart starts a hyperlink target
	HyperlinkPrefix                  // HyperlinkPrefix prefixes a hyperlink target name
	HyperlinkQuote                   // HyperlinkQuote encloses a hyperlink target name that contains any colons
	HyperlinkName                    // HyperlinkName indicates a hyperlink target name
	HyperlinkSuffix                  // HyperlinkSuffix suffixes hyperlink target name
	HyperlinkURI                     // HyperlinkURI points to a hyperlink target
	InlineReferenceOpen              // InlineReferenceOpen opens an inline reference
	InlineReferenceText              // InlineReferenceText indicates text referenced inline
	InlineReferenceClose             // InlineReferenceClose closes an inline reference
)

func (i Token) String() string {
	switch {
	case i.Type == EOF:
		return "EOF"
	case i.Type == Error:
		return "error: " + i.Text
	case len(i.Text) > 10:
		return fmt.Sprintf("%s: %.10q...", i.Type, i.Text)
	}
	return fmt.Sprintf("%s: %q", i.Type, i.Text)
}

const eof = -1

// stateFn represents the state of the scanner as a function that returns the next state.
type stateFn func(*Scanner) stateFn

// Scanner holds the state of the scanner.
type Scanner struct {
	r         io.ByteReader // reads input bytes
	done      bool          // are we done scanning?
	name      string        // name of the input; used only for error reports
	buf       []byte        // I/O buffer, re-used
	input     string        // line of text being scanned
	lastRune  rune          // most recent return from next()
	lastWidth int           // size of that rune
	line      int           // line number in input
	pos       int           // current position in the input
	start     int           // start position of this item
	token     Token         // token to return to parser
	types     [2]Type       // most recent scanned types
}

// loadLine reads the next line of input and stores it in (appends it to) the input.
// (l.input may have data left over when we are called.)
// It strips carriage returns to make subsequent processing simpler.
func (l *Scanner) loadLine() {
	l.buf = l.buf[:0]
	for {
		c, err := l.r.ReadByte()
		if err != nil {
			l.done = true
			break
		}
		if c != '\r' { // There will never be a \r in l.input.
			l.buf = append(l.buf, c)
		}
		if c == '\n' {
			break
		}
	}
	// Reset to beginning of input buffer if there is nothing pending.
	if l.start == l.pos {
		l.input = string(l.buf)
		l.start = 0
		l.pos = 0
	} else {
		l.input += string(l.buf)
	}
}

// readRune reads the next rune from the input.
func (l *Scanner) readRune() (rune, int) {
	if !l.done && l.pos == len(l.input) {
		l.loadLine()
	}
	if len(l.input) == l.pos {
		return eof, 0
	}
	return utf8.DecodeRuneInString(l.input[l.pos:])
}

// next returns the next rune in the input.
func (l *Scanner) next() rune {
	l.lastRune, l.lastWidth = l.readRune()
	l.pos += l.lastWidth
	return l.lastRune
}

// peek returns but does not consume the next rune in the input.
func (l *Scanner) peek() rune {
	r, _ := l.readRune()
	return r
}

// emit passes an item back to the client.
func (l *Scanner) emit(t Type) stateFn {
	if t == BlankLine {
		l.line++
	}
	text := l.input[l.start:l.pos]
	l.token = Token{t, l.line, text}
	l.types[0] = l.types[1]
	l.types[1] = t
	l.start = l.pos
	return nil
}

// ignore skips over the pending input before this point.
// It tracks newlines in the ignored text, so use it only
// for text that is skipped without calling l.next.
func (l *Scanner) ignore() {
	l.line += strings.Count(l.input[l.start:l.pos], "\n")
	l.start = l.pos
}

// errorf returns an error token and empties the input.
func (l *Scanner) errorf(format string, args ...interface{}) stateFn {
	l.token = Token{Error, l.start, fmt.Sprintf(format, args...)}
	l.start = 0
	l.pos = 0
	l.input = l.input[:0]
	return nil
}

// New creates and returns a new scanner.
func New(name string, r io.ByteReader) *Scanner {
	l := &Scanner{
		r:    r,
		name: name,
		line: 1,
	}
	return l
}

// Next returns the next token.
func (l *Scanner) Next() Token {
	l.lastRune = eof
	l.lastWidth = 0
	l.token = Token{EOF, l.pos, "EOF"}
	state := lexAny
	for {
		state = state(l)
		if state == nil {
			return l.token
		}
	}
}

const (
	hyperlinkStart      = ".. _"
	anonHyperlinkStart  = "__ "
	anonHyperlinkPrefix = "__:"
)

// lexAny scans non-space items.
func lexAny(l *Scanner) stateFn {
	switch r := l.next(); {
	case r == eof:
		return nil
	case r == '\n':
		return l.emit(BlankLine)
	case unicode.IsSpace(r):
		return lexSpace
	case l.isComment(r):
		return lexComment
	case l.isHyperlinkStart(r):
		return lexHyperlinkStart
	case l.input[:l.pos] == hyperlinkStart:
		return lexHyperlinkPrefix
	case r == '`':
		return lexQuote
	case l.isHyperlinkName():
		return lexHyperlinkName
	case l.isHyperlinkSuffix(r):
		return lexEndOfLine(l, HyperlinkSuffix)
	case l.isHyperlinkURI():
		return lexUntilTerminator(l, HyperlinkURI)
	case l.isInlineReferenceText():
		return lexInlineReferenceText
	case l.isInlineReferenceClose():
		return lexInlineReferenceClose
	default:
		return lexUntilTerminator(l, Text)
	}
}

// lexEndOfLine scans a lex item and ignores an end-of-line character if present.
func lexEndOfLine(l *Scanner, typ Type) stateFn {
	i := l.emit(typ)
	if l.peek() == '\n' {
		l.pos++
		l.ignore()
	}
	return i
}

// lexUntilTerminator scans a lex item until a newline or end-of-line character.
func lexUntilTerminator(l *Scanner, typ Type) stateFn {
	for {
		switch l.peek() {
		case eof:
			return l.emit(typ)
		case '\n':
			return lexEndOfLine(l, typ)
		default:
			l.next()
		}
	}
}

// lexSpace scans a run of space characters.
func lexSpace(l *Scanner) stateFn {
	for unicode.IsSpace(l.peek()) {
		l.next()
	}
	return l.emit(Space)
}

// lexComment scans a comment.
func lexComment(l *Scanner) stateFn {
	l.next()
	return lexEndOfLine(l, Comment)
}

// lexHyperlinkStart scans a hyperlink start.
func lexHyperlinkStart(l *Scanner) stateFn {
	l.next()
	return l.emit(HyperlinkStart)
}

// lexQuote scans a quote.
func lexQuote(l *Scanner) stateFn {
	switch l.types[1] {
	case HyperlinkPrefix, HyperlinkName:
		return l.emit(HyperlinkQuote)
	case Space:
		return l.emit(InlineReferenceOpen)
	case InlineReferenceText:
		return lexInlineReferenceClose
	default:
		return l.errorf("expected hyperlink or inline reference before quote")
	}
}

// lexHyperlinkPrefix scans a hyperlink prefix.
func lexHyperlinkPrefix(l *Scanner) stateFn {
	if strings.HasPrefix(l.input[l.start:], anonHyperlinkPrefix) {
		l.next()
	}
	return l.emit(HyperlinkPrefix)
}

// lexHyperlinkName scans a hyperlink name.
// Escaped colons are part of the hyperlink name.
func lexHyperlinkName(l *Scanner) stateFn {
	for {
		switch l.peek() {
		case ':':
			if l.lastRune != '\\' && l.types[1] != HyperlinkQuote {
				return l.emit(HyperlinkName)
			}
			l.next()
		case '`', eof:
			return l.emit(HyperlinkName)
		case '\n':
			return lexEndOfLine(l, HyperlinkName)
		default:
			l.next()
		}
	}
}

// lexInlineReferenceText scans inline reference text.
func lexInlineReferenceText(l *Scanner) stateFn {
	for {
		switch l.peek() {
		case '_':
			if l.pos == len(l.input)-2 {
				return l.emit(InlineReferenceText)
			}
			l.next()
		case '`', eof:
			return l.emit(InlineReferenceText)
		case '\n':
			return lexEndOfLine(l, InlineReferenceText)
		default:
			l.next()
		}
	}
}

// lexInlineReferenceClose scans an inline reference close.
func lexInlineReferenceClose(l *Scanner) stateFn {
	if l.lastRune == '`' {
		l.next()
	}
	return lexEndOfLine(l, InlineReferenceClose)
}

// isComment reports whether the scanner is on a comment.
func (l *Scanner) isComment(r rune) bool {
	return r == '.' && !strings.HasPrefix(l.input[l.start:], hyperlinkStart)
}

// isHyperlinkStart reports whether the scanner is on a hyperlink start.
func (l *Scanner) isHyperlinkStart(r rune) bool {
	switch r {
	case '.':
		return strings.HasPrefix(l.input[l.start:], hyperlinkStart)
	case '_':
		return strings.HasPrefix(l.input[l.start:], anonHyperlinkStart)
	default:
		return false
	}
}

// isHyperlinkName reports whether the scanner is on a hyperlink name.
func (l *Scanner) isHyperlinkName() bool {
	switch l.types[1] {
	case HyperlinkPrefix:
		return !strings.HasSuffix(l.input[:l.pos], anonHyperlinkPrefix)
	case HyperlinkQuote:
		return l.types[0] == HyperlinkPrefix
	case Space:
		return l.types[0] == HyperlinkName
	default:
		return false
	}
}

// isHyperlinkSuffix reports whether the scanner is on a hyperlink suffix.
func (l *Scanner) isHyperlinkSuffix(r rune) bool {
	if r != ':' {
		return false
	}
	switch l.types[1] {
	case HyperlinkPrefix, HyperlinkName:
		return true
	case HyperlinkQuote:
		return l.types[0] == HyperlinkName
	default:
		return false
	}
}

// isHyperlinkURI reports whether the scanner is on a hyperlink URI.
func (l *Scanner) isHyperlinkURI() bool {
	if isUnderscoreSuffix(l.input[l.pos:]) || l.types[1] != Space {
		return false
	}
	switch l.types[0] {
	case HyperlinkStart, HyperlinkSuffix, HyperlinkURI:
		return true
	default:
		return false
	}
}

// isInlineReferenceText reports whether the scanner is on inline reference text.
func (l *Scanner) isInlineReferenceText() bool {
	if !isUnderscoreSuffix(l.input[l.pos:]) {
		return false
	}
	switch l.types[1] {
	case Space:
		return l.types[0] == HyperlinkSuffix || l.types[0] == InlineReferenceText
	case InlineReferenceOpen:
		return true
	default:
		return false
	}
}

// isUnderscoreSuffix reports whether the string ends with an underscore.
// An escaped underscore is invalid.
func isUnderscoreSuffix(s string) bool {
	if strings.HasSuffix(s, "\\_") || strings.HasSuffix(s, "\\_\n") {
		return false
	}
	return strings.HasSuffix(s, "_") || strings.HasSuffix(s, "_\n")
}

// isInlineReferenceClose reports whether the scanner is on an inline reference close.
func (l *Scanner) isInlineReferenceClose() bool {
	return l.types[1] == InlineReferenceText
}
