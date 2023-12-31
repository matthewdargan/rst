// Copyright 2023 Matthew P. Dargan. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parse

import (
	"fmt"
	"testing"
)

// Make the types prettyprint.
var itemName = map[itemType]string{
	itemError:           "error",
	itemComment:         "comment",
	itemEOF:             "EOF",
	itemHyperlinkName:   "hyperlink name",
	itemHyperlinkPrefix: "hyperlink prefix",
	itemHyperlinkStart:  "hyperlink start",
	itemHyperlinkSuffix: "hyperlink suffix",
	itemNewLine:         "newline",
	itemSpace:           "space",
	itemText:            "text",
}

func (i itemType) String() string {
	s := itemName[i]
	if s == "" {
		return fmt.Sprintf("item%d", int(i))
	}
	return s
}

type lexTest struct {
	name  string
	input string
	items []item
}

func mkItem(typ itemType, text string) item {
	return item{
		typ: typ,
		val: text,
	}
}

var (
	tComment        = mkItem(itemComment, "..")
	tEOF            = mkItem(itemEOF, "")
	tHyperlinkStart = mkItem(itemHyperlinkStart, "..")
	tNewLine        = mkItem(itemNewLine, "\n")
	tSpace          = mkItem(itemSpace, " ")
	tIndent2        = mkItem(itemSpace, "  ")
	tIndent3        = mkItem(itemSpace, "   ")
)

