package runtime

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"php-src-go/internal/ast"
	"php-src-go/internal/runtime/builtins"
)

type Kind int

const (
	Null Kind = iota
	Bool
	Int
	Float
	String
	ArrayKind
	ObjectKind
	ClosureKind
)

type Closure struct {
	Fn       *ast.FunctionDeclStmt
	Captures map[string]*variableSlot
	// This/CalledClass implement PHP 5.3's automatic $this binding: a
	// closure literal written inside a method captures the enclosing
	// method's $this (and late-static-binding class) implicitly, without
	// needing `use ($this)`.
	This        *Object
	CalledClass string
}

type Value struct {
	Kind Kind
	B    bool
	I    int64
	F    float64
	S    string
	A    *Array
	O    *Object
	C    *Closure
}

func NullValue() Value            { return Value{Kind: Null} }
func BoolValue(v bool) Value      { return Value{Kind: Bool, B: v} }
func IntValue(v int64) Value      { return Value{Kind: Int, I: v} }
func FloatValue(v float64) Value  { return Value{Kind: Float, F: v} }
func StringValue(v string) Value  { return Value{Kind: String, S: v} }
func ArrayValue(v *Array) Value   { return Value{Kind: ArrayKind, A: v} }
func ObjectValue(v *Object) Value { return Value{Kind: ObjectKind, O: v} }
func ClosureValue(v *Closure) Value {
	return Value{Kind: ClosureKind, C: v}
}

func (v Value) Truthy() bool {
	switch v.Kind {
	case Null:
		return false
	case Bool:
		return v.B
	case Int:
		return v.I != 0
	case Float:
		return v.F != 0
	case String:
		return v.S != "" && v.S != "0"
	case ArrayKind:
		return v.A != nil && len(v.A.Entries) > 0
	case ObjectKind:
		return v.O != nil
	case ClosureKind:
		return v.C != nil
	}
	return false
}

func (v Value) String() string {
	switch v.Kind {
	case Null:
		return ""
	case Bool:
		if v.B {
			return "1"
		}
		return ""
	case Int:
		return strconv.FormatInt(v.I, 10)
	case Float:
		return builtins.FormatPHPFloat(v.F)
	case String:
		return v.S
	case ArrayKind:
		return "Array"
	case ObjectKind:
		return "Object"
	case ClosureKind:
		return "Closure"
	}
	return ""
}

func (v Value) number() (float64, bool) {
	switch v.Kind {
	case Int:
		return float64(v.I), true
	case Float:
		return v.F, true
	case Bool:
		if v.B {
			return 1, true
		}
		return 0, true
	case Null:
		return 0, true
	case String:
		return parseNumericPrefix(v.S), true
	}
	return 0, false
}

func Binary(op string, a, b Value) (Value, error) {
	value, diagnostic, err := BinaryWithDiagnostic(op, a, b)
	if diagnostic != nil {
		return value, newRuntimeError(diagnostic.Severity, diagnostic.Message)
	}
	return value, err
}

