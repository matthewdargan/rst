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
	name      string        // the name of the input; used only for error reports
	buf       []byte        // I/O buffer, re-used.
	input     string        // the line of text being scanned.
	lastRune  rune          // most recent return from next()
	lastWidth int           // size of that rune
	readOK    bool          // allow reading of a new line of input
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
		if !l.readOK { // Token did not end before newline.
			l.errorf("incomplete token")
			return '\n', 1
		}
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
	l.readOK = true
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

const hyperlinkMark = ".. _"

// lexAny scans non-space items.
func lexAny(l *Scanner) stateFn {
	switch r := l.next(); {
	case r == eof:
		return nil
	case r == '\n':
		return l.emit(BlankLine)
	case unicode.IsSpace(r):
		return lexSpace
	case strings.HasPrefix(l.input[l.start:], hyperlinkMark):
		l.next()
		return l.emit(HyperlinkStart)
	case r == '.' && l.peek() == '.':
		l.next()
		return lexEndOfLine(l, Comment)
	case l.input[:l.pos] == hyperlinkMark:
		return l.emit(HyperlinkPrefix)
	case strings.HasSuffix(l.input[:l.start], hyperlinkMark):
		return lexHyperlinkName
	case r == ':' && l.types[1] == HyperlinkName:
		return lexEndOfLine(l, HyperlinkSuffix)
	// TODO: Handle indirect hyperlink InlineReferenceText (everything before _), InlineReferenceClose here (_). CANNOT BE \_ (this would be external)
	case l.types[1] == Space && (l.types[0] == HyperlinkSuffix || l.types[0] == HyperlinkURI):
		return lexUntilTerminator(l, HyperlinkURI)
	default:
		return lexUntilTerminator(l, Text)
	}
}

// lexSpace scans a run of space characters.
func lexSpace(l *Scanner) stateFn {
	for unicode.IsSpace(l.peek()) {
		l.next()
	}
	return l.emit(Space)
}

// lexEndOfLine scans a lex item known to be present.
// An end-of-line character is ignored if it is present.
func lexEndOfLine(l *Scanner, typ Type) stateFn {
	i := l.emit(typ)
	if l.peek() == '\n' {
		l.pos++
		l.ignore()
	}
	return i
}

// lexHyperlinkName scans a hyperlink name. The hyperlink name is known to be present.
func lexHyperlinkName(l *Scanner) stateFn {
	for l.peek() != ':' {
		l.next()
	}
	return l.emit(HyperlinkName)
}

// lexUntilTerminator scans a lex item until a newline or EOF.
func lexUntilTerminator(l *Scanner, typ Type) stateFn {
	for {
		switch l.peek() {
		case eof:
			return l.emit(typ)
		case '\n':
			i := l.emit(typ)
			l.pos++
			l.ignore()
			return i
		default:
			l.next()
		}
	}
}
