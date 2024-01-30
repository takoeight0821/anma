// Code generated by "stringer -type=Kind"; DO NOT EDIT.

package token

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
	_ = x[BAR-17]
	_ = x[CASE-18]
	_ = x[DEF-19]
	_ = x[EQUAL-20]
	_ = x[FN-21]
	_ = x[INFIX-22]
	_ = x[INFIXL-23]
	_ = x[INFIXR-24]
	_ = x[LET-25]
	_ = x[TYPE-26]
	_ = x[PRIM-27]
}

const _Kind_name = "EOFLEFTPARENRIGHTPARENLEFTBRACERIGHTBRACELEFTBRACKETRIGHTBRACKETCOLONCOMMADOTSEMICOLONSHARPIDENTOPERATORINTEGERSTRINGARROWBARCASEDEFEQUALFNINFIXINFIXLINFIXRLETTYPEPRIM"

var _Kind_index = [...]uint8{0, 3, 12, 22, 31, 41, 52, 64, 69, 74, 77, 86, 91, 96, 104, 111, 117, 122, 125, 129, 132, 137, 139, 144, 150, 156, 159, 163, 167}

func (i Kind) String() string {
	if i < 0 || i >= Kind(len(_Kind_index)-1) {
		return "Kind(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _Kind_name[_Kind_index[i]:_Kind_index[i+1]]
}
