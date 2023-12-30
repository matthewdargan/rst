package parse

import (
	"fmt"
	"testing"
)

// Make the types prettyprint.
var itemName = map[itemType]string{
	itemError:   "error",
	itemComment: "comment",
	itemEOF:     "EOF",
	itemNewLine: "newline",
	itemSpace:   "space",
	itemText:    "text",
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
	tComment = mkItem(itemComment, "..")
	tEOF     = mkItem(itemEOF, "")
	tNewLine = mkItem(itemNewLine, "\n")
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
		[]item{tComment, mkItem(itemSpace, " "), mkItem(itemText, "A comment"), tNewLine, mkItem(itemText, "Paragraph."), tEOF},
	},
	{
		"comment block",
		`.. A comment
   block.

Paragraph.
`,
		[]item{
			tComment, mkItem(itemSpace, " "), mkItem(itemText, "A comment"), mkItem(itemSpace, "   "), mkItem(itemText, "block."),
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
			tComment, mkItem(itemSpace, "   "), mkItem(itemText, "A comment consisting of multiple lines"),
			mkItem(itemSpace, "   "), mkItem(itemText, "starting on the line after the"),
			mkItem(itemSpace, "   "), mkItem(itemText, "explicit markup start."), tEOF,
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
			return // TODO
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
