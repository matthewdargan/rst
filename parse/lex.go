package parse

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// item represents a token or text string returned from the scanner.
type item struct {
	typ  itemType // The type of this item.
	pos  Pos      // The starting position, in bytes, of this item in the input string.
	val  string   // The value of this item.
	line int      // The line number at the start of this item.
}

func (i item) String() string {
	switch {
	case i.typ == itemEOF:
		return "EOF"
	case i.typ == itemError:
		return i.val
	case len(i.val) > 10:
		return fmt.Sprintf("typ: %q - val: %.10q...", i.typ, i.val)
	}
	return fmt.Sprintf("typ: %q - val: %q", i.typ, i.val)
}

// itemType identifies the type of lex items.
type itemType int

const (
	itemError   itemType = iota // error occurred; value is text of error
	itemComment                 // comment text
	itemEOF
	itemNewLine // newline separating constructs
	itemSpace   // run of spaces separating arguments
	itemText    // plain text
)

const eof = -1

// stateFn represents the state of the scanner as a function that returns the next state.
type stateFn func(*lexer) stateFn

// lexer holds the state of the scanner.
type lexer struct {
	name      string // the name of the input; used only for error reports
	input     string // the string being scanned
	pos       Pos    // current position in the input
	start     Pos    // start position of this item
	atEOF     bool   // we have hit the end of input and returned eof
	line      int    // 1+number of newlines seen
	startLine int    // start line of this item
	item      item   // item to return to parser
}

// next returns the next rune in the input.
func (l *lexer) next() rune {
	if int(l.pos) >= len(l.input) {
		l.atEOF = true
		return eof
	}
	r, w := utf8.DecodeRuneInString(l.input[l.pos:])
	l.pos += Pos(w)
	if r == '\n' {
		l.line++
	}
	return r
}

// peek returns but does not consume the next rune in the input.
func (l *lexer) peek() rune {
	r := l.next()
	l.backup()
	return r
}

// backup steps back one rune.
func (l *lexer) backup() {
	if !l.atEOF && l.pos > 0 {
		r, w := utf8.DecodeLastRuneInString(l.input[:l.pos])
		l.pos -= Pos(w)
		// Correct newline count.
		if r == '\n' {
			l.line--
		}
	}
}

// thisItem returns the item at the current input point with the specified type
// and advances the input.
func (l *lexer) thisItem(t itemType) item {
	i := item{t, l.start, l.input[l.start:l.pos], l.startLine}
	l.start = l.pos
	l.startLine = l.line
	return i
}

// emit passes the trailing text as an item back to the parser.
func (l *lexer) emit(t itemType) stateFn {
	return l.emitItem(l.thisItem(t))
}

// emitItem passes the specified item to the parser.
func (l *lexer) emitItem(i item) stateFn {
	l.item = i
	return nil
}

// ignore skips over the pending input before this point.
// It tracks newlines in the ignored text, so use it only
// for text that is skipped without calling l.next.
func (l *lexer) ignore() {
	l.line += strings.Count(l.input[l.start:l.pos], "\n")
	l.start = l.pos
	l.startLine = l.line
}

// accept consumes the next rune if it's from the valid set.
func (l *lexer) accept(valid string) bool {
	if strings.ContainsRune(valid, l.next()) {
		return true
	}
	l.backup()
	return false
}

// acceptRun consumes a run of runes from the valid set.
func (l *lexer) acceptRun(valid string) {
	for strings.ContainsRune(valid, l.next()) {
	}
	l.backup()
}

// errorf returns an error token and terminates the scan by passing
// back a nil pointer that will be the next state, terminating l.nextItem.
func (l *lexer) errorf(format string, args ...any) stateFn {
	l.item = item{itemError, l.start, fmt.Sprintf(format, args...), l.startLine}
	l.start = 0
	l.pos = 0
	l.input = l.input[:0]
	return nil
}

// nextItem returns the next item from the input.
// Called by the parser, not in the lexing goroutine.
func (l *lexer) nextItem() item {
	l.item = item{itemEOF, l.pos, "EOF", l.startLine}
	state := lexAction
	for {
		state = state(l)
		if state == nil {
			return l.item
		}
	}
}

// lex creates a new scanner for the input string.
func lex(name, input string) *lexer {
	l := &lexer{
		name:      name,
		input:     input,
		line:      1,
		startLine: 1,
	}
	return l
}

// lexAction scans the elements inside action delimiters.
func lexAction(l *lexer) stateFn {
	switch r := l.next(); {
	case r == eof:
		return l.emit(itemEOF)
	case r == '\n':
		return l.emit(itemNewLine)
	case unicode.IsSpace(r):
		return lexSpace
	case r == '.' && l.peek() == '.':
		return lexComment
	default:
		return lexText
	}
}

// lexSpace scans a run of space characters.
// We have not consumed the first space, which is known to be present.
// Take care if there is a trim-marked right delimiter, which starts with a space.
func lexSpace(l *lexer) stateFn {
	var r rune
	for {
		r = l.peek()
		if !unicode.IsSpace(r) {
			break
		}
		l.next()
	}
	return l.emit(itemSpace)
}

// lexComment scans a comment. The comment marker is known to be present.
func lexComment(l *lexer) stateFn {
	l.next()
	i := l.thisItem(itemComment)
	if r := l.peek(); r == '\n' {
		l.pos++
		l.ignore()
	}
	return l.emitItem(i)
}

// lexText scans until a newline or EOF.
func lexText(l *lexer) stateFn {
	if nl := strings.IndexRune(l.input[l.pos:], '\n'); nl > 0 {
		l.pos += Pos(nl)
		i := l.thisItem(itemText)
		l.pos++
		l.ignore()
		return l.emitItem(i)
	}
	l.pos = Pos(len(l.input))
	// Correctly reached EOF.
	if l.pos > l.start {
		l.line += strings.Count(l.input[l.start:l.pos], "\n")
		return l.emit(itemText)
	}
	return l.emit(itemEOF)
}
