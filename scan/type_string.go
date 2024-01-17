// Code generated by "stringer -type Type"; DO NOT EDIT.

package scan

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[EOF-0]
	_ = x[Error-1]
	_ = x[BlankLine-2]
	_ = x[Space-3]
	_ = x[Title-4]
	_ = x[SectionAdornment-5]
	_ = x[Transition-6]
	_ = x[Paragraph-7]
	_ = x[Bullet-8]
	_ = x[Comment-9]
	_ = x[HyperlinkStart-10]
	_ = x[HyperlinkPrefix-11]
	_ = x[HyperlinkQuote-12]
	_ = x[HyperlinkName-13]
	_ = x[HyperlinkSuffix-14]
	_ = x[HyperlinkURI-15]
	_ = x[InlineReferenceOpen-16]
	_ = x[InlineReferenceText-17]
	_ = x[InlineReferenceClose-18]
}

const _Type_name = "EOFErrorBlankLineSpaceTitleSectionAdornmentTransitionParagraphBulletCommentHyperlinkStartHyperlinkPrefixHyperlinkQuoteHyperlinkNameHyperlinkSuffixHyperlinkURIInlineReferenceOpenInlineReferenceTextInlineReferenceClose"

var _Type_index = [...]uint8{0, 3, 8, 17, 22, 27, 43, 53, 62, 68, 75, 89, 104, 118, 131, 146, 158, 177, 196, 216}

func (i Type) String() string {
	if i < 0 || i >= Type(len(_Type_index)-1) {
		return "Type(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _Type_name[_Type_index[i]:_Type_index[i+1]]
}
