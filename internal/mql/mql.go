// Package mql parses the restricted MongoDB query surface accepted by the
// query bar. It supports the four read commands find, aggregate, count, and
// distinct with shell-style object and array literals. It deliberately rejects
// arbitrary JavaScript and unknown commands before an engine call.
package mql

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// Command names accepted by the query bar.
const (
	Find      = "find"
	Aggregate = "aggregate"
	Count     = "count"
	Distinct  = "distinct"
)

// Pair is one object field. An ordered slice keeps first-seen field order,
// which matters for sort and projection documents.
type Pair struct {
	Key   string
	Value any
}

// Object is one ordered MQL document literal.
type Object []Pair

// Array is one MQL array literal.
type Array []any

// ObjectID is one ObjectId("...") literal.
type ObjectID string

// Date is one ISODate("...") literal.
type Date time.Time

// Time returns the parsed timestamp.
func (d Date) Time() time.Time { return time.Time(d) }

// Command is one parsed MQL statement.
type Command struct {
	Name       string
	Collection string
	Filter     any
	Pipeline   []any
	Projection any
	Sort       any
	Field      string
	Skip       int64
	Limit      int64
}

type parser struct {
	runes []rune
	pos   int
	saved int
}

// Parse parses one MQL statement. The optional db.<collection>. prefix is
// accepted for copy-paste compatibility; without it the caller supplies the
// selected collection.
func Parse(statement string) (Command, error) {
	p := &parser{runes: []rune(statement)}
	p.skipSpace()
	if p.eof() {
		return Command{}, fmt.Errorf("statement is empty")
	}
	var command Command
	if p.peekIdentifier() == "db" {
		p.savepoint()
		p.nextIdentifier()
		if p.consume('.') {
			collection, err := p.collectionName()
			if err != nil {
				return Command{}, err
			}
			if !p.consume('.') {
				return Command{}, p.errorf("expected '.' before the command")
			}
			command.Collection = collection
		} else {
			p.restore()
		}
	}
	name, err := p.identifier()
	if err != nil {
		return Command{}, err
	}
	name = strings.ToLower(name)
	switch name {
	case Find, Aggregate, Count, Distinct:
		command.Name = name
	default:
		return Command{}, fmt.Errorf("unsupported MQL command %q; use find, aggregate, count, or distinct", name)
	}
	if !p.consume('(') {
		return Command{}, p.errorf("expected '(' after %s", name)
	}
	switch name {
	case Find:
		command.Filter = nil
		if !p.peek(')') {
			filter, err := p.value()
			if err != nil {
				return Command{}, err
			}
			command.Filter = filter
			if p.consume(',') {
				projection, err := p.value()
				if err != nil {
					return Command{}, err
				}
				command.Projection = projection
			}
		}
	case Aggregate:
		value, err := p.value()
		if err != nil {
			return Command{}, err
		}
		pipeline, ok := value.(Array)
		if !ok {
			return Command{}, p.errorf("aggregate expects an array pipeline")
		}
		command.Pipeline = []any(pipeline)
	case Count:
		command.Filter = nil
		if !p.peek(')') {
			filter, err := p.value()
			if err != nil {
				return Command{}, err
			}
			command.Filter = filter
		}
	case Distinct:
		value, err := p.value()
		if err != nil {
			return Command{}, err
		}
		field, ok := value.(string)
		if !ok {
			return Command{}, p.errorf("distinct expects a field name string")
		}
		command.Field = field
		if p.consume(',') {
			filter, err := p.value()
			if err != nil {
				return Command{}, err
			}
			command.Filter = filter
		}
	}
	if !p.consume(')') {
		return Command{}, p.errorf("expected ')'")
	}
	if name == Find {
		if err := p.chains(&command); err != nil {
			return Command{}, err
		}
	}
	p.skipSpace()
	p.consume(';')
	p.skipSpace()
	if !p.eof() {
		return Command{}, p.errorf("multiple statements are not allowed")
	}
	return command, nil
}

func (p *parser) chains(command *Command) error {
	for {
		p.skipSpace()
		if !p.consume('.') {
			return nil
		}
		name, err := p.identifier()
		if err != nil {
			return err
		}
		name = strings.ToLower(name)
		if !p.consume('(') {
			return p.errorf("expected '(' after .%s", name)
		}
		value, err := p.value()
		if err != nil {
			return err
		}
		if !p.consume(')') {
			return p.errorf("expected ')'")
		}
		switch name {
		case "sort":
			if _, ok := value.(Object); !ok {
				return p.errorf(".sort expects an object")
			}
			command.Sort = value
		case "project", "projection":
			if _, ok := value.(Object); !ok {
				return p.errorf(".%s expects an object", name)
			}
			command.Projection = value
		case "skip":
			number, ok := value.(int64)
			if !ok || number < 0 {
				return p.errorf(".skip expects a non-negative integer")
			}
			command.Skip = number
		case "limit":
			number, ok := value.(int64)
			if !ok || number < 0 {
				return p.errorf(".limit expects a non-negative integer")
			}
			command.Limit = number
		default:
			return p.errorf("unsupported cursor method .%s", name)
		}
	}
}

