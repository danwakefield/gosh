//go:generate stringer -type=Token
package arith

type Token int

const (
	EOFRune = -1

	ArithError Token = iota
	ArithAssignment
	ArithNot
	ArithAnd
	ArithOr
	ArithNumber
	ArithVariable

	// These tokens are all binary operations requiring two arguments
	// (E.g 1+2)
	ArithLessEqual
	ArithGreaterEqual
	ArithLessThan
	ArithGreaterThan
	ArithEqual
	ArithNotEqual

	// These binary operations also have assignment equivalents
	ArithBinaryAnd
	ArithBinaryOr
	ArithBinaryXor
	ArithLeftShift
	ArithRightShift
	ArithRemainder
	ArithMultiply
	ArithDivide
	ArithSubtract
	ArithAdd

	// These tokens perform assignment to a variable as well as an
	// operation (E.g  x+=1)
	ArithAssignBinaryAnd
	ArithAssignBinaryOr
	ArithAssignBinaryXor
	ArithAssignLeftShift
	ArithAssignRightShift
	ArithAssignRemainder
	ArithAssignMultiply
	ArithAssignDivide
	ArithAssignSubtract
	ArithAssignAdd

	ArithLeftParen
	ArithRightParen
	ArithBinaryNot
	ArithQuestionMark
	ArithColon

	ArithEOF

	// ArithAssignDiff is used to turn an Arith token into its ArithAssign equivalent.
	ArithAssignDiff Token = ArithAssignBinaryAnd - ArithBinaryAnd
)