func BinaryWithDiagnostic(op string, a, b Value) (Value, *Diagnostic, error) {
	switch op {
	case "+", "-", "*", "/", "%":
		na, oka := a.number()
		nb, okb := b.number()
		if !oka || !okb {
			return NullValue(), nil, fmt.Errorf("non numeric operand")
		}
		switch op {
		case "+", "-", "*":
			var result float64
			switch op {
			case "+":
				result = na + nb
			case "-":
				result = na - nb
			case "*":
				result = na * nb
			}
			if arithmeticKind(a) != Float && arithmeticKind(b) != Float {
				value, err := numericResult(result)
				return value, nil, err
			}
			return FloatValue(result), nil, nil
		case "/":
			if nb == 0 {
				return BoolValue(false), &Diagnostic{Severity: Warning, Message: "Division by zero"}, nil
			}
			if isIntLikeKind(a.Kind) && isIntLikeKind(b.Kind) {
				left := int64(na)
				right := int64(nb)
				if left%right == 0 {
					return IntValue(left / right), nil, nil
				}
			}
			return FloatValue(na / nb), nil, nil
		case "%":
			left := int64(na)
			right := int64(nb)
			if right == 0 {
				return BoolValue(false), &Diagnostic{Severity: Warning, Message: "Division by zero"}, nil
			}
			return IntValue(left % right), nil, nil
		}
	case "&", "|", "^":
		if a.Kind == String && b.Kind == String {
			return StringValue(bitwiseStringBinary(op, a.S, b.S)), nil, nil
		}
		na, oka := a.number()
		nb, okb := b.number()
		if !oka || !okb {
			return NullValue(), nil, fmt.Errorf("non numeric operand")
		}
		left := int64(na)
		right := int64(nb)
		switch op {
		case "&":
			return IntValue(left & right), nil, nil
		case "|":
			return IntValue(left | right), nil, nil
		case "^":
			return IntValue(left ^ right), nil, nil
		}
	case "<<", ">>":
		na, oka := a.number()
		nb, okb := b.number()
		if !oka || !okb {
			return NullValue(), nil, fmt.Errorf("non numeric operand")
		}
		left := int64(na)
		right := uint(int64(nb))
		switch op {
		case "<<":
			return IntValue(left << right), nil, nil
		case ">>":
			return IntValue(left >> right), nil, nil
		}
	case "==":
		return BoolValue(looseEqual(a, b)), nil, nil
	case "!=", "<>":
		return BoolValue(!looseEqual(a, b)), nil, nil
	case "===":
		return BoolValue(strictEqual(a, b)), nil, nil
	case "!==":
		return BoolValue(!strictEqual(a, b)), nil, nil
	case "<", ">", "<=", ">=":
		cmp, ok := looseCompare(a, b)
		if !ok {
			return NullValue(), nil, fmt.Errorf("non comparable operand")
		}
		switch op {
		case "<":
			return BoolValue(cmp < 0), nil, nil
		case ">":
			return BoolValue(cmp > 0), nil, nil
		case "<=":
			return BoolValue(cmp <= 0), nil, nil
		case ">=":
			return BoolValue(cmp >= 0), nil, nil
		}
	}
	return NullValue(), nil, fmt.Errorf("unsupported operator %s", op)
}

func BitwiseNot(value Value) (Value, error) {
	if value.Kind == String {
		bytes := []byte(value.S)
		for idx := range bytes {
			bytes[idx] = ^bytes[idx]
		}
		return StringValue(string(bytes)), nil
	}
	return Binary("^", value, IntValue(-1))
}

func bitwiseStringBinary(op string, a, b string) string {
	left := []byte(a)
	right := []byte(b)
	longer := left
	shorter := right
	if len(right) > len(left) {
		longer = right
		shorter = left
	}
	if op == "|" {
		result := append([]byte(nil), longer...)
		for idx := range shorter {
			result[idx] |= shorter[idx]
		}
		return string(result)
	}
	result := append([]byte(nil), shorter...)
	for idx := range result {
		switch op {
		case "&":
			result[idx] &= longer[idx]
		case "^":
			result[idx] ^= longer[idx]
		}
	}
	return string(result)
}

func looseEqual(a, b Value) bool {
	if a.Kind == Bool || b.Kind == Bool {
		return a.Truthy() == b.Truthy()
	}
	if a.Kind == Null || b.Kind == Null {
		return nullLooseEqual(a, b)
	}
	if isNumberKind(a.Kind) && b.Kind == String {
		return numericValue(a) == parseNumericPrefix(b.S)
	}
	if a.Kind == String && isNumberKind(b.Kind) {
		return parseNumericPrefix(a.S) == numericValue(b)
	}
	if isNumberKind(a.Kind) && isNumberKind(b.Kind) {
		return numericValue(a) == numericValue(b)
	}
	if a.Kind == String && b.Kind == String {
		if an, aok := fullNumericString(a.S); aok {
			if bn, bok := fullNumericString(b.S); bok {
				return an == bn
			}
		}
		return a.S == b.S
	}
	if a.Kind == ArrayKind || b.Kind == ArrayKind {
		return a.Kind == ArrayKind && b.Kind == ArrayKind && looseArrayEqual(a.A, b.A)
	}
	if a.Kind == ObjectKind || b.Kind == ObjectKind {
		return a.Kind == ObjectKind && b.Kind == ObjectKind && looseObjectEqual(a.O, b.O)
	}
	return a.String() == b.String()
}

func nullLooseEqual(a, b Value) bool {
	if a.Kind == Null && b.Kind == Null {
		return true
	}
	other := b
	if b.Kind == Null {
		other = a
	}
	switch other.Kind {
	case Bool:
		return !other.B
	case Int, Float:
		return numericValue(other) == 0
	case String:
		return other.S == ""
	case ArrayKind:
		return other.A == nil || len(other.A.Entries) == 0
	default:
		return false
	}
}