func (p *parser) collectionName() (string, error) {
	p.skipSpace()
	if p.eof() {
		return "", p.errorf("missing collection name")
	}
	if p.peek('"') || p.peek('\'') {
		value, err := p.string()
		if err != nil {
			return "", err
		}
		return value, nil
	}
	start := p.pos
	for !p.eof() {
		r := p.runes[p.pos]
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-') {
			break
		}
		p.pos++
	}
	if p.pos == start {
		return "", p.errorf("invalid collection name")
	}
	return string(p.runes[start:p.pos]), nil
}

func (p *parser) value() (any, error) {
	p.skipSpace()
	if p.eof() {
		return nil, p.errorf("missing value")
	}
	switch r := p.runes[p.pos]; {
	case r == '{':
		return p.object()
	case r == '[':
		return p.array()
	case r == '\'' || r == '"':
		return p.string()
	case r == '-' || unicode.IsDigit(r):
		return p.number()
	case r == '/':
		return nil, p.errorf("regular expression literals are not supported; use $regex")
	case unicode.IsLetter(r) || r == '_' || r == '$':
		return p.word()
	default:
		return nil, p.errorf("unexpected character %q", r)
	}
}

func (p *parser) object() (Object, error) {
	if !p.consume('{') {
		return nil, p.errorf("expected '{'")
	}
	p.skipSpace()
	object := Object{}
	if p.consume('}') {
		return object, nil
	}
	for {
		p.skipSpace()
		var key string
		var err error
		if p.peek('"') || p.peek('\'') {
			key, err = p.string()
		} else {
			key, err = p.key()
		}
		if err != nil {
			return nil, err
		}
		p.skipSpace()
		if !p.consume(':') {
			return nil, p.errorf("expected ':' after object key")
		}
		value, err := p.value()
		if err != nil {
			return nil, err
		}
		if key == "$where" || key == "$function" || key == "$accumulator" {
			return nil, p.errorf("arbitrary JavaScript is not allowed")
		}
		object = append(object, Pair{Key: key, Value: value})
		p.skipSpace()
		if p.consume('}') {
			return object, nil
		}
		if !p.consume(',') {
			return nil, p.errorf("expected ',' or '}' in object")
		}
	}
}

func (p *parser) array() (Array, error) {
	if !p.consume('[') {
		return nil, p.errorf("expected '['")
	}
	p.skipSpace()
	array := Array{}
	if p.consume(']') {
		return array, nil
	}
	for {
		value, err := p.value()
		if err != nil {
			return nil, err
		}
		array = append(array, value)
		p.skipSpace()
		if p.consume(']') {
			return array, nil
		}
		if !p.consume(',') {
			return nil, p.errorf("expected ',' or ']' in array")
		}
	}
}

func (p *parser) number() (any, error) {
	start := p.pos
	if p.peek('-') {
		p.pos++
	}
	for !p.eof() && unicode.IsDigit(p.runes[p.pos]) {
		p.pos++
	}
	floating := false
	if p.peek('.') {
		floating = true
		p.pos++
		for !p.eof() && unicode.IsDigit(p.runes[p.pos]) {
			p.pos++
		}
	}
	if p.peek('e') || p.peek('E') {
		floating = true
		p.pos++
		if p.peek('+') || p.peek('-') {
			p.pos++
		}
		for !p.eof() && unicode.IsDigit(p.runes[p.pos]) {
			p.pos++
		}
	}
	text := string(p.runes[start:p.pos])
	if text == "" || text == "-" {
		return nil, p.errorf("invalid number")
	}
	if floating {
		value, err := strconv.ParseFloat(text, 64)
		if err != nil {
			return nil, p.errorf("invalid number")
		}
		return value, nil
	}
	value, err := strconv.ParseInt(text, 10, 64)
	if err != nil {
		return nil, p.errorf("invalid integer")
	}
	return value, nil
}

