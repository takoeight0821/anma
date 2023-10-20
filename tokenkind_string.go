// Code generated by "stringer -type=TokenKind"; DO NOT EDIT.

package main

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[EOF-0]
	_ = x[LEFTPAREN-1]
	_ = x[RIGHTPAREN-2]
	_ = x[LEFTBRACE-3]
	_ = x[RIGHTBRACE-4]
	_ = x[LEFTBRACKET-5]
	_ = x[RIGHTBRACKET-6]
	_ = x[COLON-7]
	_ = x[COMMA-8]
	_ = x[DOT-9]
	_ = x[SEMICOLON-10]
	_ = x[SHARP-11]
	_ = x[IDENT-12]
	_ = x[OPERATOR-13]
	_ = x[INTEGER-14]
	_ = x[STRING-15]
	_ = x[ARROW-16]
	_ = x[CASE-17]
	_ = x[DEF-18]
	_ = x[EQUAL-19]
	_ = x[FN-20]
	_ = x[INFIX-21]
	_ = x[INFIXL-22]
	_ = x[INFIXR-23]
	_ = x[LET-24]
	_ = x[TYPE-25]
}

const _TokenKind_name = "EOFLEFTPARENRIGHTPARENLEFTBRACERIGHTBRACELEFTBRACKETRIGHTBRACKETCOLONCOMMADOTSEMICOLONSHARPIDENTOPERATORINTEGERSTRINGARROWCASEDEFEQUALFNINFIXINFIXLINFIXRLETTYPE"

var _TokenKind_index = [...]uint8{0, 3, 12, 22, 31, 41, 52, 64, 69, 74, 77, 86, 91, 96, 104, 111, 117, 122, 126, 129, 134, 136, 141, 147, 153, 156, 160}

func (i TokenKind) String() string {
	if i < 0 || i >= TokenKind(len(_TokenKind_index)-1) {
		return "TokenKind(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _TokenKind_name[_TokenKind_index[i]:_TokenKind_index[i+1]]
}