func looseCompare(a, b Value) (int, bool) {
	if a.Kind == Bool || b.Kind == Bool {
		return compareFloat(boolNumber(a.Truthy()), boolNumber(b.Truthy())), true
	}
	if a.Kind == Null || b.Kind == Null {
		return compareFloat(nullCompareNumber(a), nullCompareNumber(b)), true
	}
	if isNumberKind(a.Kind) && b.Kind == String {
		return compareFloat(numericValue(a), parseNumericPrefix(b.S)), true
	}
	if a.Kind == String && isNumberKind(b.Kind) {
		return compareFloat(parseNumericPrefix(a.S), numericValue(b)), true
	}
	if isNumberKind(a.Kind) && isNumberKind(b.Kind) {
		return compareFloat(numericValue(a), numericValue(b)), true
	}
	if a.Kind == String && b.Kind == String {
		if an, aok := fullNumericString(a.S); aok {
			if bn, bok := fullNumericString(b.S); bok {
				return compareFloat(an, bn), true
			}
		}
		return strings.Compare(a.S, b.S), true
	}
	if a.Kind == ArrayKind || b.Kind == ArrayKind {
		if a.Kind == ArrayKind && b.Kind == ArrayKind {
			return compareArrays(a.A, b.A), true
		}
		if a.Kind == ArrayKind {
			return 1, true
		}
		return -1, true
	}
	if a.Kind == ObjectKind || b.Kind == ObjectKind {
		if a.Kind == ObjectKind && b.Kind == ObjectKind {
			return compareObjects(a.O, b.O), true
		}
		if a.Kind == ObjectKind {
			return 1, true
		}
		return -1, true
	}
	return strings.Compare(a.String(), b.String()), true
}

func isNumberKind(kind Kind) bool {
	return kind == Int || kind == Float
}

func numericValue(value Value) float64 {
	switch value.Kind {
	case Int:
		return float64(value.I)
	case Float:
		return value.F
	default:
		return 0
	}
}

func fullNumericString(text string) (float64, bool) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return 0, false
	}
	value, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		return 0, false
	}
	return value, true
}

func boolNumber(value bool) float64 {
	if value {
		return 1
	}
	return 0
}

func nullCompareNumber(value Value) float64 {
	if value.Kind == Null {
		return 0
	}
	if value.Kind == String {
		return parseNumericPrefix(value.S)
	}
	if value.Kind == Bool {
		return boolNumber(value.B)
	}
	if isNumberKind(value.Kind) {
		return numericValue(value)
	}
	if value.Kind == ArrayKind && value.A != nil && len(value.A.Entries) > 0 {
		return 1
	}
	return 0
}

func compareFloat(a, b float64) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

func looseArrayEqual(a, b *Array) bool {
	if arrayLen(a) != arrayLen(b) {
		return false
	}
	for _, entry := range arrayEntries(a) {
		other, ok := b.Get(entry.Key)
		if !ok || !looseEqual(entry.RuntimeValue(), other) {
			return false
		}
	}
	return true
}

func strictArrayEqual(a, b *Array) bool {
	left := arrayEntries(a)
	right := arrayEntries(b)
	if len(left) != len(right) {
		return false
	}
	for idx, leftEntry := range left {
		rightEntry := right[idx]
		if leftEntry.Key.identity() != rightEntry.Key.identity() {
			return false
		}
		if !strictEqual(leftEntry.RuntimeValue(), rightEntry.RuntimeValue()) {
			return false
		}
	}
	return true
}

func compareArrays(a, b *Array) int {
	if arrayLen(a) < arrayLen(b) {
		return -1
	}
	if arrayLen(a) > arrayLen(b) {
		return 1
	}
	for _, entry := range arrayEntries(a) {
		other, ok := b.Get(entry.Key)
		if !ok {
			return 1
		}
		cmp, ok := looseCompare(entry.RuntimeValue(), other)
		if !ok {
			continue
		}
		if cmp != 0 {
			return cmp
		}
	}
	return 0
}

func arrayLen(a *Array) int {
	if a == nil {
		return 0
	}
	return len(a.Entries)
}

func arrayEntries(a *Array) []ArrayEntry {
	if a == nil {
		return nil
	}
	return a.Entries
}

