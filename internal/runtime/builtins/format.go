package builtins

import (
	"strconv"
	"strings"
)

func printR(args []Value) (Value, string, error) {
	if err := checkArgCount("print_r", len(args), 1, 2); err != nil {
		return NullValue(), "", err
	}
	output := printRValue(args[0], 0)
	if len(args) == 2 && args[1].Truthy() {
		return StringValue(output), "", nil
	}
	return BoolValue(true), output, nil
}

func printRValue(value Value, depth int) string {
	if value.Kind != ArrayKind || value.A == nil {
		return value.String()
	}
	indent := strings.Repeat("    ", depth)
	childIndent := strings.Repeat("    ", depth+1)
	var b strings.Builder
	b.WriteString("Array\n")
	b.WriteString(indent)
	b.WriteString("(\n")
	for _, entry := range value.A.Entries {
		b.WriteString(childIndent)
		b.WriteString("[")
		b.WriteString(arrayKeyString(entry.Key))
		b.WriteString("] => ")
		b.WriteString(printRValue(entry.Value, depth+1))
		b.WriteString("\n")
	}
	b.WriteString(indent)
	b.WriteString(")\n")
	return b.String()
}

func varDump(args []Value) (Value, string, error) {
	if len(args) == 0 {
		return NullValue(), "", checkArgCount("var_dump", len(args), 1, 1)
	}
	var b strings.Builder
	for _, arg := range args {
		writeVarDump(&b, arg, 0)
	}
	return NullValue(), b.String(), nil
}

func writeVarDump(b *strings.Builder, value Value, depth int) {
	indent := strings.Repeat("  ", depth)
	switch value.Kind {
	case NullKind:
		b.WriteString(indent)
		b.WriteString("NULL\n")
	case BoolKind:
		b.WriteString(indent)
		if value.B {
			b.WriteString("bool(true)\n")
		} else {
			b.WriteString("bool(false)\n")
		}
	case IntKind:
		b.WriteString(indent)
		b.WriteString("int(")
		b.WriteString(strconv.FormatInt(value.I, 10))
		b.WriteString(")\n")
	case FloatKind:
		b.WriteString(indent)
		b.WriteString("float(")
		b.WriteString(FormatPHPFloat(value.F))
		b.WriteString(")\n")
	case StringKind:
		b.WriteString(indent)
		b.WriteString("string(")
		b.WriteString(strconv.Itoa(len(value.S)))
		b.WriteString(") \"")
		b.WriteString(value.S)
		b.WriteString("\"\n")
	case ArrayKind:
		b.WriteString(indent)
		if value.A == nil {
			b.WriteString("array(0) {\n")
			b.WriteString(indent)
			b.WriteString("}\n")
			return
		}
		b.WriteString("array(")
		b.WriteString(strconv.Itoa(len(value.A.Entries)))
		b.WriteString(") {\n")
		for _, entry := range value.A.Entries {
			b.WriteString(indent)
			b.WriteString("  [")
			b.WriteString(arrayKeyString(entry.Key))
			b.WriteString("]=>\n")
			writeVarDump(b, entry.Value, depth+1)
		}
		b.WriteString(indent)
		b.WriteString("}\n")
	case ObjectKind:
		b.WriteString(indent)
		b.WriteString("object\n")
	}
}

func arrayKeyString(key ArrayKey) string {
	if key.IsInt {
		return strconv.FormatInt(key.Int, 10)
	}
	return key.Str
}
