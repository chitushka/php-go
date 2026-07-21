package builtins

import (
	"fmt"
	"strconv"
	"strings"
)

func serializeFn(args []Value) (Value, error) {
	if err := checkArgCount("serialize", len(args), 1, 1); err != nil {
		return NullValue(), err
	}
	return StringValue(serializeValue(args[0])), nil
}

func serializeValue(value Value) string {
	switch value.Kind {
	case NullKind:
		return "N;"
	case BoolKind:
		if value.B {
			return "b:1;"
		}
		return "b:0;"
	case IntKind:
		return "i:" + strconv.FormatInt(value.I, 10) + ";"
	case FloatKind:
		return "d:" + FormatPHPFloat(value.F) + ";"
	case StringKind:
		return "s:" + strconv.Itoa(len(value.S)) + ":\"" + value.S + "\";"
	case ArrayKind:
		if value.A == nil {
			return "a:0:{}"
		}
		var b strings.Builder
		b.WriteString("a:")
		b.WriteString(strconv.Itoa(len(value.A.Entries)))
		b.WriteString(":{")
		for _, entry := range value.A.Entries {
			if entry.Key.IsInt {
				b.WriteString("i:")
				b.WriteString(strconv.FormatInt(entry.Key.Int, 10))
				b.WriteString(";")
			} else {
				b.WriteString(serializeValue(StringValue(entry.Key.Str)))
			}
			b.WriteString(serializeValue(entry.Value))
		}
		b.WriteString("}")
		return b.String()
	default:
		return "N;"
	}
}

func unserializeFn(args []Value) (Value, error) {
	if err := checkArgCount("unserialize", len(args), 1, 1); err != nil {
		return NullValue(), err
	}
	parser := serializedParser{input: args[0].String()}
	value, err := parser.parse()
	if err != nil {
		return NullValue(), err
	}
	return value, nil
}

type serializedParser struct {
	input string
	pos   int
}

func (p *serializedParser) parse() (Value, error) {
	if p.pos >= len(p.input) {
		return NullValue(), fmt.Errorf("unexpected end of serialized value")
	}
	switch p.input[p.pos] {
	case 'N':
		p.pos += 2
		return NullValue(), nil
	case 'b':
		n, err := p.readNumberAfterPrefix()
		if err != nil {
			return NullValue(), err
		}
		return BoolValue(n != 0), nil
	case 'i':
		n, err := p.readNumberAfterPrefix()
		if err != nil {
			return NullValue(), err
		}
		return IntValue(n), nil
	case 'd':
		raw, err := p.readRawAfterPrefix()
		if err != nil {
			return NullValue(), err
		}
		f, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return NullValue(), err
		}
		return FloatValue(f), nil
	case 's':
		return p.parseString()
	case 'a':
		return p.parseArray()
	default:
		return NullValue(), fmt.Errorf("unsupported serialized type %q", p.input[p.pos])
	}
}

func (p *serializedParser) parseString() (Value, error) {
	p.pos += 2
	lengthText, err := p.readUntil(':')
	if err != nil {
		return NullValue(), err
	}
	length, err := strconv.Atoi(lengthText)
	if err != nil {
		return NullValue(), err
	}
	if p.pos >= len(p.input) || p.input[p.pos] != '"' {
		return NullValue(), fmt.Errorf("expected string quote")
	}
	p.pos++
	if p.pos+length > len(p.input) {
		return NullValue(), fmt.Errorf("serialized string length out of range")
	}
	s := p.input[p.pos : p.pos+length]
	p.pos += length
	if p.pos+2 > len(p.input) || p.input[p.pos] != '"' || p.input[p.pos+1] != ';' {
		return NullValue(), fmt.Errorf("invalid serialized string terminator")
	}
	p.pos += 2
	return StringValue(s), nil
}

func (p *serializedParser) parseArray() (Value, error) {
	p.pos += 2
	countText, err := p.readUntil(':')
	if err != nil {
		return NullValue(), err
	}
	count, err := strconv.Atoi(countText)
	if err != nil {
		return NullValue(), err
	}
	if p.pos >= len(p.input) || p.input[p.pos] != '{' {
		return NullValue(), fmt.Errorf("expected array body")
	}
	p.pos++
	arr := NewArray()
	for idx := 0; idx < count; idx++ {
		key, err := p.parse()
		if err != nil {
			return NullValue(), err
		}
		value, err := p.parse()
		if err != nil {
			return NullValue(), err
		}
		arr.Set(KeyFromValue(key), value)
	}
	if p.pos >= len(p.input) || p.input[p.pos] != '}' {
		return NullValue(), fmt.Errorf("expected array terminator")
	}
	p.pos++
	return ArrayValue(arr), nil
}

func (p *serializedParser) readNumberAfterPrefix() (int64, error) {
	raw, err := p.readRawAfterPrefix()
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(raw, 10, 64)
}

func (p *serializedParser) readRawAfterPrefix() (string, error) {
	p.pos += 2
	return p.readUntil(';')
}

func (p *serializedParser) readUntil(delim byte) (string, error) {
	start := p.pos
	for p.pos < len(p.input) && p.input[p.pos] != delim {
		p.pos++
	}
	if p.pos >= len(p.input) {
		return "", fmt.Errorf("delimiter %q not found", delim)
	}
	out := p.input[start:p.pos]
	p.pos++
	return out, nil
}
