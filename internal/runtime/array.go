package runtime

import "strconv"

type ArrayKey struct {
	IsInt bool
	Int   int64
	Str   string
}

func IntKey(v int64) ArrayKey     { return ArrayKey{IsInt: true, Int: v} }
func StringKey(v string) ArrayKey { return ArrayKey{Str: v} }

func (k ArrayKey) Value() Value {
	if k.IsInt {
		return IntValue(k.Int)
	}
	return StringValue(k.Str)
}

func (k ArrayKey) identity() string {
	if k.IsInt {
		return "i:" + strconv.FormatInt(k.Int, 10)
	}
	return "s:" + k.Str
}

type ArrayEntry struct {
	Key   ArrayKey
	Value Value
	Slot  *variableSlot
}

func (e ArrayEntry) RuntimeValue() Value {
	if e.Slot != nil {
		return e.Slot.value
	}
	return e.Value
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

func (a *Array) AppendSlot(slot *variableSlot) {
	key := IntKey(a.nextKey)
	a.nextKey++
	a.SetSlot(key, slot)
}

func (a *Array) Set(key ArrayKey, v Value) {
	id := key.identity()
	if pos, ok := a.index[id]; ok {
		if a.Entries[pos].Slot != nil {
			a.Entries[pos].Slot.value = v
			return
		}
		a.Entries[pos].Value = v
		return
	}
	a.index[id] = len(a.Entries)
	a.Entries = append(a.Entries, ArrayEntry{Key: key, Value: v})
	if key.IsInt && key.Int >= a.nextKey {
		a.nextKey = key.Int + 1
	}
}

func (a *Array) SetSlot(key ArrayKey, slot *variableSlot) {
	id := key.identity()
	if pos, ok := a.index[id]; ok {
		a.Entries[pos].Value = slot.value
		a.Entries[pos].Slot = slot
		return
	}
	a.index[id] = len(a.Entries)
	a.Entries = append(a.Entries, ArrayEntry{Key: key, Value: slot.value, Slot: slot})
	if key.IsInt && key.Int >= a.nextKey {
		a.nextKey = key.Int + 1
	}
}

func (a *Array) EnsureSlot(key ArrayKey) *variableSlot {
	if pos, ok := a.index[key.identity()]; ok {
		if a.Entries[pos].Slot == nil {
			a.Entries[pos].Slot = &variableSlot{value: a.Entries[pos].Value}
		}
		return a.Entries[pos].Slot
	}
	slot := &variableSlot{}
	a.SetSlot(key, slot)
	return slot
}

func (a *Array) Get(key ArrayKey) (Value, bool) {
	if pos, ok := a.index[key.identity()]; ok {
		return a.Entries[pos].RuntimeValue(), true
	}
	return NullValue(), false
}

func (a *Array) Delete(key ArrayKey) {
	id := key.identity()
	pos, ok := a.index[id]
	if !ok {
		return
	}
	delete(a.index, id)
	a.Entries = append(a.Entries[:pos], a.Entries[pos+1:]...)
	for idx := pos; idx < len(a.Entries); idx++ {
		a.index[a.Entries[idx].Key.identity()] = idx
	}
}

func KeyFromValue(v Value) ArrayKey {
	switch v.Kind {
	case Int:
		return IntKey(v.I)
	case Bool:
		if v.B {
			return IntKey(1)
		}
		return IntKey(0)
	case Float:
		return IntKey(int64(v.F))
	case Null:
		return StringKey("")
	default:
		return StringKey(v.String())
	}
}
