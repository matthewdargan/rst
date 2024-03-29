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
	Space                            // Space indents elements
	Title                            // Title identifies a section
	SectionAdornment                 // SectionAdornment underlines or overlines a title
	Transition                       // Transition separate other body elements
	Paragraph                        // Paragraph is left-aligned text with no markup
	Bullet                           // Bullet starts a bullet list
	Enum                             // Enum starts an enumerated list
	BlockQuote                       // BlockQuote indents a text block relative to the preceding text
	Attribution                      // Attribution is a text block that ends a block quote
	Comment                          // Comment starts a comment
	HyperlinkStart                   // HyperlinkStart starts a hyperlink target
	HyperlinkPrefix                  // HyperlinkPrefix prefixes a hyperlink target name
	HyperlinkQuote                   // HyperlinkQuote encloses a hyperlink target name that contains any colons
	HyperlinkName                    // HyperlinkName identifies a hyperlink target for cross-referencing
	HyperlinkSuffix                  // HyperlinkSuffix suffixes a hyperlink target name
	HyperlinkURI                     // HyperlinkURI is the URI a hyperlink target points to
	InlineReferenceOpen              // InlineReferenceOpen opens an inline reference
	InlineReferenceText              // InlineReferenceText is reference text a hyperlink target points to
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
	r          io.ByteReader // reads input bytes
	done       bool          // are we done scanning?
	name       string        // name of the input; used only for error reports
	buf        []byte        // I/O buffer, re-used
	input      string        // line of text being scanned
	lastRune   rune          // most recent return from next()
	lastWidth  int           // size of that rune
	line       int           // line number in input
	pos        int           // current position in the input
	start      int           // start position of this item
	token      Token         // token to return to parser
	types      [2]Type       // most recent scanned types
	indent     int           // current indentation level in the input
	lastMarkup Type          // most recent markup type
	lastEnum   enum          // most recent enumeration
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
		l.lastEnum = enum{typ: none, val: 0}
	}
	if l.start == 0 && notSpace(rune(l.input[l.start])) {
		l.indent = 0
	}
	text := l.input[l.start:l.pos]
	l.token = Token{t, l.line, text}
	l.types[0] = l.types[1]
	l.types[1] = t
	l.start = l.pos
	return nil
}

// notSpace reports whether the rune is not a space character.
func notSpace(c rune) bool {
	return !unicode.IsSpace(c)
}

// ignore skips over the pending input before this point.
// It tracks newlines in the ignored text, so use it only
// for text that is skipped without calling l.next.
func (l *Scanner) ignore() {
	l.line += strings.Count(l.input[l.start:l.pos], "\n")
	l.start = l.pos
}

// errorf returns an error token and empties the input.
func (l *Scanner) errorf(format string, args ...any) stateFn {
	l.token = Token{Error, l.start, fmt.Sprintf(format, args...)}
	l.start = 0
	l.pos = 0
	l.input = l.input[:0]
	return nil
}

// New creates and returns a new scanner.
func New(name string, r io.ByteReader) *Scanner {
	return &Scanner{r: r, name: name, line: 1}
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
	comment                   = ".."
	hyperlinkStart            = ".. _"
	anonHyperlinkStart        = "__ "
	anonHyperlinkPrefix       = "__:"
	bullets                   = "*+-•‣⁃"
	adornments                = "!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~"
	minSection, minTransition = 2, 4
)

// lexAny scans any item.
func lexAny(l *Scanner) stateFn {
	switch r := l.next(); {
	case r == eof:
		return nil
	case r == '\n':
		return lexBlankLine
	case l.isBlockQuote():
		return lexSpace(l, BlockQuote)
	case l.isAttribution():
		return lexAttribution
	case unicode.IsSpace(r):
		return lexSpace(l, Space)
	case l.isBullet(r):
		return lexBullet
	case l.isComment():
		return lexComment
	case l.isTransition(r):
		return lexTransition
	case l.isSectionAdornment(r):
		return lexSection
	case l.isHyperlinkStart():
		return lexHyperlinkStart
	case l.isHyperlinkPrefix():
		return lexHyperlinkPrefix
	case r == '`':
		return lexQuote
	case l.isHyperlinkName():
		return lexHyperlinkName
	case l.isHyperlinkSuffix():
		return lexEndOfLine(l, HyperlinkSuffix)
	case l.isHyperlinkURI():
		return lexUntilTerminator(l, HyperlinkURI)
	case l.isInlineReferenceText():
		return lexInlineReferenceText
	case l.isInlineReferenceClose():
		return lexInlineReferenceClose
	case l.isTitle():
		return lexTitle
	case l.isEnum(r):
		return lexEnum
	default:
		return lexParagraph
	}
}

