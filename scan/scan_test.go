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

func item(typ Type, text string) Token {
	return Token{Type: typ, Text: text}
}

var (
	tEOF                   = item(EOF, "EOF")
	tBlankLine             = item(BlankLine, "\n")
	tSpace                 = item(Space, " ")
	tSpace2                = item(Space, "  ")
	tSpace3                = item(Space, "   ")
	tComment               = item(Comment, "..")
	tHyperlinkStart        = item(HyperlinkStart, "..")
	tAnonHyperlinkStart    = item(HyperlinkStart, "__")
	tHyperlinkPrefix       = item(HyperlinkPrefix, "_")
	tAnonHyperlinkPrefix   = item(HyperlinkPrefix, "__")
	tHyperlinkQuote        = item(HyperlinkQuote, "`")
	tHyperlinkSuffix       = item(HyperlinkSuffix, ":")
	tInlineReferenceOpen   = item(InlineReferenceOpen, "`")
	tInlineReferenceClose1 = item(InlineReferenceClose, "_")
	tInlineReferenceClose2 = item(InlineReferenceClose, "`_")
)

var scanTests = []scanTest{
	{"empty", "", []Token{tEOF}},
	{"spaces", " \t\n", []Token{item(Space, " \t\n"), tEOF}},
	{"quote error", "`", []Token{item(Error, "expected hyperlink or inline reference before quote")}},
	{"text", `now is the time`, []Token{item(Text, "now is the time"), tEOF}},
	// comments
	{
		"line comment",
		`.. A comment

Paragraph.`,
		[]Token{tComment, tSpace, item(Text, "A comment"), tBlankLine, item(Text, "Paragraph."), tEOF},
	},
	{
		"comment block",
		`.. A comment
   block.

Paragraph.`,
		[]Token{
			tComment, tSpace, item(Text, "A comment"), tSpace3, item(Text, "block."),
			tBlankLine, item(Text, "Paragraph."), tEOF,
		},
	},
	{
		"multi-line comment block",
		`..
   A comment consisting of multiple lines
   starting on the line after the
   explicit markup start.`,
		[]Token{
			tComment, tSpace3, item(Text, "A comment consisting of multiple lines"),
			tSpace3, item(Text, "starting on the line after the"),
			tSpace3, item(Text, "explicit markup start."), tEOF,
		},
	},
	{
		"2 line comments",
		`.. A comment.
.. Another.

Paragraph.`,
		[]Token{
			tComment, tSpace, item(Text, "A comment."),
			tComment, tSpace, item(Text, "Another."),
			tBlankLine, item(Text, "Paragraph."), tEOF,
		},
	},
	{
		"line comment, no blank line",
		`.. A comment
no blank line

Paragraph.`,
		[]Token{
			tComment, tSpace, item(Text, "A comment"), item(Text, "no blank line"),
			tBlankLine, item(Text, "Paragraph."), tEOF,
		},
	},
	{
		"2 line comments, no blank line",
		`.. A comment.
.. Another.
no blank line

Paragraph.`,
		[]Token{
			tComment, tSpace, item(Text, "A comment."),
			tComment, tSpace, item(Text, "Another."), item(Text, "no blank line"),
			tBlankLine, item(Text, "Paragraph."), tEOF,
		},
	},
	{
		"line comment with directive",
		`.. A comment::

Paragraph.`,
		[]Token{tComment, tSpace, item(Text, "A comment::"), tBlankLine, item(Text, "Paragraph."), tEOF},
	},
	{
		"comment block with directive",
		`..
   comment::

The extra newline before the comment text prevents
the parser from recognizing a directive.`,
		[]Token{
			tComment, tSpace3, item(Text, "comment::"), tBlankLine,
			item(Text, "The extra newline before the comment text prevents"),
			item(Text, "the parser from recognizing a directive."), tEOF,
		},
	},
	{
		"comment block with hyperlink target",
		`..
   _comment: http://example.org

The extra newline before the comment text prevents
the parser from recognizing a hyperlink target.`,
		[]Token{
			tComment, tSpace3, item(Text, "_comment: http://example.org"), tBlankLine,
			item(Text, "The extra newline before the comment text prevents"),
			item(Text, "the parser from recognizing a hyperlink target."), tEOF,
		},
	},
	{
		"comment block with citation",
		`..
   [comment] Not a citation.

The extra newline before the comment text prevents
the parser from recognizing a citation.`,
		[]Token{
			tComment, tSpace3, item(Text, "[comment] Not a citation."), tBlankLine,
			item(Text, "The extra newline before the comment text prevents"),
			item(Text, "the parser from recognizing a citation."), tEOF,
		},
	},
	{
		"comment block with substitution definition",
		`..
   |comment| image:: bogus.png

The extra newline before the comment text prevents
the parser from recognizing a substitution definition.`,
		[]Token{
			tComment, tSpace3, item(Text, "|comment| image:: bogus.png"), tBlankLine,
			item(Text, "The extra newline before the comment text prevents"),
			item(Text, "the parser from recognizing a substitution definition."), tEOF,
		},
	},
	{
		"comment block and empty comment",
		`.. Next is an empty comment, which serves to end this comment and
   prevents the following block quote being swallowed up.

..

    A block quote.`,
		[]Token{
			tComment, tSpace, item(Text, "Next is an empty comment, which serves to end this comment and"),
			tSpace3, item(Text, "prevents the following block quote being swallowed up."),
			tBlankLine, tComment, tBlankLine, item(Space, "    "),
			item(Text, "A block quote."), // TODO: Should be BlockQuote once implemented
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
			item(Text, "term 1"), // TODO: Should be DefinitionTerm once implemented
			tSpace2, item(Text, "definition 1"), tBlankLine,
			tSpace2, tComment, tSpace, item(Text, "a comment"), tBlankLine,
			item(Text, "term 2"), tSpace2, item(Text, "definition 2"), tEOF,
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
			item(Text, "term 1"), // TODO: Should be DefinitionTerm once implemented
			tSpace2, item(Text, "definition 1"), tBlankLine,
			tComment, tSpace, item(Text, "a comment"), tBlankLine,
			item(Text, "term 2"), tSpace2, item(Text, "definition 2"), tEOF,
		},
	},
	{
		"comment between bullet paragraphs 2 and 3",
		`+ bullet paragraph 1

  bullet paragraph 2

  .. comment between bullet paragraphs 2 and 3

  bullet paragraph 3`,
		[]Token{
			item(Text, "+ bullet paragraph 1"), // TODO: Should be Bullet once implemented
			tBlankLine, tSpace2, item(Text, "bullet paragraph 2"), tBlankLine,
			tSpace2, tComment, tSpace, item(Text, "comment between bullet paragraphs 2 and 3"),
			tBlankLine, tSpace2, item(Text, "bullet paragraph 3"), tEOF,
		},
	},
	{
		"comment between bullet paragraphs 1 and 2",
		`+ bullet paragraph 1

  .. comment between bullet paragraphs 1 (leader) and 2

  bullet paragraph 2`,
		[]Token{
			item(Text, "+ bullet paragraph 1"), // TODO: Should be Bullet once implemented
			tBlankLine, tSpace2,
			tComment, tSpace, item(Text, "comment between bullet paragraphs 1 (leader) and 2"),
			tBlankLine, tSpace2, item(Text, "bullet paragraph 2"), tEOF,
		},
	},
	{
		"comment trailing bullet paragraph",
		`+ bullet

  .. trailing comment`,
		[]Token{
			item(Text, "+ bullet"), // TODO: Should be Bullet once implemented
			tBlankLine, tSpace2, tComment, tSpace, item(Text, "trailing comment"), tEOF,
		},
	},
	{
		"comment, not target", ".. _",
		[]Token{tComment, tSpace, item(Text, "_"), tEOF},
	},
	// targets
	{
		"hyperlink target",
		`.. _target:

(Internal hyperlink target.)`,
		[]Token{
			tHyperlinkStart, tSpace, tHyperlinkPrefix,
			item(HyperlinkName, "target"), tHyperlinkSuffix,
			tBlankLine, item(Text, "(Internal hyperlink target.)"), tEOF,
		},
	},
	{
		"hyperlink target with optional space before colon", ".. _optional space before colon :",
		[]Token{
			tHyperlinkStart, tSpace, tHyperlinkPrefix, item(HyperlinkName, "optional space before colon "),
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
			item(Text, "External hyperlink targets:"), tBlankLine,
			tHyperlinkStart, tSpace, tHyperlinkPrefix, item(HyperlinkName, "one-liner"), tHyperlinkSuffix,
			tSpace, item(HyperlinkURI, "http://structuredtext.sourceforge.net"), tBlankLine,
			tHyperlinkStart, tSpace, tHyperlinkPrefix, item(HyperlinkName, "starts-on-this-line"), tHyperlinkSuffix,
			tSpace, item(HyperlinkURI, "http://"), item(Space, "                         "), item(HyperlinkURI, "structuredtext."),
			item(Space, "                         "), item(HyperlinkURI, "sourceforge.net"), tBlankLine,
			tHyperlinkStart, tSpace, tHyperlinkPrefix, item(HyperlinkName, "entirely-below"), tHyperlinkSuffix,
			tSpace3, item(HyperlinkURI, "http://structuredtext."), tSpace3, item(HyperlinkURI, "sourceforge.net"), tBlankLine,
			tHyperlinkStart, tSpace, tHyperlinkPrefix, item(HyperlinkName, "escaped-whitespace"), tHyperlinkSuffix,
			tSpace, item(HyperlinkURI, `http://example.org/a\ path\ with\`), tSpace3, item(HyperlinkURI, "spaces.html"), tBlankLine,
			tHyperlinkStart, tSpace, tHyperlinkPrefix, item(HyperlinkName, "not-indirect"), tHyperlinkSuffix,
			tSpace, item(HyperlinkURI, `uri\_`), tEOF,
		},
	},
	{
		"indirect hyperlink targets",
		`Indirect hyperlink targets:

.. _target1: reference_

` + ".. _target2: `phrase-link reference`_",
		[]Token{
			item(Text, "Indirect hyperlink targets:"), tBlankLine,
			tHyperlinkStart, tSpace, tHyperlinkPrefix, item(HyperlinkName, "target1"), tHyperlinkSuffix,
			tSpace, item(InlineReferenceText, "reference"), tInlineReferenceClose1, tBlankLine,
			tHyperlinkStart, tSpace, tHyperlinkPrefix, item(HyperlinkName, "target2"), tHyperlinkSuffix,
			tSpace, tInlineReferenceOpen, item(InlineReferenceText, "phrase-link reference"), tInlineReferenceClose2,
			tEOF,
		},
	},
	{
		"escaped hyperlink target names",
		`.. _a long target name:

` + ".. _`a target name: including a colon (quoted)`:" + `

.. _a target name\: including a colon (escaped):`,
		[]Token{
			tHyperlinkStart, tSpace, tHyperlinkPrefix, item(HyperlinkName, "a long target name"), tHyperlinkSuffix, tBlankLine,
			tHyperlinkStart, tSpace, tHyperlinkPrefix, tHyperlinkQuote, item(HyperlinkName, "a target name: including a colon (quoted)"),
			tHyperlinkQuote, tHyperlinkSuffix, tBlankLine,
			tHyperlinkStart, tSpace, tHyperlinkPrefix, item(HyperlinkName, `a target name\: including a colon (escaped)`), tHyperlinkSuffix,
			tEOF,
		},
	},
	{
		"hyperlink target names with no matching backquotes",
		".. _`target: No matching backquote.\n.. _`: No matching backquote either.",
		[]Token{
			tHyperlinkStart, tSpace, tHyperlinkPrefix, tHyperlinkQuote, item(HyperlinkName, "target: No matching backquote."),
			tHyperlinkStart, tSpace, tHyperlinkPrefix, tHyperlinkQuote, item(HyperlinkName, ": No matching backquote either."), tEOF,
		},
	},
	{
		"hyperlink target names split across lines, 1 regular, 1 backquoted",
		`.. _a very long target name,
   split across lines:
` + ".. _`and another,\n   with backquotes`:",
		[]Token{
			tHyperlinkStart, tSpace, tHyperlinkPrefix, item(HyperlinkName, "a very long target name,"),
			tSpace3, item(HyperlinkName, "split across lines"), tHyperlinkSuffix,
			tHyperlinkStart, tSpace, tHyperlinkPrefix, tHyperlinkQuote, item(HyperlinkName, "and another,"),
			tSpace3, item(HyperlinkName, "with backquotes"), tHyperlinkQuote, tHyperlinkSuffix, tEOF,
		},
	},
	{
		"external hyperlink target",
		`External hyperlink:

.. _target: http://www.python.org/`,
		[]Token{
			item(Text, "External hyperlink:"), tBlankLine,
			tHyperlinkStart, tSpace, tHyperlinkPrefix, item(HyperlinkName, "target"), tHyperlinkSuffix,
			tSpace, item(HyperlinkURI, "http://www.python.org/"), tEOF,
		},
	},
	{
		"email targets",
		`.. _email: jdoe@example.com

.. _multi-line email: jdoe
   @example.com`,
		[]Token{
			tHyperlinkStart, tSpace, tHyperlinkPrefix, item(HyperlinkName, "email"), tHyperlinkSuffix,
			tSpace, item(HyperlinkURI, "jdoe@example.com"), tBlankLine,
			tHyperlinkStart, tSpace, tHyperlinkPrefix, item(HyperlinkName, "multi-line email"), tHyperlinkSuffix,
			tSpace, item(HyperlinkURI, "jdoe"), tSpace3, item(HyperlinkURI, "@example.com"), tEOF,
		},
	},
	{
		"malformed target",
		`Malformed target:

.. __malformed: no good

Target beginning with an underscore:

` + ".. _`_target`: OK",
		[]Token{
			item(Text, "Malformed target:"), tBlankLine,
			tHyperlinkStart, tSpace, tHyperlinkPrefix, item(HyperlinkName, "_malformed"), tHyperlinkSuffix,
			tSpace, item(HyperlinkURI, "no good"), tBlankLine,
			item(Text, "Target beginning with an underscore:"), tBlankLine,
			tHyperlinkStart, tSpace, tHyperlinkPrefix, tHyperlinkQuote, item(HyperlinkName, "_target"), tHyperlinkQuote, tHyperlinkSuffix,
			tSpace, item(HyperlinkURI, "OK"), tEOF,
		},
	},
	{
		"duplicate external targets, different URIs",
		`Duplicate external targets (different URIs):

.. _target: first

.. _target: second`,
		[]Token{
			item(Text, "Duplicate external targets (different URIs):"), tBlankLine,
			tHyperlinkStart, tSpace, tHyperlinkPrefix, item(HyperlinkName, "target"), tHyperlinkSuffix,
			tSpace, item(HyperlinkURI, "first"), tBlankLine,
			tHyperlinkStart, tSpace, tHyperlinkPrefix, item(HyperlinkName, "target"), tHyperlinkSuffix,
			tSpace, item(HyperlinkURI, "second"), tEOF,
		},
	},
	{
		"duplicate external targets, same URIs",
		`Duplicate external targets (same URIs):

.. _target: first

.. _target: first`,
		[]Token{
			item(Text, "Duplicate external targets (same URIs):"), tBlankLine,
			tHyperlinkStart, tSpace, tHyperlinkPrefix, item(HyperlinkName, "target"), tHyperlinkSuffix,
			tSpace, item(HyperlinkURI, "first"), tBlankLine,
			tHyperlinkStart, tSpace, tHyperlinkPrefix, item(HyperlinkName, "target"), tHyperlinkSuffix,
			tSpace, item(HyperlinkURI, "first"), tEOF,
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
			item(Text, "Duplicate implicit targets."), tBlankLine,
			item(Text, "Title"), item(Text, "====="), // TODO: Should be Title once implemented
			tBlankLine, item(Text, "Paragraph."), tBlankLine,
			item(Text, "Title"), item(Text, "====="),
			tBlankLine, item(Text, "Paragraph."), tEOF,
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
			item(Text, "Duplicate implicit/explicit targets."), tBlankLine,
			item(Text, "Title"), item(Text, "====="), // TODO: Should be Title once implemented
			tBlankLine, tHyperlinkStart, tSpace, tHyperlinkPrefix, item(HyperlinkName, "title"), tHyperlinkSuffix,
			tBlankLine, item(Text, "Paragraph."), tEOF,
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
			item(Text, "Duplicate implicit/directive targets."), tBlankLine,
			item(Text, "Title"), item(Text, "====="), // TODO: Should be Title once implemented
			tBlankLine, tComment, tSpace, item(Text, "target-notes::"), // TODO: Should be Directive once implemented
			tSpace3, item(Text, ":name: title"), tEOF,
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
			item(Text, "Duplicate explicit targets."), tBlankLine,
			tHyperlinkStart, tSpace, tHyperlinkPrefix, item(HyperlinkName, "title"), tHyperlinkSuffix,
			tBlankLine, item(Text, "First."), tBlankLine,
			tHyperlinkStart, tSpace, tHyperlinkPrefix, item(HyperlinkName, "title"), tHyperlinkSuffix,
			tBlankLine, item(Text, "Second."), tBlankLine,
			tHyperlinkStart, tSpace, tHyperlinkPrefix, item(HyperlinkName, "title"), tHyperlinkSuffix,
			tBlankLine, item(Text, "Third."), tEOF,
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
			item(Text, "Duplicate explicit/directive targets."), tBlankLine,
			tHyperlinkStart, tSpace, tHyperlinkPrefix, item(HyperlinkName, "title"), tHyperlinkSuffix,
			tBlankLine, item(Text, "First."), tBlankLine,
			tComment, tSpace, item(Text, "rubric:: this is a title too"), // TODO: Should be Directive once implemented
			tSpace3, item(Text, ":name: title"), tBlankLine, tEOF,
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
			item(Text, "Duplicate targets:"), tBlankLine,
			item(Text, "Target"), item(Text, "======"), // TODO: Should be Title once implemented
			tBlankLine, item(Text, "Implicit section header target."), tBlankLine,
			tComment, tSpace, item(Text, "[TARGET] Citation target."), // TODO: Should be Citation once implemented
			tBlankLine, tComment, tSpace, item(Text, "[#target] Autonumber-labeled footnote target."), // TODO: Should be Footnote once implemented
			tBlankLine, tHyperlinkStart, tSpace, tHyperlinkPrefix, item(HyperlinkName, "target"), tHyperlinkSuffix,
			tBlankLine, item(Text, "Explicit internal target."), tBlankLine,
			tHyperlinkStart, tSpace, tHyperlinkPrefix, item(HyperlinkName, "target"), tHyperlinkSuffix,
			tSpace, item(HyperlinkURI, "Explicit_external_target"), tBlankLine,
			tComment, tSpace, item(Text, "rubric:: directive with target"), // TODO: Should be Directive once implemented
			tSpace3, item(Text, ":name: Target"), tEOF,
		},
	},
	{
		"colon escapes",
		`.. _unescaped colon at end:: no good

.. _:: no good either

.. _escaped colon\:: OK

` + ".. _`unescaped colon, quoted: `: OK",
		[]Token{
			tHyperlinkStart, tSpace, tHyperlinkPrefix, item(HyperlinkName, "unescaped colon at end"), tHyperlinkSuffix,
			item(Text, ": no good"), tBlankLine,
			tHyperlinkStart, tSpace, tHyperlinkPrefix, item(HyperlinkName, ":"), tHyperlinkSuffix,
			tSpace, item(HyperlinkURI, "no good either"), tBlankLine,
			tHyperlinkStart, tSpace, tHyperlinkPrefix, item(HyperlinkName, `escaped colon\:`), tHyperlinkSuffix,
			tSpace, item(HyperlinkURI, "OK"), tBlankLine,
			tHyperlinkStart, tSpace, tHyperlinkPrefix, tHyperlinkQuote, item(HyperlinkName, "unescaped colon, quoted: "),
			tHyperlinkQuote, tHyperlinkSuffix, tSpace, item(HyperlinkURI, "OK"), tEOF,
		},
	},
	// anonymous targets
	{
		"anonymous external hyperlink target",
		`Anonymous external hyperlink target:

.. __: http://w3c.org/`,
		[]Token{
			item(Text, "Anonymous external hyperlink target:"), tBlankLine,
			tHyperlinkStart, tSpace, tAnonHyperlinkPrefix, tHyperlinkSuffix,
			tSpace, item(HyperlinkURI, "http://w3c.org/"), tEOF,
		},
	},
	{
		"anonymous external hyperlink target, alternative syntax",
		`Anonymous external hyperlink target:

__ http://w3c.org/`,
		[]Token{
			item(Text, "Anonymous external hyperlink target:"), tBlankLine,
			tAnonHyperlinkStart, tSpace, item(HyperlinkURI, "http://w3c.org/"), tEOF,
		},
	},
	{
		"anonymous indirect hyperlink target",
		`Anonymous indirect hyperlink target:

.. __: reference_`,
		[]Token{
			item(Text, "Anonymous indirect hyperlink target:"), tBlankLine,
			tHyperlinkStart, tSpace, tAnonHyperlinkPrefix, tHyperlinkSuffix, tSpace,
			item(InlineReferenceText, "reference"), tInlineReferenceClose1, tEOF,
		},
	},
	{
		"anonymous external hyperlink targets",
		`Anonymous external hyperlink target, not indirect:

__ uri\_

__ this URI ends with an underscore_`,
		[]Token{
			item(Text, "Anonymous external hyperlink target, not indirect:"), tBlankLine,
			tAnonHyperlinkStart, tSpace, item(HyperlinkURI, `uri\_`), tBlankLine,
			tAnonHyperlinkStart, tSpace, item(HyperlinkURI, "this URI ends with an underscore_"), tEOF,
		},
	},
	{
		"anonymous indirect hyperlink targets",
		`Anonymous indirect hyperlink targets:

__ reference_
` + "__ `a very long\n   reference`_",
		[]Token{
			item(Text, "Anonymous indirect hyperlink targets:"), tBlankLine,
			tAnonHyperlinkStart, tSpace, item(InlineReferenceText, "reference"), tInlineReferenceClose1,
			tAnonHyperlinkStart, tSpace, tInlineReferenceOpen, item(InlineReferenceText, "a very long"),
			tSpace3, item(InlineReferenceText, "reference"), tInlineReferenceClose2, tEOF,
		},
	},
	{
		"mixed anonymous/named indirect hyperlink targets",
		`Mixed anonymous & named indirect hyperlink targets:

__ reference_
.. __: reference_
__ reference_
.. _target1: reference_
no blank line

.. _target2: reference_
__ reference_
.. __: reference_
__ reference_
no blank line`,
		[]Token{
			item(Text, "Mixed anonymous & named indirect hyperlink targets:"), tBlankLine,
			tAnonHyperlinkStart, tSpace, item(InlineReferenceText, "reference"), tInlineReferenceClose1,
			tHyperlinkStart, tSpace, tAnonHyperlinkPrefix, tHyperlinkSuffix, tSpace,
			item(InlineReferenceText, "reference"), tInlineReferenceClose1,
			tAnonHyperlinkStart, tSpace, item(InlineReferenceText, "reference"), tInlineReferenceClose1,
			tHyperlinkStart, tSpace, tHyperlinkPrefix, item(HyperlinkName, "target1"), tHyperlinkSuffix,
			tSpace, item(InlineReferenceText, "reference"), tInlineReferenceClose1,
			item(Text, "no blank line"), tBlankLine,
			tHyperlinkStart, tSpace, tHyperlinkPrefix, item(HyperlinkName, "target2"), tHyperlinkSuffix,
			tSpace, item(InlineReferenceText, "reference"), tInlineReferenceClose1,
			tAnonHyperlinkStart, tSpace, item(InlineReferenceText, "reference"), tInlineReferenceClose1,
			tHyperlinkStart, tSpace, tAnonHyperlinkPrefix, tHyperlinkSuffix, tSpace,
			item(InlineReferenceText, "reference"), tInlineReferenceClose1,
			tAnonHyperlinkStart, tSpace, item(InlineReferenceText, "reference"), tInlineReferenceClose1,
			item(Text, "no blank line"), tEOF,
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
