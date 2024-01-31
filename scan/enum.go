// Copyright 2023 Matthew P. Dargan. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package scan

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// lexEnum scans an enumeration.
func lexEnum(l *Scanner) stateFn {
	for {
		switch r := l.peek(); {
		case r == '\n':
			return lexEndOfLine(l, Enum)
		case r == eof, unicode.IsSpace(r):
			return l.emit(Enum)
		}
		l.next()
	}
}

// isEnum reports whether the scanner is on an enumeration.
func (l *Scanner) isEnum(r rune) bool {
	if l.types[0] == BlankLine && l.types[1] == Paragraph {
		return false
	}
	pos, lastWidth := l.pos, l.lastWidth
	defer func() {
		l.pos, l.lastWidth = pos, lastWidth
	}()
	if r == '(' {
		r = l.next()
	}
	i, ok := l.enumSuffix()
	if !ok {
		return false
	}
	e, ok := l.enum(r, i)
	if !ok {
		return false
	}
	l.lastEnum = e
	for r != eof && r != '\n' {
		r = l.next()
	}
	if r == eof {
		return true
	}
	r = l.next()
	if r == '(' {
		r = l.next()
	}
	if !unicode.IsDigit(r) && !unicode.IsLetter(r) {
		return true
	}
	i, ok = l.enumSuffix()
	if !ok {
		return false
	}
	_, ok = l.enum(r, i)
	return ok
}

const (
	enumSuffixes   = ".)"
	roman          = "Ii"
	ambiguousRoman = "VXLCDMvxlcdm"
)

// enumSuffix returns the index of the first enumeration suffix character.
func (l *Scanner) enumSuffix() (int, bool) {
	i := strings.IndexAny(l.input[l.pos:], enumSuffixes)
	if i < 0 {
		return i, false
	}
	if l.pos+i+1 < len(l.input) && !unicode.IsSpace(rune(l.input[l.pos+i+1])) {
		return i, false
	}
	return i, true
}

type enumType int

const (
	none enumType = iota
	arabic
	upperAlpha
	lowerAlpha
	upperRoman
	lowerRoman
)

type enum struct {
	typ  enumType
	val  int
	auto bool
}

// enum interprets an enumeration up to index i.
func (l *Scanner) enum(r rune, i int) (enum, bool) {
	var e enum
	if l.lastEnum.auto && r != '#' {
		return e, false
	}
	switch {
	case unicode.IsDigit(r):
		n, _ := strconv.Atoi(l.input[l.pos-1 : l.pos+i])
		e = enum{typ: arabic, val: n}
	case unicode.IsLetter(r):
		switch {
		case l.isRoman(r):
			n, ok := parseRoman(l.input[l.pos-1 : l.pos+i])
			if !ok {
				return e, false
			}
			e = enum{typ: upperRoman, val: n}
			if unicode.IsLower(r) {
				e.typ = lowerRoman
			}
		case i > 0:
			return e, false
		default:
			e = enum{typ: upperAlpha, val: int(r - '0')}
			if unicode.IsLower(r) {
				e.typ = lowerAlpha
			}
		}
	case r == '#' && i == 0:
		e = enum{typ: l.lastEnum.typ, val: l.lastEnum.val + 1, auto: true}
	default:
		return e, false
	}
	if e.typ == l.lastEnum.typ && e.val-l.lastEnum.val != 1 {
		return e, false
	}
	return e, true
}

// isRoman reports whether r is a roman numeral.
func (l *Scanner) isRoman(r rune) bool {
	switch {
	case strings.ContainsRune(roman, r):
		return true
	case strings.ContainsRune(ambiguousRoman, r):
		return l.lastEnum.typ == upperRoman || l.lastEnum.typ == lowerRoman
	}
	return false
}

var (
	nums       = map[rune]int{'I': 1, 'V': 5, 'X': 10, 'L': 50, 'C': 100, 'D': 500, 'M': 1000}
	numPattern = regexp.MustCompile("^M{0,4}(CM|CD|D?C{0,3})(XC|XL|L?X{0,3})(IX|IV|V?I{0,3})$")
)

// parseRoman converts a roman numeral to an integer.
func parseRoman(s string) (int, bool) {
	s = strings.ToUpper(s)
	if !numPattern.MatchString(s) {
		return 0, false
	}
	var sum int
	for _, r := range s {
		sum += nums[r]
	}
	return sum, true
}