// lexEndOfLine scans an item and ignores an end-of-line character if present.
func lexEndOfLine(l *Scanner, typ Type) stateFn {
	i := l.emit(typ)
	if l.peek() == '\n' {
		l.pos++
		l.ignore()
	}
	return i
}

// lexUntilTerminator scans an item until a newline or end-of-line character.
func lexUntilTerminator(l *Scanner, typ Type) stateFn {
	for {
		switch l.peek() {
		case eof:
			return l.emit(typ)
		case '\n':
			return lexEndOfLine(l, typ)
		}
		l.next()
	}
}

// lexBlankLine scans a blank line.
func lexBlankLine(l *Scanner) stateFn {
	if l.types[1] == Comment {
		l.lastMarkup = EOF
	}
	return l.emit(BlankLine)
}

// lexSpace scans a run of space characters.
func lexSpace(l *Scanner, typ Type) stateFn {
	var i int
	for i = 1; unicode.IsSpace(l.peek()); i++ {
		l.next()
	}
	if l.start == 0 || l.input[l.start-1] == '\n' {
		l.indent = i
	}
	return l.emit(typ)
}

// lexAttribution scans an attribution.
func lexAttribution(l *Scanner) stateFn {
	l.indent = 0
	return lexUntilTerminator(l, Attribution)
}

// lexBullet scans a bullet.
func lexBullet(l *Scanner) stateFn {
	l.lastMarkup = Bullet
	return lexEndOfLine(l, Bullet)
}

// lexComment scans a comment.
func lexComment(l *Scanner) stateFn {
	l.lastMarkup = Comment
	l.next()
	return lexEndOfLine(l, Comment)
}

// lexTransition scans a transition.
func lexTransition(l *Scanner) stateFn {
	l.lastMarkup = Transition
	return lexUntilTerminator(l, Transition)
}

// lexSection scans a section adornment.
func lexSection(l *Scanner) stateFn {
	l.lastMarkup = SectionAdornment
	return lexUntilTerminator(l, SectionAdornment)
}

// lexHyperlinkStart scans a hyperlink start.
func lexHyperlinkStart(l *Scanner) stateFn {
	l.lastMarkup = HyperlinkStart
	l.next()
	return l.emit(HyperlinkStart)
}

// lexHyperlinkPrefix scans a hyperlink prefix.
func lexHyperlinkPrefix(l *Scanner) stateFn {
	if strings.HasPrefix(l.input[l.start:], anonHyperlinkPrefix) {
		l.next()
	}
	return l.emit(HyperlinkPrefix)
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
	}
	return l.errorf("expected hyperlink or inline reference before quote")
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
			if l.pos > len(l.input)-3 {
				return l.emit(InlineReferenceText)
			}
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

// lexTitle scans a title.
func lexTitle(l *Scanner) stateFn {
	l.lastMarkup = Title
	return lexUntilTerminator(l, Title)
}

// lexParagraph scans a paragraph.
func lexParagraph(l *Scanner) stateFn {
	if l.start == 0 && l.indent == 0 {
		l.lastMarkup = EOF
	}
	return lexUntilTerminator(l, Paragraph)
}

// isBlockQuote reports whether the scanner is on a block quote.
func (l *Scanner) isBlockQuote() bool {
	if l.lastMarkup != EOF {
		return false
	}
	switch l.types[0] {
	case Paragraph, Attribution, Comment:
	default:
		if l.types[1] != Paragraph || (l.types[1] != BlankLine && l.indent > 0) {
			return false
		}
	}
	i := strings.IndexFunc(l.input[l.start:], notSpace)
	return i > 0 && i != l.indent
}

// isAttribution reports whether the scanner is on an attribution.
func (l *Scanner) isAttribution() bool {
	if l.types[0] == Attribution && l.types[1] == Space {
		return true
	}
	if l.types[1] == BlockQuote {
		return false
	}
	s := l.input[l.start:]
	if !strings.HasPrefix(s, "--") && !strings.HasPrefix(s, "—") {
		return false
	}
	s = strings.TrimSpace(strings.TrimLeft(s, "-"))
	if len(s) == 0 || strings.ContainsAny(s, "-") {
		return false
	}
	pos, lastWidth := l.pos, l.lastWidth
	var r rune
	for r != eof && r != '\n' {
		r = l.next()
	}
	l.next()
	i := strings.IndexFunc(l.input[l.pos-1:], notSpace)
	l.pos, l.lastWidth = pos, lastWidth
	return i < 1 || i == l.indent
}

// isBullet reports whether the scanner is on a bullet.
func (l *Scanner) isBullet(r rune) bool {
	return strings.ContainsRune(bullets, r) && unicode.IsSpace(l.peek())
}