var lexTests = []lexTest{
	{"empty", "", []item{tEOF}},
	{"spaces", " \t\n", []item{mkItem(itemSpace, " \t\n"), tEOF}},
	{"text", `now is the time`, []item{mkItem(itemText, "now is the time"), tEOF}},
	// comments
	{
		"line comment",
		`.. A comment

Paragraph.
`,
		[]item{tComment, tSpace, mkItem(itemText, "A comment"), tNewLine, mkItem(itemText, "Paragraph."), tEOF},
	},
	{
		"comment block",
		`.. A comment
   block.

Paragraph.
`,
		[]item{
			tComment, tSpace, mkItem(itemText, "A comment"), tIndent3, mkItem(itemText, "block."),
			tNewLine, mkItem(itemText, "Paragraph."), tEOF,
		},
	},
	{
		"multi-line comment block",
		`..
   A comment consisting of multiple lines
   starting on the line after the
   explicit markup start.
`,
		[]item{
			tComment, tIndent3, mkItem(itemText, "A comment consisting of multiple lines"),
			tIndent3, mkItem(itemText, "starting on the line after the"),
			tIndent3, mkItem(itemText, "explicit markup start."), tEOF,
		},
	},
	{
		"2 line comments",
		`.. A comment.
.. Another.

Paragraph.
`,
		[]item{
			tComment, tSpace, mkItem(itemText, "A comment."),
			tComment, tSpace, mkItem(itemText, "Another."),
			tNewLine, mkItem(itemText, "Paragraph."), tEOF,
		},
	},
	{
		"line comment, no blank line",
		`.. A comment
no blank line

Paragraph.
`,
		[]item{
			tComment, tSpace, mkItem(itemText, "A comment"), mkItem(itemText, "no blank line"),
			tNewLine, mkItem(itemText, "Paragraph."), tEOF,
		},
	},
	{
		"2 line comments, no blank line",
		`.. A comment.
.. Another.
no blank line

Paragraph.
`,
		[]item{
			tComment, tSpace, mkItem(itemText, "A comment."),
			tComment, tSpace, mkItem(itemText, "Another."), mkItem(itemText, "no blank line"),
			tNewLine, mkItem(itemText, "Paragraph."), tEOF,
		},
	},
	{
		"line comment with directive",
		`.. A comment::

Paragraph.
`,
		[]item{tComment, tSpace, mkItem(itemText, "A comment::"), tNewLine, mkItem(itemText, "Paragraph."), tEOF},
	},
	{
		"comment block with directive",
		`..
   comment::

The extra newline before the comment text prevents
the parser from recognizing a directive.
`,
		[]item{
			tComment, tIndent3, mkItem(itemText, "comment::"), tNewLine,
			mkItem(itemText, "The extra newline before the comment text prevents"),
			mkItem(itemText, "the parser from recognizing a directive."), tEOF,
		},
	},
	{
		"comment block with hyperlink target",
		`..
   _comment: http://example.org

The extra newline before the comment text prevents
the parser from recognizing a hyperlink target.
`,
		[]item{
			tComment, tIndent3, mkItem(itemText, "_comment: http://example.org"), tNewLine,
			mkItem(itemText, "The extra newline before the comment text prevents"),
			mkItem(itemText, "the parser from recognizing a hyperlink target."), tEOF,
		},
	},
	{
		"comment block with citation",
		`..
   [comment] Not a citation.

The extra newline before the comment text prevents
the parser from recognizing a citation.
`,
		[]item{
			tComment, tIndent3, mkItem(itemText, "[comment] Not a citation."), tNewLine,
			mkItem(itemText, "The extra newline before the comment text prevents"),
			mkItem(itemText, "the parser from recognizing a citation."), tEOF,
		},
	},
	{
		"comment block with substitution definition",
		`..
   |comment| image:: bogus.png

The extra newline before the comment text prevents
the parser from recognizing a substitution definition.
`,
		[]item{
			tComment, tIndent3, mkItem(itemText, "|comment| image:: bogus.png"), tNewLine,
			mkItem(itemText, "The extra newline before the comment text prevents"),
			mkItem(itemText, "the parser from recognizing a substitution definition."), tEOF,
		},
	},
	{
		"comment block and empty comment",
		`.. Next is an empty comment, which serves to end this comment and
   prevents the following block quote being swallowed up.

..

    A block quote.
`,
		[]item{
			tComment, tSpace, mkItem(itemText, "Next is an empty comment, which serves to end this comment and"),
			tIndent3, mkItem(itemText, "prevents the following block quote being swallowed up."),
			tNewLine, tComment, tNewLine, mkItem(itemSpace, "    "),
			mkItem(itemText, "A block quote."), // TODO: Should be itemBlockQuote once implemented
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
		[]item{
			mkItem(itemText, "term 1"), // TODO: Should be itemDefinitionTerm once implemented
			tIndent2, mkItem(itemText, "definition 1"), tNewLine,
			tIndent2, tComment, tSpace, mkItem(itemText, "a comment"), tNewLine,
			mkItem(itemText, "term 2"), tIndent2, mkItem(itemText, "definition 2"), tEOF,
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
		[]item{
			mkItem(itemText, "term 1"), // TODO: Should be itemDefinitionTerm once implemented
			tIndent2, mkItem(itemText, "definition 1"), tNewLine,
			tComment, tSpace, mkItem(itemText, "a comment"), tNewLine,
			mkItem(itemText, "term 2"), tIndent2, mkItem(itemText, "definition 2"), tEOF,
		},
	},
	{
		"comment between bullet paragraphs 2 and 3",
		`+ bullet paragraph 1

  bullet paragraph 2

  .. comment between bullet paragraphs 2 and 3

  bullet paragraph 3
`,
		[]item{
			mkItem(itemText, "+ bullet paragraph 1"), // TODO: Should be itemBullet once implemented
			tNewLine, tIndent2, mkItem(itemText, "bullet paragraph 2"), tNewLine,
			tIndent2, tComment, tSpace, mkItem(itemText, "comment between bullet paragraphs 2 and 3"),
			tNewLine, tIndent2, mkItem(itemText, "bullet paragraph 3"), tEOF,
		},
	},
	{
		"comment between bullet paragraphs 1 and 2",
		`+ bullet paragraph 1

  .. comment between bullet paragraphs 1 (leader) and 2

  bullet paragraph 2
`,
		[]item{
			mkItem(itemText, "+ bullet paragraph 1"), // TODO: Should be itemBullet once implemented
			tNewLine, tIndent2,
			tComment, tSpace, mkItem(itemText, "comment between bullet paragraphs 1 (leader) and 2"),
			tNewLine, tIndent2, mkItem(itemText, "bullet paragraph 2"), tEOF,
		},
	},
	{
		"comment trailing bullet paragraph",
		`+ bullet

  .. trailing comment
`,
		[]item{
			mkItem(itemText, "+ bullet"), // TODO: Should be itemBullet once implemented
			tNewLine, tIndent2, tComment, tSpace, mkItem(itemText, "trailing comment"), tEOF,
		},
	},
	// hyperlink targets
	{
		"hyperlink target",
		`.. _target:

(Internal hyperlink target.)
`,
		[]item{
			tHyperlinkStart, tSpace, mkItem(itemHyperlinkPrefix, "_"),
			mkItem(itemHyperlinkName, "target"), mkItem(itemHyperlinkSuffix, ":"),
			tNewLine, mkItem(itemText, "(Internal hyperlink target.)"), tEOF,
		},
	},
}

// collect gathers the emitted items into a slice.
func collect(t *lexTest) (items []item) {
	l := lex(t.name, t.input)
	for {
		i := l.nextItem()
		items = append(items, i)
		if i.typ == itemEOF || i.typ == itemError {
			break
		}
	}
	return
}

func equal(i1, i2 []item, checkPos bool) bool {
	if len(i1) != len(i2) {
		return false
	}
	for k := range i1 {
		if i1[k].typ != i2[k].typ {
			return false
		}
		if i1[k].val != i2[k].val {
			return false
		}
		if checkPos && i1[k].pos != i2[k].pos {
			return false
		}
		if checkPos && i1[k].line != i2[k].line {
			return false
		}
	}
	return true
}

func TestLex(t *testing.T) {
	for _, test := range lexTests {
		items := collect(&test)
		if !equal(items, test.items, false) {
			t.Errorf("%s: got\n\t%+v\nexpected\n\t%v", test.name, items, test.items)
			return
		}
		t.Log(test.name, "OK")
	}
}

// parseLexer is a local version of parse that lets us pass in the lexer instead of building it.
// We expect an error, so the tree set and funcs list are explicitly nil.
func (t *Tree) parseLexer(lex *lexer) (tree *Tree, err error) {
	defer t.recover(&err)
	t.ParseName = t.Name
	t.startParse(nil, lex, map[string]*Tree{})
	t.parse()
	t.add()
	t.stopParse()
	return t, nil
}