func (p *parser) word() (any, error) {
	name, err := p.identifier()
	if err != nil {
		return nil, err
	}
	switch name {
	case "true":
		return true, nil
	case "false":
		return false, nil
	case "null":
		return nil, nil
	case "ObjectId", "ObjectID":
		p.skipSpace()
		if !p.consume('(') {
			return nil, p.errorf("ObjectId expects a string")
		}
		value, err := p.string()
		if err != nil {
			return nil, err
		}
		p.skipSpace()
		if !p.consume(')') {
			return nil, p.errorf("ObjectId expects a closing ')'")
		}
		return ObjectID(value), nil
	case "ISODate":
		p.skipSpace()
		if !p.consume('(') {
			return nil, p.errorf("ISODate expects a string")
		}
		value, err := p.string()
		if err != nil {
			return nil, err
		}
		p.skipSpace()
		if !p.consume(')') {
			return nil, p.errorf("ISODate expects a closing ')'")
		}
		parsed, parseErr := time.Parse(time.RFC3339, value)
		if parseErr != nil {
			return nil, p.errorf("ISODate expects an RFC3339 timestamp")
		}
		return Date(parsed), nil
	default:
		if p.peek('(') {
			return nil, p.errorf("unsupported function %q", name)
		}
		return nil, p.errorf("unknown value %q", name)
	}
}

func (p *parser) string() (string, error) {
	if p.eof() || (p.runes[p.pos] != '\'' && p.runes[p.pos] != '"') {
		return "", p.errorf("expected a string")
	}
	quote := p.runes[p.pos]
	p.pos++
	var b strings.Builder
	for !p.eof() {
		r := p.runes[p.pos]
		p.pos++
		if r == quote {
			return b.String(), nil
		}
		if r != '\\' {
			b.WriteRune(r)
			continue
		}
		if p.eof() {
			return "", p.errorf("unterminated string escape")
		}
		escaped := p.runes[p.pos]
		p.pos++
		switch escaped {
		case 'n':
			b.WriteRune('\n')
		case 't':
			b.WriteRune('\t')
		case 'r':
			b.WriteRune('\r')
		case '\\', '\'', '"', '/':
			b.WriteRune(escaped)
		default:
			b.WriteRune(escaped)
		}
	}
	return "", p.errorf("unterminated string")
}

func (p *parser) key() (string, error) {
	start := p.pos
	if !p.eof() && p.runes[p.pos] == '$' {
		p.pos++
	}
	for !p.eof() {
		r := p.runes[p.pos]
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '$') {
			break
		}
		p.pos++
	}
	if p.pos == start {
		return "", p.errorf("expected an object key")
	}
	return string(p.runes[start:p.pos]), nil
}

func (p *parser) identifier() (string, error) {
	p.skipSpace()
	start := p.pos
	for !p.eof() {
		r := p.runes[p.pos]
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_') {
			break
		}
		p.pos++
	}
	if p.pos == start {
		return "", p.errorf("expected an identifier")
	}
	return string(p.runes[start:p.pos]), nil
}

func (p *parser) peekIdentifier() string {
	p.skipSpace()
	save := p.pos
	name, err := p.identifier()
	p.pos = save
	if err != nil {
		return ""
	}
	return name
}

func (p *parser) nextIdentifier() string {
	name, _ := p.identifier()
	return name
}

func (p *parser) skipSpace() {
	for !p.eof() {
		r := p.runes[p.pos]
		if unicode.IsSpace(r) {
			p.pos++
			continue
		}
		if r == '/' && p.pos+1 < len(p.runes) && p.runes[p.pos+1] == '/' {
			p.pos += 2
			for !p.eof() && p.runes[p.pos] != '\n' {
				p.pos++
			}
			continue
		}
		if r == '/' && p.pos+1 < len(p.runes) && p.runes[p.pos+1] == '*' {
			p.pos += 2
			for p.pos+1 < len(p.runes) && !(p.runes[p.pos] == '*' && p.runes[p.pos+1] == '/') {
				p.pos++
			}
			if p.pos+1 < len(p.runes) {
				p.pos += 2
			}
			continue
		}
		return
	}
}

func (p *parser) consume(r rune) bool {
	if p.eof() || p.runes[p.pos] != r {
		return false
	}
	p.pos++
	return true
}

func (p *parser) peek(r rune) bool {
	return !p.eof() && p.runes[p.pos] == r
}

func (p *parser) eof() bool { return p.pos >= len(p.runes) }

func (p *parser) savepoint() { p.saved = p.pos }
func (p *parser) restore()   { p.pos = p.saved }

func (p *parser) errorf(format string, args ...any) error {
	return fmt.Errorf("%s at column %d", fmt.Sprintf(format, args...), p.pos+1)
}
