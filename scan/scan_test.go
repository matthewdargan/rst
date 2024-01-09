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
	tAnonHyperlinkStart    = mkItem(HyperlinkStart, "__")
	tHyperlinkPrefix       = mkItem(HyperlinkPrefix, "_")
	tAnonHyperlinkPrefix   = mkItem(HyperlinkPrefix, "__")
	tHyperlinkQuote        = mkItem(HyperlinkQuote, "`")
	tHyperlinkSuffix       = mkItem(HyperlinkSuffix, ":")
	tInlineReferenceOpen   = mkItem(InlineReferenceOpen, "`")
	tInlineReferenceClose1 = mkItem(InlineReferenceClose, "_")
	tInlineReferenceClose2 = mkItem(InlineReferenceClose, "`_")
)

var scanTests = []scanTest{
	{"empty", "", []Token{tEOF}},
	{"spaces", " \t\n", []Token{mkItem(Space, " \t\n"), tEOF}},
	{"quote error", "`", []Token{mkItem(Error, "expected hyperlink or inline reference before quote")}},
	{"text", `now is the time`, []Token{mkItem(Text, "now is the time"), tEOF}},
	// comments
	{
		"line comment",
		`.. A comment

Paragraph.`,
		[]Token{tComment, tSpace, mkItem(Text, "A comment"), tBlankLine, mkItem(Text, "Paragraph."), tEOF},
	},
	{
		"comment block",
		`.. A comment
   block.

Paragraph.`,
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
   explicit markup start.`,
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

Paragraph.`,
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

Paragraph.`,
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

Paragraph.`,
		[]Token{
			tComment, tSpace, mkItem(Text, "A comment."),
			tComment, tSpace, mkItem(Text, "Another."), mkItem(Text, "no blank line"),
			tBlankLine, mkItem(Text, "Paragraph."), tEOF,
		},
	},
	{
		"line comment with directive",
		`.. A comment::

Paragraph.`,
		[]Token{tComment, tSpace, mkItem(Text, "A comment::"), tBlankLine, mkItem(Text, "Paragraph."), tEOF},
	},
	{
		"comment block with directive",
		`..
   comment::

The extra newline before the comment text prevents
the parser from recognizing a directive.`,
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
the parser from recognizing a hyperlink target.`,
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
the parser from recognizing a citation.`,
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
the parser from recognizing a substitution definition.`,
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

    A block quote.`,
		[]Token{
			tComment, tSpace, mkItem(Text, "Next is an empty comment, which serves to end this comment and"),
			tSpace3, mkItem(Text, "prevents the following block quote being swallowed up."),
			tBlankLine, tComment, tBlankLine, mkItem(Space, "    "),
			mkItem(Text, "A block quote."), // TODO: Should be BlockQuote once implemented
			tEOF,
		},
	},
	{
		"comment in definition lists",
		`term 1
  definition 1

  .. a comment

term 2
  definition 2`,
		[]Token{
			mkItem(Text, "term 1"), // TODO: Should be DefinitionTerm once implemented
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
  definition 2`,
		[]Token{
			mkItem(Text, "term 1"), // TODO: Should be DefinitionTerm once implemented
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

  bullet paragraph 3`,
		[]Token{
			mkItem(Text, "+ bullet paragraph 1"), // TODO: Should be Bullet once implemented
			tBlankLine, tSpace2, mkItem(Text, "bullet paragraph 2"), tBlankLine,
			tSpace2, tComment, tSpace, mkItem(Text, "comment between bullet paragraphs 2 and 3"),
			tBlankLine, tSpace2, mkItem(Text, "bullet paragraph 3"), tEOF,
		},
	},
	{
		"comment between bullet paragraphs 1 and 2",
		`+ bullet paragraph 1

  .. comment between bullet paragraphs 1 (leader) and 2

  bullet paragraph 2`,
		[]Token{
			mkItem(Text, "+ bullet paragraph 1"), // TODO: Should be Bullet once implemented
			tBlankLine, tSpace2,
			tComment, tSpace, mkItem(Text, "comment between bullet paragraphs 1 (leader) and 2"),
			tBlankLine, tSpace2, mkItem(Text, "bullet paragraph 2"), tEOF,
		},
	},
	{
		"comment trailing bullet paragraph",
		`+ bullet

  .. trailing comment`,
		[]Token{
			mkItem(Text, "+ bullet"), // TODO: Should be Bullet once implemented
			tBlankLine, tSpace2, tComment, tSpace, mkItem(Text, "trailing comment"), tEOF,
		},
	},
	// targets
	{
		"hyperlink target",
		`.. _target:

(Internal hyperlink target.)`,
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

.. _not-indirect: uri\_`,
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
	{
		"escaped hyperlink target names",
		`.. _a long target name:

` + ".. _`a target name: including a colon (quoted)`:" + `

.. _a target name\: including a colon (escaped):`,
		[]Token{
			tHyperlinkStart, tSpace, tHyperlinkPrefix, mkItem(HyperlinkName, "a long target name"), tHyperlinkSuffix, tBlankLine,
			tHyperlinkStart, tSpace, tHyperlinkPrefix, tHyperlinkQuote, mkItem(HyperlinkName, "a target name: including a colon (quoted)"),
			tHyperlinkQuote, tHyperlinkSuffix, tBlankLine,
			tHyperlinkStart, tSpace, tHyperlinkPrefix, mkItem(HyperlinkName, `a target name\: including a colon (escaped)`), tHyperlinkSuffix,
			tEOF,
		},
	},
	{
		"hyperlink target names with no matching backquotes",
		".. _`target: No matching backquote.\n.. _`: No matching backquote either.",
		[]Token{
			tHyperlinkStart, tSpace, tHyperlinkPrefix, tHyperlinkQuote, mkItem(HyperlinkName, "target: No matching backquote."),
			tHyperlinkStart, tSpace, tHyperlinkPrefix, tHyperlinkQuote, mkItem(HyperlinkName, ": No matching backquote either."), tEOF,
		},
	},
	{
		"hyperlink target names split across lines, 1 regular, 1 backquoted",
		`.. _a very long target name,
   split across lines:
` + ".. _`and another,\n   with backquotes`:",
		[]Token{
			tHyperlinkStart, tSpace, tHyperlinkPrefix, mkItem(HyperlinkName, "a very long target name,"),
			tSpace3, mkItem(HyperlinkName, "split across lines"), tHyperlinkSuffix,
			tHyperlinkStart, tSpace, tHyperlinkPrefix, tHyperlinkQuote, mkItem(HyperlinkName, "and another,"),
			tSpace3, mkItem(HyperlinkName, "with backquotes"), tHyperlinkQuote, tHyperlinkSuffix, tEOF,
		},
	},
	{
		"external hyperlink target",
		`External hyperlink:

.. _target: http://www.python.org/`,
		[]Token{
			mkItem(Text, "External hyperlink:"), tBlankLine,
			tHyperlinkStart, tSpace, tHyperlinkPrefix, mkItem(HyperlinkName, "target"), tHyperlinkSuffix,
			tSpace, mkItem(HyperlinkURI, "http://www.python.org/"), tEOF,
		},
	},
	{
		"email targets",
		`.. _email: jdoe@example.com

.. _multi-line email: jdoe
   @example.com`,
		[]Token{
			tHyperlinkStart, tSpace, tHyperlinkPrefix, mkItem(HyperlinkName, "email"), tHyperlinkSuffix,
			tSpace, mkItem(HyperlinkURI, "jdoe@example.com"), tBlankLine,
			tHyperlinkStart, tSpace, tHyperlinkPrefix, mkItem(HyperlinkName, "multi-line email"), tHyperlinkSuffix,
			tSpace, mkItem(HyperlinkURI, "jdoe"), tSpace3, mkItem(HyperlinkURI, "@example.com"), tEOF,
		},
	},
	{
		"malformed target",
		`Malformed target:

.. __malformed: no good

Target beginning with an underscore:

` + ".. _`_target`: OK",
		[]Token{
			mkItem(Text, "Malformed target:"), tBlankLine,
			tHyperlinkStart, tSpace, tHyperlinkPrefix, mkItem(HyperlinkName, "_malformed"), tHyperlinkSuffix,
			tSpace, mkItem(HyperlinkURI, "no good"), tBlankLine,
			mkItem(Text, "Target beginning with an underscore:"), tBlankLine,
			tHyperlinkStart, tSpace, tHyperlinkPrefix, tHyperlinkQuote, mkItem(HyperlinkName, "_target"), tHyperlinkQuote, tHyperlinkSuffix,
			tSpace, mkItem(HyperlinkURI, "OK"), tEOF,
		},
	},
	{
		"duplicate external targets, different URIs",
		`Duplicate external targets (different URIs):

.. _target: first

.. _target: second`,
		[]Token{
			mkItem(Text, "Duplicate external targets (different URIs):"), tBlankLine,
			tHyperlinkStart, tSpace, tHyperlinkPrefix, mkItem(HyperlinkName, "target"), tHyperlinkSuffix,
			tSpace, mkItem(HyperlinkURI, "first"), tBlankLine,
			tHyperlinkStart, tSpace, tHyperlinkPrefix, mkItem(HyperlinkName, "target"), tHyperlinkSuffix,
			tSpace, mkItem(HyperlinkURI, "second"), tEOF,
		},
	},
	{
		"duplicate external targets, same URIs",
		`Duplicate external targets (same URIs):

.. _target: first

.. _target: first`,
		[]Token{
			mkItem(Text, "Duplicate external targets (same URIs):"), tBlankLine,
			tHyperlinkStart, tSpace, tHyperlinkPrefix, mkItem(HyperlinkName, "target"), tHyperlinkSuffix,
			tSpace, mkItem(HyperlinkURI, "first"), tBlankLine,
			tHyperlinkStart, tSpace, tHyperlinkPrefix, mkItem(HyperlinkName, "target"), tHyperlinkSuffix,
			tSpace, mkItem(HyperlinkURI, "first"), tEOF,
		},
	},
	{
		"duplicate implicit targets",
		`Duplicate implicit targets.

Title
=====

Paragraph.

Title
=====

Paragraph.`,
		[]Token{
			mkItem(Text, "Duplicate implicit targets."), tBlankLine,
			mkItem(Text, "Title"), mkItem(Text, "====="), // TODO: Should be Title once implemented
			tBlankLine, mkItem(Text, "Paragraph."), tBlankLine,
			mkItem(Text, "Title"), mkItem(Text, "====="),
			tBlankLine, mkItem(Text, "Paragraph."), tEOF,
		},
	},
	{
		"duplicate implicit/explicit targets",
		`Duplicate implicit/explicit targets.

Title
=====

.. _title:

Paragraph.`,
		[]Token{
			mkItem(Text, "Duplicate implicit/explicit targets."), tBlankLine,
			mkItem(Text, "Title"), mkItem(Text, "====="), // TODO: Should be Title once implemented
			tBlankLine, tHyperlinkStart, tSpace, tHyperlinkPrefix, mkItem(HyperlinkName, "title"), tHyperlinkSuffix,
			tBlankLine, mkItem(Text, "Paragraph."), tEOF,
		},
	},
	{
		"duplicate implicit/directive targets",
		`Duplicate implicit/directive targets.

Title
=====

.. target-notes::
   :name: title`,
		[]Token{
			mkItem(Text, "Duplicate implicit/directive targets."), tBlankLine,
			mkItem(Text, "Title"), mkItem(Text, "====="), // TODO: Should be Title once implemented
			tBlankLine, tComment, tSpace, mkItem(Text, "target-notes::"), // TODO: Should be Directive once implemented
			tSpace3, mkItem(Text, ":name: title"), tEOF,
		},
	},
	{
		"duplicate explicit targets",
		`Duplicate explicit targets.

.. _title:

First.

.. _title:

Second.

.. _title:

Third.`,
		[]Token{
			mkItem(Text, "Duplicate explicit targets."), tBlankLine,
			tHyperlinkStart, tSpace, tHyperlinkPrefix, mkItem(HyperlinkName, "title"), tHyperlinkSuffix,
			tBlankLine, mkItem(Text, "First."), tBlankLine,
			tHyperlinkStart, tSpace, tHyperlinkPrefix, mkItem(HyperlinkName, "title"), tHyperlinkSuffix,
			tBlankLine, mkItem(Text, "Second."), tBlankLine,
			tHyperlinkStart, tSpace, tHyperlinkPrefix, mkItem(HyperlinkName, "title"), tHyperlinkSuffix,
			tBlankLine, mkItem(Text, "Third."), tEOF,
		},
	},
	{
		"duplicate explicit/directive targets",
		`Duplicate explicit/directive targets.

.. _title:

First.

.. rubric:: this is a title too
   :name: title

`,
		[]Token{
			mkItem(Text, "Duplicate explicit/directive targets."), tBlankLine,
			tHyperlinkStart, tSpace, tHyperlinkPrefix, mkItem(HyperlinkName, "title"), tHyperlinkSuffix,
			tBlankLine, mkItem(Text, "First."), tBlankLine,
			tComment, tSpace, mkItem(Text, "rubric:: this is a title too"), // TODO: Should be Directive once implemented
			tSpace3, mkItem(Text, ":name: title"), tBlankLine, tEOF,
		},
	},
	{
		"duplicate targets",
		`Duplicate targets:

Target
======

Implicit section header target.

.. [TARGET] Citation target.

.. [#target] Autonumber-labeled footnote target.

.. _target:

Explicit internal target.

.. _target: Explicit_external_target

.. rubric:: directive with target
   :name: Target`,
		[]Token{
			mkItem(Text, "Duplicate targets:"), tBlankLine,
			mkItem(Text, "Target"), mkItem(Text, "======"), // TODO: Should be Title once implemented
			tBlankLine, mkItem(Text, "Implicit section header target."), tBlankLine,
			tComment, tSpace, mkItem(Text, "[TARGET] Citation target."), // TODO: Should be Citation once implemented
			tBlankLine, tComment, tSpace, mkItem(Text, "[#target] Autonumber-labeled footnote target."), // TODO: Should be Footnote once implemented
			tBlankLine, tHyperlinkStart, tSpace, tHyperlinkPrefix, mkItem(HyperlinkName, "target"), tHyperlinkSuffix,
			tBlankLine, mkItem(Text, "Explicit internal target."), tBlankLine,
			tHyperlinkStart, tSpace, tHyperlinkPrefix, mkItem(HyperlinkName, "target"), tHyperlinkSuffix,
			tSpace, mkItem(HyperlinkURI, "Explicit_external_target"), tBlankLine,
			tComment, tSpace, mkItem(Text, "rubric:: directive with target"), // TODO: Should be Directive once implemented
			tSpace3, mkItem(Text, ":name: Target"), tEOF,
		},
	},
	{
		"colon escapes",
		`.. _unescaped colon at end:: no good

.. _:: no good either

.. _escaped colon\:: OK

` + ".. _`unescaped colon, quoted: `: OK",
		[]Token{
			tHyperlinkStart, tSpace, tHyperlinkPrefix, mkItem(HyperlinkName, "unescaped colon at end"), tHyperlinkSuffix,
			mkItem(Text, ": no good"), tBlankLine,
			tHyperlinkStart, tSpace, tHyperlinkPrefix, mkItem(HyperlinkName, ":"), tHyperlinkSuffix,
			tSpace, mkItem(HyperlinkURI, "no good either"), tBlankLine,
			tHyperlinkStart, tSpace, tHyperlinkPrefix, mkItem(HyperlinkName, `escaped colon\:`), tHyperlinkSuffix,
			tSpace, mkItem(HyperlinkURI, "OK"), tBlankLine,
			tHyperlinkStart, tSpace, tHyperlinkPrefix, tHyperlinkQuote, mkItem(HyperlinkName, "unescaped colon, quoted: "),
			tHyperlinkQuote, tHyperlinkSuffix, tSpace, mkItem(HyperlinkURI, "OK"), tEOF,
		},
	},
	// anonymous targets
	{
		"anonymous external hyperlink target",
		`Anonymous external hyperlink target:

.. __: http://w3c.org/`,
		[]Token{
			mkItem(Text, "Anonymous external hyperlink target:"), tBlankLine,
			tHyperlinkStart, tSpace, tAnonHyperlinkPrefix, tHyperlinkSuffix,
			tSpace, mkItem(HyperlinkURI, "http://w3c.org/"), tEOF,
		},
	},
	{
		"anonymous external hyperlink target, alternative syntax",
		`Anonymous external hyperlink target:

__ http://w3c.org/`,
		[]Token{
			mkItem(Text, "Anonymous external hyperlink target:"), tBlankLine,
			tAnonHyperlinkStart, tSpace, mkItem(HyperlinkURI, "http://w3c.org/"), tEOF,
		},
	},
	{
		"anonymous indirect hyperlink target",
		`Anonymous indirect hyperlink target:

.. __: reference_`,
		[]Token{
			mkItem(Text, "Anonymous indirect hyperlink target:"), tBlankLine,
			tHyperlinkStart, tSpace, tAnonHyperlinkPrefix, tHyperlinkSuffix,
			tSpace, mkItem(InlineReferenceText, "reference"), tInlineReferenceClose1, tEOF,
		},
	},
	{
		"anonymous external hyperlink targets",
		`Anonymous external hyperlink target, not indirect:

__ uri\_

__ this URI ends with an underscore_`,
		[]Token{
			mkItem(Text, "Anonymous external hyperlink target, not indirect:"), tBlankLine,
			tAnonHyperlinkStart, tSpace, mkItem(HyperlinkURI, `uri\_`), tBlankLine,
			tAnonHyperlinkStart, tSpace, mkItem(HyperlinkURI, "this URI ends with an underscore_"), tEOF,
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
