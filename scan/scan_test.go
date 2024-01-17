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
	tSpace4                = item(Space, "    ")
	tSectionAdornment2     = item(SectionAdornment, "==")
	tSectionAdornment3     = item(SectionAdornment, "===")
	tSectionAdornment5     = item(SectionAdornment, "=====")
	tSectionAdornment7     = item(SectionAdornment, "=======")
	tSectionAdornment24    = item(SectionAdornment, "========================")
	tSectionAdornmentDash3 = item(SectionAdornment, "---")
	tSectionAdornmentDash5 = item(SectionAdornment, "-----")
	tSectionAdornmentDash7 = item(SectionAdornment, "-------")
	tSectionAdornmentDot3  = item(SectionAdornment, "...")
	tSectionAdornmentTick7 = item(SectionAdornment, "```````")
	tTransition24          = item(Transition, "========================")
	tTransitionDash8       = item(Transition, "--------")
	tBulletAsterisk        = item(Bullet, "*")
	tBulletPlus            = item(Bullet, "+")
	tBulletDash            = item(Bullet, "-")
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
	{"text", `now is the time`, []Token{item(Paragraph, "now is the time"), tEOF}},
	// comments
	{
		"comment",
		`.. A comment

Paragraph.`,
		[]Token{tComment, tSpace, item(Paragraph, "A comment"), tBlankLine, item(Paragraph, "Paragraph."), tEOF},
	},
	{
		"comment block",
		`.. A comment
   block.

Paragraph.`,
		[]Token{
			tComment, tSpace, item(Paragraph, "A comment"), tSpace3, item(Paragraph, "block."),
			tBlankLine, item(Paragraph, "Paragraph."), tEOF,
		},
	},
	{
		"multi-line comment block",
		`..
   A comment consisting of multiple lines
   starting on the line after the
   explicit markup start.`,
		[]Token{
			tComment, tSpace3, item(Paragraph, "A comment consisting of multiple lines"),
			tSpace3, item(Paragraph, "starting on the line after the"),
			tSpace3, item(Paragraph, "explicit markup start."), tEOF,
		},
	},
	{
		"2 comments",
		`.. A comment.
.. Another.

Paragraph.`,
		[]Token{
			tComment, tSpace, item(Paragraph, "A comment."),
			tComment, tSpace, item(Paragraph, "Another."),
			tBlankLine, item(Paragraph, "Paragraph."), tEOF,
		},
	},
	{
		"comment, no blank line",
		`.. A comment
no blank line

Paragraph.`,
		[]Token{
			tComment, tSpace, item(Paragraph, "A comment"), item(Paragraph, "no blank line"),
			tBlankLine, item(Paragraph, "Paragraph."), tEOF,
		},
	},
	{
		"2 comments, no blank line",
		`.. A comment.
.. Another.
no blank line

Paragraph.`,
		[]Token{
			tComment, tSpace, item(Paragraph, "A comment."),
			tComment, tSpace, item(Paragraph, "Another."), item(Paragraph, "no blank line"),
			tBlankLine, item(Paragraph, "Paragraph."), tEOF,
		},
	},
	{
		"comment with directive",
		`.. A comment::

Paragraph.`,
		[]Token{tComment, tSpace, item(Paragraph, "A comment::"), tBlankLine, item(Paragraph, "Paragraph."), tEOF},
	},
	{
		"comment block with directive",
		`..
   comment::

The extra newline before the comment text prevents
the parser from recognizing a directive.`,
		[]Token{
			tComment, tSpace3, item(Paragraph, "comment::"), tBlankLine,
			item(Paragraph, "The extra newline before the comment text prevents"),
			item(Paragraph, "the parser from recognizing a directive."), tEOF,
		},
	},
	{
		"comment block with hyperlink target",
		`..
   _comment: http://example.org

The extra newline before the comment text prevents
the parser from recognizing a hyperlink target.`,
		[]Token{
			tComment, tSpace3, item(Paragraph, "_comment: http://example.org"), tBlankLine,
			item(Paragraph, "The extra newline before the comment text prevents"),
			item(Paragraph, "the parser from recognizing a hyperlink target."), tEOF,
		},
	},
	{
		"comment block with citation",
		`..
   [comment] Not a citation.

The extra newline before the comment text prevents
the parser from recognizing a citation.`,
		[]Token{
			tComment, tSpace3, item(Paragraph, "[comment] Not a citation."), tBlankLine,
			item(Paragraph, "The extra newline before the comment text prevents"),
			item(Paragraph, "the parser from recognizing a citation."), tEOF,
		},
	},
	{
		"comment block with substitution definition",
		`..
   |comment| image:: bogus.png

The extra newline before the comment text prevents
the parser from recognizing a substitution definition.`,
		[]Token{
			tComment, tSpace3, item(Paragraph, "|comment| image:: bogus.png"), tBlankLine,
			item(Paragraph, "The extra newline before the comment text prevents"),
			item(Paragraph, "the parser from recognizing a substitution definition."), tEOF,
		},
	},
	{
		"comment block and empty comment",
		`.. Next is an empty comment, which serves to end this comment and
   prevents the following block quote being swallowed up.

..

    A block quote.`,
		[]Token{
			tComment, tSpace, item(Paragraph, "Next is an empty comment, which serves to end this comment and"),
			tSpace3, item(Paragraph, "prevents the following block quote being swallowed up."),
			tBlankLine, tComment, tBlankLine, tSpace4,
			item(Paragraph, "A block quote."), // TODO: Should be BlockQuote once implemented
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
			item(Paragraph, "term 1"), // TODO: Should be DefinitionTerm once implemented
			tSpace2, item(Paragraph, "definition 1"), tBlankLine,
			tSpace2, tComment, tSpace, item(Paragraph, "a comment"), tBlankLine,
			item(Paragraph, "term 2"), tSpace2, item(Paragraph, "definition 2"), tEOF,
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
			item(Paragraph, "term 1"), // TODO: Should be DefinitionTerm once implemented
			tSpace2, item(Paragraph, "definition 1"), tBlankLine,
			tComment, tSpace, item(Paragraph, "a comment"), tBlankLine,
			item(Paragraph, "term 2"), tSpace2, item(Paragraph, "definition 2"), tEOF,
		},
	},
	{
		"comment between bullet paragraphs 2 and 3",
		`+ bullet paragraph 1

  bullet paragraph 2

  .. comment between bullet paragraphs 2 and 3

  bullet paragraph 3`,
		[]Token{
			tBulletPlus, tSpace, item(Paragraph, "bullet paragraph 1"), tBlankLine,
			tSpace2, item(Paragraph, "bullet paragraph 2"), tBlankLine,
			tSpace2, tComment, tSpace, item(Paragraph, "comment between bullet paragraphs 2 and 3"),
			tBlankLine, tSpace2, item(Paragraph, "bullet paragraph 3"), tEOF,
		},
	},
	{
		"comment between bullet paragraphs 1 and 2",
		`+ bullet paragraph 1

  .. comment between bullet paragraphs 1 (leader) and 2

  bullet paragraph 2`,
		[]Token{
			tBulletPlus, tSpace, item(Paragraph, "bullet paragraph 1"), tBlankLine,
			tSpace2, tComment, tSpace, item(Paragraph, "comment between bullet paragraphs 1 (leader) and 2"),
			tBlankLine, tSpace2, item(Paragraph, "bullet paragraph 2"), tEOF,
		},
	},
	{
		"comment trailing bullet paragraph",
		`+ bullet

  .. trailing comment`,
		[]Token{
			tBulletPlus, tSpace, item(Paragraph, "bullet"), tBlankLine,
			tSpace2, tComment, tSpace, item(Paragraph, "trailing comment"), tEOF,
		},
	},
	{"comment, not target", ".. _", []Token{tComment, tSpace, item(Paragraph, "_"), tEOF}},
	// targets
	{
		"hyperlink target",
		`.. _target:

(Internal hyperlink target.)`,
		[]Token{
			tHyperlinkStart, tSpace, tHyperlinkPrefix,
			item(HyperlinkName, "target"), tHyperlinkSuffix,
			tBlankLine, item(Paragraph, "(Internal hyperlink target.)"), tEOF,
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
			item(Paragraph, "External hyperlink targets:"), tBlankLine,
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
			item(Paragraph, "Indirect hyperlink targets:"), tBlankLine,
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
			item(Paragraph, "External hyperlink:"), tBlankLine,
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
			item(Paragraph, "Malformed target:"), tBlankLine,
			tHyperlinkStart, tSpace, tHyperlinkPrefix, item(HyperlinkName, "_malformed"), tHyperlinkSuffix,
			tSpace, item(HyperlinkURI, "no good"), tBlankLine,
			item(Paragraph, "Target beginning with an underscore:"), tBlankLine,
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
			item(Paragraph, "Duplicate external targets (different URIs):"), tBlankLine,
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
			item(Paragraph, "Duplicate external targets (same URIs):"), tBlankLine,
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
			item(Paragraph, "Duplicate implicit targets."), tBlankLine,
			item(Title, "Title"), tSectionAdornment5,
			tBlankLine, item(Paragraph, "Paragraph."), tBlankLine,
			item(Title, "Title"), tSectionAdornment5,
			tBlankLine, item(Paragraph, "Paragraph."), tEOF,
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
			item(Paragraph, "Duplicate implicit/explicit targets."), tBlankLine,
			item(Title, "Title"), tSectionAdornment5,
			tBlankLine, tHyperlinkStart, tSpace, tHyperlinkPrefix, item(HyperlinkName, "title"), tHyperlinkSuffix,
			tBlankLine, item(Paragraph, "Paragraph."), tEOF,
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
			item(Paragraph, "Duplicate implicit/directive targets."), tBlankLine,
			item(Title, "Title"), tSectionAdornment5,
			tBlankLine, tComment, tSpace, item(Paragraph, "target-notes::"), // TODO: Should be Directive once implemented
			tSpace3, item(Paragraph, ":name: title"), tEOF,
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
			item(Paragraph, "Duplicate explicit targets."), tBlankLine,
			tHyperlinkStart, tSpace, tHyperlinkPrefix, item(HyperlinkName, "title"), tHyperlinkSuffix,
			tBlankLine, item(Paragraph, "First."), tBlankLine,
			tHyperlinkStart, tSpace, tHyperlinkPrefix, item(HyperlinkName, "title"), tHyperlinkSuffix,
			tBlankLine, item(Paragraph, "Second."), tBlankLine,
			tHyperlinkStart, tSpace, tHyperlinkPrefix, item(HyperlinkName, "title"), tHyperlinkSuffix,
			tBlankLine, item(Paragraph, "Third."), tEOF,
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
			item(Paragraph, "Duplicate explicit/directive targets."), tBlankLine,
			tHyperlinkStart, tSpace, tHyperlinkPrefix, item(HyperlinkName, "title"), tHyperlinkSuffix,
			tBlankLine, item(Paragraph, "First."), tBlankLine,
			tComment, tSpace, item(Paragraph, "rubric:: this is a title too"), // TODO: Should be Directive once implemented
			tSpace3, item(Paragraph, ":name: title"), tBlankLine, tEOF,
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
			item(Paragraph, "Duplicate targets:"), tBlankLine,
			item(Title, "Target"), item(SectionAdornment, "======"),
			tBlankLine, item(Paragraph, "Implicit section header target."), tBlankLine,
			tComment, tSpace, item(Paragraph, "[TARGET] Citation target."), // TODO: Should be Citation once implemented
			tBlankLine, tComment, tSpace, item(Paragraph, "[#target] Autonumber-labeled footnote target."), // TODO: Should be Footnote once implemented
			tBlankLine, tHyperlinkStart, tSpace, tHyperlinkPrefix, item(HyperlinkName, "target"), tHyperlinkSuffix,
			tBlankLine, item(Paragraph, "Explicit internal target."), tBlankLine,
			tHyperlinkStart, tSpace, tHyperlinkPrefix, item(HyperlinkName, "target"), tHyperlinkSuffix,
			tSpace, item(HyperlinkURI, "Explicit_external_target"), tBlankLine,
			tComment, tSpace, item(Paragraph, "rubric:: directive with target"), // TODO: Should be Directive once implemented
			tSpace3, item(Paragraph, ":name: Target"), tEOF,
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
			item(Paragraph, ": no good"), tBlankLine,
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
			item(Paragraph, "Anonymous external hyperlink target:"), tBlankLine,
			tHyperlinkStart, tSpace, tAnonHyperlinkPrefix, tHyperlinkSuffix,
			tSpace, item(HyperlinkURI, "http://w3c.org/"), tEOF,
		},
	},
	{
		"anonymous external hyperlink target, alternative syntax",
		`Anonymous external hyperlink target:

__ http://w3c.org/`,
		[]Token{
			item(Paragraph, "Anonymous external hyperlink target:"), tBlankLine,
			tAnonHyperlinkStart, tSpace, item(HyperlinkURI, "http://w3c.org/"), tEOF,
		},
	},
	{
		"anonymous indirect hyperlink target",
		`Anonymous indirect hyperlink target:

.. __: reference_`,
		[]Token{
			item(Paragraph, "Anonymous indirect hyperlink target:"), tBlankLine,
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
			item(Paragraph, "Anonymous external hyperlink target, not indirect:"), tBlankLine,
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
			item(Paragraph, "Anonymous indirect hyperlink targets:"), tBlankLine,
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
			item(Paragraph, "Mixed anonymous & named indirect hyperlink targets:"), tBlankLine,
			tAnonHyperlinkStart, tSpace, item(InlineReferenceText, "reference"), tInlineReferenceClose1,
			tHyperlinkStart, tSpace, tAnonHyperlinkPrefix, tHyperlinkSuffix, tSpace,
			item(InlineReferenceText, "reference"), tInlineReferenceClose1,
			tAnonHyperlinkStart, tSpace, item(InlineReferenceText, "reference"), tInlineReferenceClose1,
			tHyperlinkStart, tSpace, tHyperlinkPrefix, item(HyperlinkName, "target1"), tHyperlinkSuffix,
			tSpace, item(InlineReferenceText, "reference"), tInlineReferenceClose1,
			item(Paragraph, "no blank line"), tBlankLine,
			tHyperlinkStart, tSpace, tHyperlinkPrefix, item(HyperlinkName, "target2"), tHyperlinkSuffix,
			tSpace, item(InlineReferenceText, "reference"), tInlineReferenceClose1,
			tAnonHyperlinkStart, tSpace, item(InlineReferenceText, "reference"), tInlineReferenceClose1,
			tHyperlinkStart, tSpace, tAnonHyperlinkPrefix, tHyperlinkSuffix, tSpace,
			item(InlineReferenceText, "reference"), tInlineReferenceClose1,
			tAnonHyperlinkStart, tSpace, item(InlineReferenceText, "reference"), tInlineReferenceClose1,
			item(Paragraph, "no blank line"), tEOF,
		},
	},
	// paragraphs
	{"paragraph", "A paragraph.", []Token{item(Paragraph, "A paragraph."), tEOF}},
	{
		"2 paragraphs",
		`Paragraph 1.

Paragraph 2.`,
		[]Token{item(Paragraph, "Paragraph 1."), tBlankLine, item(Paragraph, "Paragraph 2."), tEOF},
	},
	{
		"paragraph with 3 lines",
		`Line 1.
Line 2.
Line 3.`,
		[]Token{item(Paragraph, "Line 1."), item(Paragraph, "Line 2."), item(Paragraph, "Line 3."), tEOF},
	},
	{
		"2 paragraphs with 3 lines",
		`Paragraph 1, Line 1.
Line 2.
Line 3.

Paragraph 2, Line 1.
Line 2.
Line 3.`,
		[]Token{
			item(Paragraph, "Paragraph 1, Line 1."), item(Paragraph, "Line 2."), item(Paragraph, "Line 3."), tBlankLine,
			item(Paragraph, "Paragraph 2, Line 1."), item(Paragraph, "Line 2."), item(Paragraph, "Line 3."), tEOF,
		},
	},
	{
		"paragraph with line break",
		`A. Einstein was a really
smart dude.`,
		[]Token{item(Paragraph, "A. Einstein was a really"), item(Paragraph, "smart dude."), tEOF},
	},
	// section headers
	{
		"title",
		`Title
=====

Paragraph.`,
		[]Token{item(Title, "Title"), tSectionAdornment5, tBlankLine, item(Paragraph, "Paragraph."), tEOF},
	},
	{
		"title, no line break",
		`Title
=====
Paragraph (no blank line).`,
		[]Token{item(Title, "Title"), tSectionAdornment5, item(Paragraph, "Paragraph (no blank line)."), tEOF},
	},
	{
		"paragraph, title, paragraph",
		`Paragraph.

Title
=====

Paragraph.`,
		[]Token{
			item(Paragraph, "Paragraph."), tBlankLine, item(Title, "Title"), tSectionAdornment5, tBlankLine,
			item(Paragraph, "Paragraph."), tEOF,
		},
	},
	{
		"unexpected section titles",
		`Test unexpected section titles.

    Title
    =====
    Paragraph.

    -----
    Title
    -----
    Paragraph.`,
		[]Token{
			item(Paragraph, "Test unexpected section titles."), tBlankLine, tSpace4, item(Title, "Title"),
			tSpace4, tSectionAdornment5, tSpace4, item(Paragraph, "Paragraph."), tBlankLine,
			tSpace4, tSectionAdornmentDash5, tSpace4, item(Title, "Title"), tSpace4, tSectionAdornmentDash5,
			tSpace4, item(Paragraph, "Paragraph."), tEOF,
		},
	},
	{
		"short underline",
		`Title
====

Test short underline.`,
		[]Token{
			item(Title, "Title"), item(SectionAdornment, "===="), tBlankLine,
			item(Paragraph, "Test short underline."), tEOF,
		},
	},
	{
		"title combining characters",
		`à with combining varia
======================

Do not count combining chars in title column width.`,
		[]Token{
			item(Title, "à with combining varia"), item(SectionAdornment, "======================"), tBlankLine,
			item(Paragraph, "Do not count combining chars in title column width."), tEOF,
		},
	},
	{
		"title, over/underline",
		`=====
Title
=====

Test overline title.`,
		[]Token{
			tSectionAdornment5, item(Title, "Title"), tSectionAdornment5, tBlankLine,
			item(Paragraph, "Test overline title."), tEOF,
		},
	},
	{
		"title, missing underline",
		`========================
 Test Missing Underline`,
		[]Token{tSectionAdornment24, tSpace, item(Paragraph, "Test Missing Underline"), tEOF},
	},
	{
		"title, missing underline, blank line",
		`========================
 Test Missing Underline

`,
		[]Token{tSectionAdornment24, tSpace, item(Paragraph, "Test Missing Underline"), tBlankLine, tEOF},
	},
	{
		"title, missing underline, paragraph",
		`=======
 Title

Test missing underline, with paragraph.`,
		[]Token{
			tSectionAdornment7, tSpace, item(Paragraph, "Title"), tBlankLine,
			item(Paragraph, "Test missing underline, with paragraph."), tEOF,
		},
	},
	{
		"long title",
		`=======
 Long    Title
=======

Test long title and space normalization.`,
		[]Token{
			tSectionAdornment7, tSpace, item(Title, "Long    Title"), tSectionAdornment7,
			tBlankLine, item(Paragraph, "Test long title and space normalization."), tEOF,
		},
	},
	{
		"title, over/underline mismatch",
		`=======
 Title
-------

Paragraph.`,
		[]Token{
			tSectionAdornment7, tSpace, item(Title, "Title"), tSectionAdornmentDash7,
			tBlankLine, item(Paragraph, "Paragraph."), tEOF,
		},
	},
	{
		"missing titles, blank line in-between",
		`========================

========================

Test missing titles; blank line in-between.

========================

========================`,
		[]Token{
			tTransition24, tBlankLine, tTransition24, tBlankLine,
			item(Paragraph, "Test missing titles; blank line in-between."), tBlankLine,
			tTransition24, tBlankLine, tTransition24, tEOF,
		},
	},
	{
		"missing titles",
		`========================
========================

Test missing titles; nothing in-between.

========================
========================`,
		[]Token{
			tSectionAdornment24, tSectionAdornment24, tBlankLine,
			item(Paragraph, "Test missing titles; nothing in-between."), tBlankLine,
			tSectionAdornment24, tSectionAdornment24, tEOF,
		},
	},
	{
		"highest-level section (Title 3)",
		`.. Test return to existing, highest-level section (Title 3).

Title 1
=======
Paragraph 1.

Title 2
-------
Paragraph 2.

Title 3
=======
Paragraph 3.

Title 4
-------
Paragraph 4.`,
		[]Token{
			tComment, tSpace, item(Paragraph, "Test return to existing, highest-level section (Title 3)."), tBlankLine,
			item(Title, "Title 1"), tSectionAdornment7, item(Paragraph, "Paragraph 1."), tBlankLine,
			item(Title, "Title 2"), tSectionAdornmentDash7, item(Paragraph, "Paragraph 2."), tBlankLine,
			item(Title, "Title 3"), tSectionAdornment7, item(Paragraph, "Paragraph 3."), tBlankLine,
			item(Title, "Title 4"), tSectionAdornmentDash7, item(Paragraph, "Paragraph 4."), tEOF,
		},
	},
	{
		"highest-level section (Title 3, with overlines)",
		`Test return to existing, highest-level section (Title 3, with overlines).

=======
Title 1
=======
Paragraph 1.

-------
Title 2
-------
Paragraph 2.

=======
Title 3
=======
Paragraph 3.

-------
Title 4
-------
Paragraph 4.`,
		[]Token{
			item(Paragraph, "Test return to existing, highest-level section (Title 3, with overlines)."), tBlankLine,
			tSectionAdornment7, item(Title, "Title 1"), tSectionAdornment7, item(Paragraph, "Paragraph 1."), tBlankLine,
			tSectionAdornmentDash7, item(Title, "Title 2"), tSectionAdornmentDash7, item(Paragraph, "Paragraph 2."), tBlankLine,
			tSectionAdornment7, item(Title, "Title 3"), tSectionAdornment7, item(Paragraph, "Paragraph 3."), tBlankLine,
			tSectionAdornmentDash7, item(Title, "Title 4"), tSectionAdornmentDash7, item(Paragraph, "Paragraph 4."), tEOF,
		},
	},
	{
		"higher-level section (Title 4)",
		`Test return to existing, higher-level section (Title 4).

Title 1
=======
Paragraph 1.

Title 2
-------
Paragraph 2.

Title 3
` + "```````" + `
Paragraph 3.

Title 4
-------
Paragraph 4.`,
		[]Token{
			item(Paragraph, "Test return to existing, higher-level section (Title 4)."), tBlankLine,
			item(Title, "Title 1"), tSectionAdornment7, item(Paragraph, "Paragraph 1."), tBlankLine,
			item(Title, "Title 2"), tSectionAdornmentDash7, item(Paragraph, "Paragraph 2."), tBlankLine,
			item(Title, "Title 3"), tSectionAdornmentTick7, item(Paragraph, "Paragraph 3."), tBlankLine,
			item(Title, "Title 4"), tSectionAdornmentDash7, item(Paragraph, "Paragraph 4."), tEOF,
		},
	},
	{
		"bad subsection order (Title 4)",
		`Test bad subsection order (Title 4).

Title 1
=======
Paragraph 1.

Title 2
-------
Paragraph 2.

Title 3
=======
Paragraph 3.

Title 4
` + "```````" + `
Paragraph 4.`,
		[]Token{
			item(Paragraph, "Test bad subsection order (Title 4)."), tBlankLine,
			item(Title, "Title 1"), tSectionAdornment7, item(Paragraph, "Paragraph 1."), tBlankLine,
			item(Title, "Title 2"), tSectionAdornmentDash7, item(Paragraph, "Paragraph 2."), tBlankLine,
			item(Title, "Title 3"), tSectionAdornment7, item(Paragraph, "Paragraph 3."), tBlankLine,
			item(Title, "Title 4"), tSectionAdornmentTick7, item(Paragraph, "Paragraph 4."), tEOF,
		},
	},
	{
		"bad subsection order (Title 4, with overlines)",
		`Test bad subsection order (Title 4, with overlines).

=======
Title 1
=======
Paragraph 1.

-------
Title 2
-------
Paragraph 2.

=======
Title 3
=======
Paragraph 3.

` + "```````" + `
Title 4
` + "```````" + `
Paragraph 4.`,
		[]Token{
			item(Paragraph, "Test bad subsection order (Title 4, with overlines)."), tBlankLine,
			tSectionAdornment7, item(Title, "Title 1"), tSectionAdornment7, item(Paragraph, "Paragraph 1."), tBlankLine,
			tSectionAdornmentDash7, item(Title, "Title 2"), tSectionAdornmentDash7, item(Paragraph, "Paragraph 2."), tBlankLine,
			tSectionAdornment7, item(Title, "Title 3"), tSectionAdornment7, item(Paragraph, "Paragraph 3."), tBlankLine,
			tSectionAdornmentTick7, item(Title, "Title 4"), tSectionAdornmentTick7, item(Paragraph, "Paragraph 4."), tEOF,
		},
	},
	{
		"title, inline markup",
		`Title containing *inline* ` + "``markup``" + `
====================================

Paragraph.`,
		[]Token{
			item(Title, "Title containing *inline* ``markup``"), // TODO: Should be InlineEmphasisOpen, etc. once implemented
			item(SectionAdornment, "===================================="), tBlankLine,
			item(Paragraph, "Paragraph."), tEOF,
		},
	},
	{
		"numbered title",
		`1. Numbered Title
=================

Paragraph.`,
		[]Token{
			item(Title, "1. Numbered Title"), item(SectionAdornment, "================="), tBlankLine,
			item(Paragraph, "Paragraph."), tEOF,
		},
	},
	{
		"enumerated list, numbered title",
		`1. Item 1.
2. Item 2.
3. Numbered Title
=================

Paragraph.`,
		[]Token{
			item(Paragraph, "1. Item 1."), item(Paragraph, "2. Item 2."), // TODO: Should be EnumListArabic, etc. once implemented
			item(Title, "3. Numbered Title"), item(SectionAdornment, "================="), tBlankLine,
			item(Paragraph, "Paragraph."), tEOF,
		},
	},
	{
		"short title",
		`ABC
===

Short title.`,
		[]Token{item(Title, "ABC"), tSectionAdornment3, tBlankLine, item(Paragraph, "Short title."), tEOF},
	},
	{
		"title, short underline",
		`ABC
==

Underline too short.`,
		[]Token{item(Title, "ABC"), tSectionAdornment2, tBlankLine, item(Paragraph, "Underline too short."), tEOF},
	},
	{
		"title, short over/underline",
		`==
ABC
==

Over & underline too short.`,
		[]Token{
			tSectionAdornment2, item(Title, "ABC"), tSectionAdornment2, tBlankLine,
			item(Paragraph, "Over & underline too short."), tEOF,
		},
	},
	{
		"title, short overline",
		`==
ABC

Overline too short, no underline.`,
		[]Token{
			tSectionAdornment2, item(Paragraph, "ABC"), tBlankLine,
			item(Paragraph, "Overline too short, no underline."), tEOF,
		},
	},
	{
		"incomplete section",
		`==
ABC`,
		[]Token{tSectionAdornment2, item(Paragraph, "ABC"), tEOF},
	},
	{
		"definition list",
		`==
  Not a title: a definition list item.`,
		[]Token{
			tSectionAdornment2, tSpace2, item(Paragraph, "Not a title: a definition list item."), // TODO: Should be DefinitionList once implemented
			tEOF,
		},
	},
	{
		"definition lists",
		`==
  Not a title: a definition list item.
--
  Another definition list item.  It's in a different list,
  but that's an acceptable limitation given that this will
  probably never happen in real life.

  The next line will trigger a warning:
==`,
		[]Token{
			tSectionAdornment2, tSpace2, item(Title, "Not a title: a definition list item."), // TODO: Should be DefinitionList once implemented
			item(SectionAdornment, "--"), tSpace2, item(Paragraph, "Another definition list item.  It's in a different list,"),
			tSpace2, item(Paragraph, "but that's an acceptable limitation given that this will"),
			tSpace2, item(Paragraph, "probably never happen in real life."), tBlankLine,
			tSpace2, item(Title, "The next line will trigger a warning:"), tSectionAdornment2, tEOF,
		},
	},
	{
		"indented title, short over/underline",
		`Paragraph

    ==
    ABC
    ==

    Over & underline too short.`,
		[]Token{
			item(Paragraph, "Paragraph"), tBlankLine, tSpace4, tSectionAdornment2,
			tSpace4, item(Title, "ABC"), tSpace4, tSectionAdornment2, tBlankLine, tSpace4,
			item(Paragraph, "Over & underline too short."), // TODO: Should be BlockQuote once implemented
			tEOF,
		},
	},
	{
		"indented title, short underline",
		`Paragraph

    ABC
    ==

    Underline too short.`,
		[]Token{
			item(Paragraph, "Paragraph"), tBlankLine, tSpace4, item(Title, "ABC"),
			tSpace4, tSectionAdornment2, tBlankLine, tSpace4,
			item(Paragraph, "Underline too short."), // TODO: Should be BlockQuote once implemented
			tEOF,
		},
	},
	{
		"incomplete sections",
		`...
...

...
---

...
...
...`,
		[]Token{
			tSectionAdornmentDot3, tSectionAdornmentDot3, tBlankLine,
			tSectionAdornmentDot3, tSectionAdornmentDash3, tBlankLine,
			tSectionAdornmentDot3, tSectionAdornmentDot3, tSectionAdornmentDot3, tEOF,
		},
	},
	{
		"2 character section titles",
		`..
Hi
..

...
Yo
...

Ho`,
		[]Token{
			tComment, item(Title, "Hi"), item(SectionAdornment, ".."), tBlankLine,
			tSectionAdornmentDot3, item(Title, "Yo"), tSectionAdornmentDot3, tBlankLine,
			item(Paragraph, "Ho"), tEOF,
		},
	},
	{
		"empty section",
		`Empty Section
=============`,
		[]Token{item(Title, "Empty Section"), item(SectionAdornment, "============="), tEOF},
	},
	{
		"3 character section titles",
		`===
One
===

The bubble-up parser strategy conflicts with short titles
(<= 3 char-long over- & underlines).

===
Two
===

The parser currently contains a work-around kludge.
Without it, the parser ends up in an infinite loop.`,
		[]Token{
			tSectionAdornment3, item(Title, "One"), tSectionAdornment3, tBlankLine,
			item(Paragraph, "The bubble-up parser strategy conflicts with short titles"),
			item(Paragraph, "(<= 3 char-long over- & underlines)."), tBlankLine,
			tSectionAdornment3, item(Title, "Two"), tSectionAdornment3, tBlankLine,
			item(Paragraph, "The parser currently contains a work-around kludge."),
			item(Paragraph, "Without it, the parser ends up in an infinite loop."), tEOF,
		},
	},
	// bullet lists
	{
		"bullet list", "- item",
		[]Token{tBulletDash, tSpace, item(Paragraph, "item"), tEOF},
	},
	{
		"bullet list, 2 items",
		`* item 1

* item 2`,
		[]Token{
			tBulletAsterisk, tSpace, item(Paragraph, "item 1"), tBlankLine,
			tBulletAsterisk, tSpace, item(Paragraph, "item 2"), tEOF,
		},
	},
	{
		"no blank line between bullet list",
		`No blank line between:

+ item 1
+ item 2`,
		[]Token{
			item(Paragraph, "No blank line between:"), tBlankLine,
			tBulletPlus, tSpace, item(Paragraph, "item 1"),
			tBulletPlus, tSpace, item(Paragraph, "item 2"), tEOF,
		},
	},
	{
		"bullet list with paragraph",
		`- item 1, para 1.

  item 1, para 2.

- item 2`,
		[]Token{
			tBulletDash, tSpace, item(Paragraph, "item 1, para 1."), tBlankLine,
			tSpace2, item(Paragraph, "item 1, para 2."), tBlankLine,
			tBulletDash, tSpace, item(Paragraph, "item 2"), tEOF,
		},
	},
	{
		"bullet list with paragraph, no blank line",
		`- item 1, line 1
  item 1, line 2
- item 2`,
		[]Token{
			tBulletDash, tSpace, item(Paragraph, "item 1, line 1"),
			tSpace2, item(Paragraph, "item 1, line 2"),
			tBulletDash, tSpace, item(Paragraph, "item 2"), tEOF,
		},
	},
	{
		"different bullets",
		`Different bullets:

- item 1

+ item 1

* item 1
- item 1`,
		[]Token{
			item(Paragraph, "Different bullets:"), tBlankLine,
			tBulletDash, tSpace, item(Paragraph, "item 1"), tBlankLine,
			tBulletPlus, tSpace, item(Paragraph, "item 1"), tBlankLine,
			tBulletAsterisk, tSpace, item(Paragraph, "item 1"),
			tBulletDash, tSpace, item(Paragraph, "item 1"), tEOF,
		},
	},
	{
		"bullet list, no blank line",
		`- item
no blank line`,
		[]Token{tBulletDash, tSpace, item(Paragraph, "item"), item(Paragraph, "no blank line"), tEOF},
	},
	{
		"empty bullet list",
		`-

empty item above`,
		[]Token{tBulletDash, tBlankLine, item(Paragraph, "empty item above"), tEOF},
	},
	{
		"empty bullet list, no blank line",
		`-
empty item above, no blank line`,
		[]Token{tBulletDash, item(Paragraph, "empty item above, no blank line"), tEOF},
	},
	{
		"unicode bullet lists",
		`Unicode bullets:

• BULLET

‣ TRIANGULAR BULLET

⁃ HYPHEN BULLET`,
		[]Token{
			item(Paragraph, "Unicode bullets:"), tBlankLine,
			item(Bullet, "•"), tSpace, item(Paragraph, "BULLET"), tBlankLine,
			item(Bullet, "‣"), tSpace, item(Paragraph, "TRIANGULAR BULLET"), tBlankLine,
			item(Bullet, "⁃"), tSpace, item(Paragraph, "HYPHEN BULLET"), tEOF,
		},
	},
	// transitions
	{
		"transition",
		`Test transition markers.

--------

Paragraph`,
		[]Token{
			item(Paragraph, "Test transition markers."), tBlankLine,
			tTransitionDash8, tBlankLine, item(Paragraph, "Paragraph"), tEOF,
		},
	},
	{
		"section, transition, section",
		`Section 1
=========
First text division of section 1.

--------

Second text division of section 1.

Section 2
---------
Paragraph 2 in section 2.`,
		[]Token{
			item(Title, "Section 1"), item(SectionAdornment, "========="),
			item(Paragraph, "First text division of section 1."), tBlankLine,
			tTransitionDash8, tBlankLine, item(Paragraph, "Second text division of section 1."),
			tBlankLine, item(Title, "Section 2"), item(SectionAdornment, "---------"),
			item(Paragraph, "Paragraph 2 in section 2."), tEOF,
		},
	},
	{
		"non-standard transitions",
		`--------

According to the DTD, a section or document may not begin with a transition.

Note: There is currently no warning, but in future these
DTD violations should be prevented or at least trigger a warning.
Alternatively, the DTD may be relaxed to accomodate for more use cases.

The DTD specifies that two transitions may not
be adjacent:

--------

--------

--------

The DTD also specifies that a section or document
may not end with a transition.

--------`,
		[]Token{
			tTransitionDash8, tBlankLine,
			item(Paragraph, "According to the DTD, a section or document may not begin with a transition."),
			tBlankLine, item(Paragraph, "Note: There is currently no warning, but in future these"),
			item(Paragraph, "DTD violations should be prevented or at least trigger a warning."),
			item(Paragraph, "Alternatively, the DTD may be relaxed to accomodate for more use cases."),
			tBlankLine, item(Paragraph, "The DTD specifies that two transitions may not"),
			item(Paragraph, "be adjacent:"), tBlankLine, tTransitionDash8, tBlankLine,
			tTransitionDash8, tBlankLine, tTransitionDash8, tBlankLine,
			item(Paragraph, "The DTD also specifies that a section or document"),
			item(Paragraph, "may not end with a transition."), tBlankLine,
			tTransitionDash8, tEOF,
		},
	},
	{
		"block quote, unexpected transition",
		`Test unexpected transition markers.

    Block quote.

    --------

    Paragraph.`,
		[]Token{
			item(Paragraph, "Test unexpected transition markers."), tBlankLine,
			tSpace4, item(Paragraph, "Block quote."), tBlankLine,
			tSpace4, tTransitionDash8, tBlankLine,
			tSpace4, item(Paragraph, "Paragraph."), tEOF,
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