func looseObjectEqual(a, b *Object) bool {
	if a == b {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if !strings.EqualFold(a.ClassName, b.ClassName) {
		return false
	}
	if len(a.Props) != len(b.Props) {
		return false
	}
	for _, key := range sortedObjectPropertyKeys(a) {
		left := a.Props[key]
		right, ok := b.Props[key]
		if !ok || !looseEqual(left, right) {
			return false
		}
	}
	return true
}

func compareObjects(a, b *Object) int {
	if a == b {
		return 0
	}
	if a == nil {
		return -1
	}
	if b == nil {
		return 1
	}
	if cmp := strings.Compare(strings.ToLower(a.ClassName), strings.ToLower(b.ClassName)); cmp != 0 {
		return cmp
	}
	if len(a.Props) < len(b.Props) {
		return -1
	}
	if len(a.Props) > len(b.Props) {
		return 1
	}
	for _, key := range sortedObjectPropertyKeys(a) {
		right, ok := b.Props[key]
		if !ok {
			return 1
		}
		cmp, ok := looseCompare(a.Props[key], right)
		if !ok {
			continue
		}
		if cmp != 0 {
			return cmp
		}
	}
	for _, key := range sortedObjectPropertyKeys(b) {
		if _, ok := a.Props[key]; !ok {
			return -1
		}
	}
	return 0
}

func sortedObjectPropertyKeys(obj *Object) []string {
	if obj == nil {
		return nil
	}
	keys := make([]string, 0, len(obj.Props))
	for key := range obj.Props {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func strictEqual(a, b Value) bool {
	if a.Kind != b.Kind {
		return false
	}
	switch a.Kind {
	case Null:
		return true
	case Bool:
		return a.B == b.B
	case Int:
		return a.I == b.I
	case Float:
		return a.F == b.F
	case String:
		return a.S == b.S
	case ArrayKind:
		return strictArrayEqual(a.A, b.A)
	case ObjectKind:
		return a.O == b.O
	default:
		return false
	}
}

// maxInt64Float is 2^63, the smallest float64 magnitude that no longer fits
// in an int64. PHP promotes arithmetic results to float on overflow instead
// of wrapping, and int64(v) for out-of-range or non-finite v is undefined
// (or silently wraps), so both cases must be routed to FloatValue first.
const maxInt64Float = 9223372036854775808.0

func numericResult(v float64) (Value, error) {
	if math.Trunc(v) == v && v >= -maxInt64Float && v < maxInt64Float {
		return IntValue(int64(v)), nil
	}
	return FloatValue(v), nil
}

// cloneArrayValue returns an independent copy of v when it is an array,
// implementing PHP's array value (copy-on-assignment) semantics (BUG-9):
// every other Kind is returned unchanged, since scalars are already copied
// by Go's Value struct assignment and objects/closures are intentionally
// shared by PHP's own reference-type semantics. Call this at every point
// where a Value moves into new independent storage by value — variable
// assignment, non-by-ref argument/capture binding, array-literal/element
// construction, foreach without & — never at reference-binding sites.
func cloneArrayValue(v Value) Value {
	if v.Kind == ArrayKind && v.A != nil {
		return ArrayValue(v.A.Clone())
	}
	return v
}

// isIntLikeKind reports whether a value's Kind carries an exact integer
// numeric value (Int/Bool/Null), as opposed to Float/String where the
// division result must stay float even when it happens to come out whole.
func isIntLikeKind(k Kind) bool {
	return k == Int || k == Bool || k == Null
}

// arithmeticKind reports whether a value contributes an Int- or Float-typed
// operand to +/-/* PHP arithmetic, per PHP's rule that the result is float
// if either operand is (or is shaped like) a float, regardless of whether
// the numeric result happens to come out whole. Int/Bool/Null always
// contribute Int; Float always contributes Float; String contributes Float
// only when its leading numeric prefix has a decimal point or exponent
// (e.g. "5.0", "1e2") — a bare-digit prefix like "5" or "10 apples"
// contributes Int, matching PHP's numeric-string int/float distinction.
func arithmeticKind(v Value) Kind {
	switch v.Kind {
	case Float:
		return Float
	case String:
		if stringNumericPrefixIsFloatForm(v.S) {
			return Float
		}
		return Int
	default:
		return Int
	}
}

// stringNumericPrefixIsFloatForm scans the same leading numeric prefix as
// parseNumericPrefix and reports whether it contains a decimal point or
// exponent, mirroring PHP's is_numeric_string int/float classification.
func stringNumericPrefixIsFloatForm(text string) bool {
	text = strings.TrimLeft(text, " \t\r\n")
	end := 0
	if end < len(text) && (text[end] == '+' || text[end] == '-') {
		end++
	}
	for end < len(text) && text[end] >= '0' && text[end] <= '9' {
		end++
	}
	if end < len(text) && text[end] == '.' {
		return true
	}
	if end < len(text) && (text[end] == 'e' || text[end] == 'E') {
		expEnd := end + 1
		if expEnd < len(text) && (text[expEnd] == '+' || text[expEnd] == '-') {
			expEnd++
		}
		if expEnd < len(text) && text[expEnd] >= '0' && text[expEnd] <= '9' {
			return true
		}
	}
	return false
}
