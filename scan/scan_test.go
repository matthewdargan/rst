// Copyright 2023 Matthew P. Dargan. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package scan

import (
	"strings"
	"testing"
)

type scanTest struct {
	name  string
	input string
	items []Token
}

func mkItem(typ Type, text string) Token {
	return Token{
		Type: typ,
		Text: text,
	}
}

var (
	tEOF                   = mkItem(EOF, "EOF")
	tBlankLine             = mkItem(BlankLine, "\n")
	tSpace                 = mkItem(Space, " ")
	tSpace2                = mkItem(Space, "  ")
	tSpace3                = mkItem(Space, "   ")
	tComment               = mkItem(Comment, "..")
	tHyperlinkStart        = mkItem(HyperlinkStart, "..")
	tHyperlinkPrefix       = mkItem(HyperlinkPrefix, "_")
	tHyperlinkSuffix       = mkItem(HyperlinkSuffix, ":")
	tInlineReferenceOpen   = mkItem(InlineReferenceOpen, "`")
	tInlineReferenceClose1 = mkItem(InlineReferenceClose, "_")
	tInlineReferenceClose2 = mkItem(InlineReferenceClose, "`_")
)

var scanTests = []scanTest{
	{"empty", "", []Token{tEOF}},
	{"spaces", " \t\n", []Token{mkItem(Space, " \t\n"), tEOF}},
	{"text", `now is the time`, []Token{mkItem(Text, "now is the time"), tEOF}},
	// comments
	{
		"line comment",
		`.. A comment

Paragraph.
`,
		[]Token{tComment, tSpace, mkItem(Text, "A comment"), tBlankLine, mkItem(Text, "Paragraph."), tEOF},
	},
	{
		"comment block",
		`.. A comment
   block.

Paragraph.
`,
		[]Token{
			tComment, tSpace, mkItem(Text, "A comment"), tSpace3, mkItem(Text, "block."),
			tBlankLine, mkItem(Text, "Paragraph."), tEOF,
		},
	},
	{
		"multi-line comment block",
		`..
   A comment consisting of multiple lines
   starting on the line after the
   explicit markup start.
`,
		[]Token{
			tComment, tSpace3, mkItem(Text, "A comment consisting of multiple lines"),
			tSpace3, mkItem(Text, "starting on the line after the"),
			tSpace3, mkItem(Text, "explicit markup start."), tEOF,
		},
	},
	{
		"2 line comments",
		`.. A comment.
.. Another.

Paragraph.
`,
		[]Token{
			tComment, tSpace, mkItem(Text, "A comment."),
			tComment, tSpace, mkItem(Text, "Another."),
			tBlankLine, mkItem(Text, "Paragraph."), tEOF,
		},
	},
	{
		"line comment, no blank line",
		`.. A comment
no blank line

Paragraph.
`,
		[]Token{
			tComment, tSpace, mkItem(Text, "A comment"), mkItem(Text, "no blank line"),
			tBlankLine, mkItem(Text, "Paragraph."), tEOF,
		},
	},
	{
		"2 line comments, no blank line",
		`.. A comment.
.. Another.
no blank line

Paragraph.
`,
		[]Token{
			tComment, tSpace, mkItem(Text, "A comment."),
			tComment, tSpace, mkItem(Text, "Another."), mkItem(Text, "no blank line"),
			tBlankLine, mkItem(Text, "Paragraph."), tEOF,
		},
	},
	{
		"line comment with directive",
		`.. A comment::

Paragraph.
`,
		[]Token{tComment, tSpace, mkItem(Text, "A comment::"), tBlankLine, mkItem(Text, "Paragraph."), tEOF},
	},
	{
		"comment block with directive",
		`..
   comment::

The extra newline before the comment text prevents
the parser from recognizing a directive.
`,
		[]Token{
			tComment, tSpace3, mkItem(Text, "comment::"), tBlankLine,
			mkItem(Text, "The extra newline before the comment text prevents"),
			mkItem(Text, "the parser from recognizing a directive."), tEOF,
		},
	},
	{
		"comment block with hyperlink target",
		`..
   _comment: http://example.org

The extra newline before the comment text prevents
the parser from recognizing a hyperlink target.
`,
		[]Token{
			tComment, tSpace3, mkItem(Text, "_comment: http://example.org"), tBlankLine,
			mkItem(Text, "The extra newline before the comment text prevents"),
			mkItem(Text, "the parser from recognizing a hyperlink target."), tEOF,
		},
	},
	{
		"comment block with citation",
		`..
   [comment] Not a citation.

The extra newline before the comment text prevents
the parser from recognizing a citation.
`,
		[]Token{
			tComment, tSpace3, mkItem(Text, "[comment] Not a citation."), tBlankLine,
			mkItem(Text, "The extra newline before the comment text prevents"),
			mkItem(Text, "the parser from recognizing a citation."), tEOF,
		},
	},
	{
		"comment block with substitution definition",
		`..
   |comment| image:: bogus.png

The extra newline before the comment text prevents
the parser from recognizing a substitution definition.
`,
		[]Token{
			tComment, tSpace3, mkItem(Text, "|comment| image:: bogus.png"), tBlankLine,
			mkItem(Text, "The extra newline before the comment text prevents"),
			mkItem(Text, "the parser from recognizing a substitution definition."), tEOF,
		},
	},
	{
		"comment block and empty comment",
		`.. Next is an empty comment, which serves to end this comment and
   prevents the following block quote being swallowed up.

..

    A block quote.
`,
		[]Token{
			tComment, tSpace, mkItem(Text, "Next is an empty comment, which serves to end this comment and"),
			tSpace3, mkItem(Text, "prevents the following block quote being swallowed up."),
			tBlankLine, tComment, tBlankLine, mkItem(Space, "    "),
			mkItem(Text, "A block quote."), // TODO: Should be itemBlockQuote once implemented
			tEOF,
		},
	},
	{
		"comment in definition lists",
		`term 1
  definition 1

  .. a comment

term 2
  definition 2
`,
		[]Token{
			mkItem(Text, "term 1"), // TODO: Should be itemDefinitionTerm once implemented
			tSpace2, mkItem(Text, "definition 1"), tBlankLine,
			tSpace2, tComment, tSpace, mkItem(Text, "a comment"), tBlankLine,
			mkItem(Text, "term 2"), tSpace2, mkItem(Text, "definition 2"), tEOF,
		},
	},
	{
		"comment after definition lists",
		`term 1
  definition 1

.. a comment

term 2
  definition 2
`,
		[]Token{
			mkItem(Text, "term 1"), // TODO: Should be itemDefinitionTerm once implemented
			tSpace2, mkItem(Text, "definition 1"), tBlankLine,
			tComment, tSpace, mkItem(Text, "a comment"), tBlankLine,
			mkItem(Text, "term 2"), tSpace2, mkItem(Text, "definition 2"), tEOF,
		},
	},
	{
		"comment between bullet paragraphs 2 and 3",
		`+ bullet paragraph 1

  bullet paragraph 2

  .. comment between bullet paragraphs 2 and 3

  bullet paragraph 3
`,
		[]Token{
			mkItem(Text, "+ bullet paragraph 1"), // TODO: Should be itemBullet once implemented
			tBlankLine, tSpace2, mkItem(Text, "bullet paragraph 2"), tBlankLine,
			tSpace2, tComment, tSpace, mkItem(Text, "comment between bullet paragraphs 2 and 3"),
			tBlankLine, tSpace2, mkItem(Text, "bullet paragraph 3"), tEOF,
		},
	},
	{
		"comment between bullet paragraphs 1 and 2",
		`+ bullet paragraph 1

  .. comment between bullet paragraphs 1 (leader) and 2

  bullet paragraph 2
`,
		[]Token{
			mkItem(Text, "+ bullet paragraph 1"), // TODO: Should be itemBullet once implemented
			tBlankLine, tSpace2,
			tComment, tSpace, mkItem(Text, "comment between bullet paragraphs 1 (leader) and 2"),
			tBlankLine, tSpace2, mkItem(Text, "bullet paragraph 2"), tEOF,
		},
	},
	{
		"comment trailing bullet paragraph",
		`+ bullet

  .. trailing comment
`,
		[]Token{
			mkItem(Text, "+ bullet"), // TODO: Should be itemBullet once implemented
			tBlankLine, tSpace2, tComment, tSpace, mkItem(Text, "trailing comment"), tEOF,
		},
	},
	// hyperlink targets
	{
		"hyperlink target",
		`.. _target:

(Internal hyperlink target.)
`,
		[]Token{
			tHyperlinkStart, tSpace, tHyperlinkPrefix,
			mkItem(HyperlinkName, "target"), tHyperlinkSuffix,
			tBlankLine, mkItem(Text, "(Internal hyperlink target.)"), tEOF,
		},
	},
	{
		"hyperlink target with optional space before colon", ".. _optional space before colon :",
		[]Token{
			tHyperlinkStart, tSpace, tHyperlinkPrefix, mkItem(HyperlinkName, "optional space before colon "),
			tHyperlinkSuffix, tEOF,
		},
	},
	{
		"external hyperlink targets",
		`External hyperlink targets:

.. _one-liner: http://structuredtext.sourceforge.net

.. _starts-on-this-line: http://
                         structuredtext.
                         sourceforge.net

.. _entirely-below:
   http://structuredtext.
   sourceforge.net

.. _escaped-whitespace: http://example.org/a\ path\ with\
   spaces.html

.. _not-indirect: uri\_
`,
		[]Token{
			mkItem(Text, "External hyperlink targets:"), tBlankLine,
			tHyperlinkStart, tSpace, tHyperlinkPrefix, mkItem(HyperlinkName, "one-liner"), tHyperlinkSuffix,
			tSpace, mkItem(HyperlinkURI, "http://structuredtext.sourceforge.net"), tBlankLine,
			tHyperlinkStart, tSpace, tHyperlinkPrefix, mkItem(HyperlinkName, "starts-on-this-line"), tHyperlinkSuffix,
			tSpace, mkItem(HyperlinkURI, "http://"), mkItem(Space, "                         "), mkItem(HyperlinkURI, "structuredtext."),
			mkItem(Space, "                         "), mkItem(HyperlinkURI, "sourceforge.net"), tBlankLine,
			tHyperlinkStart, tSpace, tHyperlinkPrefix, mkItem(HyperlinkName, "entirely-below"), tHyperlinkSuffix,
			tSpace3, mkItem(HyperlinkURI, "http://structuredtext."), tSpace3, mkItem(HyperlinkURI, "sourceforge.net"), tBlankLine,
			tHyperlinkStart, tSpace, tHyperlinkPrefix, mkItem(HyperlinkName, "escaped-whitespace"), tHyperlinkSuffix,
			tSpace, mkItem(HyperlinkURI, `http://example.org/a\ path\ with\`), tSpace3, mkItem(HyperlinkURI, "spaces.html"), tBlankLine,
			tHyperlinkStart, tSpace, tHyperlinkPrefix, mkItem(HyperlinkName, "not-indirect"), tHyperlinkSuffix,
			tSpace, mkItem(HyperlinkURI, `uri\_`), tEOF,
		},
	},
	{
		"indirect hyperlink targets",
		`Indirect hyperlink targets:

.. _target1: reference_

` + ".. _target2: `phrase-link reference`_",
		[]Token{
			mkItem(Text, "Indirect hyperlink targets:"), tBlankLine,
			tHyperlinkStart, tSpace, tHyperlinkPrefix, mkItem(HyperlinkName, "target1"), tHyperlinkSuffix,
			tSpace, mkItem(InlineReferenceText, "reference"), tInlineReferenceClose1, tBlankLine,
			tHyperlinkStart, tSpace, tHyperlinkPrefix, mkItem(HyperlinkName, "target2"), tHyperlinkSuffix,
			tSpace, tInlineReferenceOpen, mkItem(InlineReferenceText, "phrase-link reference"), tInlineReferenceClose2,
			tEOF,
		},
	},
}

// collect gathers the emitted items into a slice.
func collect(t *scanTest) (items []Token) {
	s := New(t.name, strings.NewReader(t.input))
	for {
		i := s.Next()
		items = append(items, i)
		if i.Type == EOF || i.Type == Error {
			break
		}
	}
	return
}

func equal(i1, i2 []Token, checkPos bool) bool {
	if len(i1) != len(i2) {
		return false
	}
	for k := range i1 {
		if i1[k].Type != i2[k].Type {
			return false
		}
		if i1[k].Text != i2[k].Text {
			return false
		}
		if checkPos && i1[k].Line != i2[k].Line {
			return false
		}
	}
	return true
}

func TestScan(t *testing.T) {
	for _, test := range scanTests {
		items := collect(&test)
		if !equal(items, test.items, false) {
			t.Fatalf("%s: got\n\t%+v\nexpected\n\t%v", test.name, items, test.items)
		}
	}
}