// isComment reports whether the scanner is on a comment.
func (l *Scanner) isComment() bool {
	if l.types[1] == Title {
		return false
	}
	s := l.input[l.start:]
	if strings.HasPrefix(s, hyperlinkStart) && len(s) > len(hyperlinkStart) {
		return false
	}
	return strings.HasPrefix(s, comment+" ") || strings.HasPrefix(s, comment+"\n")
}

// isTransition reports whether the scanner is on a transition.
func (l *Scanner) isTransition(r rune) bool {
	switch l.types[1] {
	case EOF, BlankLine:
	case Space:
		if l.types[0] != BlankLine {
			return false
		}
	default:
		return false
	}
	if !strings.ContainsRune(adornments, r) {
		return false
	}
	s := strings.TrimSuffix(l.input[l.start:], "\n")
	if len(s) < minTransition || s != strings.Repeat(string(r), len(s)) {
		return false
	}
	pos, lastWidth := l.pos, l.lastWidth
	for r != eof && r != '\n' {
		r = l.next()
	}
	r = l.peek()
	l.pos, l.lastWidth = pos, lastWidth
	if r != eof && r != '\n' {
		return false
	}
	return true
}

// isSectionAdornment reports whether the scanner is on a section adornment.
func (l *Scanner) isSectionAdornment(r rune) bool {
	if l.lastMarkup == Title {
		return true
	}
	if !l.isSection(r) {
		return false
	}
	if l.types[1] != BlankLine {
		return true
	}
	pos, lastWidth := l.pos, l.lastWidth
	for r != eof && r != '\n' {
		r = l.next()
	}
	r = l.peek()
	l.pos, l.lastWidth = pos, lastWidth
	return r != '\n'
}

// isSection reports whether the scanner is on a section.
func (l *Scanner) isSection(r rune) bool {
	if !strings.ContainsRune(adornments, r) {
		return false
	}
	s := strings.SplitN(l.input[l.pos-1:], "\n", 2)[0]
	return len(s) >= minSection && s == strings.Repeat(string(r), len(s))
}

// isHyperlinkStart reports whether the scanner is on a hyperlink start.
func (l *Scanner) isHyperlinkStart() bool {
	s := l.input[l.start:]
	return strings.HasPrefix(s, hyperlinkStart) || strings.HasPrefix(s, anonHyperlinkStart)
}

// isHyperlinkPrefix reports whether the scanner is on a hyperlink prefix.
func (l *Scanner) isHyperlinkPrefix() bool {
	switch l.peek() {
	case '\n', eof:
		return false
	}
	return l.input[:l.pos] == hyperlinkStart
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
	}
	return false
}

// isHyperlinkSuffix reports whether the scanner is on a hyperlink suffix.
func (l *Scanner) isHyperlinkSuffix() bool {
	switch l.types[1] {
	case HyperlinkPrefix, HyperlinkName:
		return true
	case HyperlinkQuote:
		return l.types[0] == HyperlinkName
	}
	return false
}

// isHyperlinkURI reports whether the scanner is on a hyperlink URI.
func (l *Scanner) isHyperlinkURI() bool {
	s := strings.TrimSuffix(l.input[l.pos:], "\n")
	if isUnderscoreSuffix(s) && !strings.ContainsFunc(s, unicode.IsSpace) {
		return false
	}
	switch l.types[0] {
	case HyperlinkStart, HyperlinkSuffix, HyperlinkURI:
		return l.types[1] == Space
	}
	return false
}

// isUnderscoreSuffix reports whether the string ends with an underscore.
// An escaped underscore is invalid.
func isUnderscoreSuffix(s string) bool {
	return strings.HasSuffix(s, "_") && !strings.HasSuffix(s, "\\_")
}

// isInlineReferenceText reports whether the scanner is on inline reference text.
func (l *Scanner) isInlineReferenceText() bool {
	switch l.types[1] {
	case Space:
		switch l.types[0] {
		case HyperlinkStart, HyperlinkSuffix, InlineReferenceText:
			return true
		}
	case InlineReferenceOpen:
		return true
	}
	return false
}

// isInlineReferenceClose reports whether the scanner is on an inline reference close.
func (l *Scanner) isInlineReferenceClose() bool {
	return l.types[1] == InlineReferenceText
}

// isTitle reports whether the scanner is on a title.
func (l *Scanner) isTitle() bool {
	pos, lastWidth := l.pos, l.lastWidth
	var r rune
	for r != eof && r != '\n' {
		r = l.next()
	}
	r = l.next()
	if i := strings.IndexFunc(l.input[l.pos:], notSpace); i > 0 {
		l.pos += i
		r = l.next()
	}
	ok := l.isSection(r)
	l.pos, l.lastWidth = pos, lastWidth
	return ok
}
