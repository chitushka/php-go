package builtins

import (
	"strconv"
	"strings"
)

type ArrayKey struct {
	IsInt bool
	Int   int64
	Str   string
}

func IntKey(v int64) ArrayKey     { return ArrayKey{IsInt: true, Int: v} }
func StringKey(v string) ArrayKey { return ArrayKey{Str: v} }

func (k ArrayKey) identity() string {
	if k.IsInt {
		return "i:" + strconv.FormatInt(k.Int, 10)
	}
	return "s:" + k.Str
}

type ArrayEntry struct {
	Key   ArrayKey
	Value Value
}

type Array struct {
	Entries []ArrayEntry
	index   map[string]int
	nextKey int64
}

func NewArray() *Array {
	return &Array{index: map[string]int{}}
}

func (a *Array) Append(v Value) {
	key := IntKey(a.nextKey)
	a.nextKey++
	a.Set(key, v)
}

func (a *Array) Set(key ArrayKey, v Value) {
	id := key.identity()
	if pos, ok := a.index[id]; ok {
		a.Entries[pos].Value = v
		return
	}
	a.index[id] = len(a.Entries)
	a.Entries = append(a.Entries, ArrayEntry{Key: key, Value: v})
	if key.IsInt && key.Int >= a.nextKey {
		a.nextKey = key.Int + 1
	}
}

func (a *Array) Get(key ArrayKey) (Value, bool) {
	if pos, ok := a.index[key.identity()]; ok {
		return a.Entries[pos].Value, true
	}
	return NullValue(), false
}

func KeyFromValue(v Value) ArrayKey {
	switch v.Kind {
	case IntKind:
		return IntKey(v.I)
	case BoolKind:
		if v.B {
			return IntKey(1)
		}
		return IntKey(0)
	case FloatKind:
		return IntKey(int64(v.F))
	case NullKind:
		return StringKey("")
	default:
		return StringKey(v.String())
	}
}

func count(args []Value) (Value, error) {
	if err := checkArgCount("count", len(args), 1, 1); err != nil {
		return NullValue(), err
	}
	if args[0].Kind == ArrayKind && args[0].A != nil {
		return IntValue(int64(len(args[0].A.Entries))), nil
	}
	if args[0].Kind == NullKind {
		return IntValue(0), nil
	}
	return IntValue(1), nil
}

func arrayKeyExists(args []Value) (Value, error) {
	if err := checkArgCount("array_key_exists", len(args), 2, 2); err != nil {
		return NullValue(), err
	}
	if args[1].Kind != ArrayKind || args[1].A == nil {
		return BoolValue(false), nil
	}
	_, ok := args[1].A.Get(KeyFromValue(args[0]))
	return BoolValue(ok), nil
}

func inArray(args []Value) (Value, error) {
	if err := checkArgCount("in_array", len(args), 2, 2); err != nil {
		return NullValue(), err
	}
	if args[1].Kind != ArrayKind || args[1].A == nil {
		return BoolValue(false), nil
	}
	for _, entry := range args[1].A.Entries {
		if entry.Value.String() == args[0].String() {
			return BoolValue(true), nil
		}
	}
	return BoolValue(false), nil
}

func implode(args []Value) (Value, error) {
	if err := checkArgCount("implode", len(args), 2, 2); err != nil {
		return NullValue(), err
	}
	glue := args[0].String()
	if args[1].Kind != ArrayKind || args[1].A == nil {
		return StringValue(""), nil
	}
	parts := make([]string, 0, len(args[1].A.Entries))
	for _, entry := range args[1].A.Entries {
		parts = append(parts, entry.Value.String())
	}
	return StringValue(strings.Join(parts, glue)), nil
}

func explode(args []Value) (Value, error) {
	if err := checkArgCount("explode", len(args), 2, 2); err != nil {
		return NullValue(), err
	}
	delimiter := args[0].String()
	s := args[1].String()
	arr := NewArray()
	if delimiter == "" {
		arr.Append(StringValue(s))
		return ArrayValue(arr), nil
	}
	for _, part := range strings.Split(s, delimiter) {
		arr.Append(StringValue(part))
	}
	return ArrayValue(arr), nil
}
