// Package guard validates one statement before it reaches an engine. SQL uses a
// conservative token blocklist; MongoDB uses the restricted MQL parser. The
// guard is an input safeguard, not a security boundary.
package guard

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/mcgru/db-adm-tui/internal/mql"
)

// Decision is the guard verdict for one statement. A blocked statement never
// reaches the engine; the reason is safe to display.
type Decision struct {
	Allowed  bool
	Reason   string
	ModeHint bool
}

var sqlWriteTokens = map[string]bool{
	"insert": true, "update": true, "delete": true, "merge": true,
	"truncate": true, "create": true, "alter": true, "drop": true,
	"grant": true, "revoke": true, "comment": true, "copy": true,
	"call": true, "do": true, "vacuum": true,
	"replace": true, "load": true, "rename": true, "lock": true,
	"unlock": true, "set": true, "reset": true, "into": true,
	"share": true, "begin": true, "start": true, "commit": true,
	"rollback": true, "savepoint": true, "release": true,
	"prepare": true, "execute": true, "deallocate": true,
	"reindex": true, "cluster": true, "refresh": true, "discard": true,
}

var sqlReadStarts = map[string]bool{
	"select": true, "show": true, "describe": true, "desc": true,
	"explain": true, "values": true, "table": true, "with": true,
}

// CheckSQL applies the conservative read-only SQL blocklist.
func CheckSQL(statement string) Decision {
	return CheckSQLMode(statement, true)
}

// CheckSQLMode validates one statement. Read-only also blocks write/admin
// keywords and non-read statements; read-write allows them but still rejects
// multiple statements and malformed quoting.
func CheckSQLMode(statement string, readOnly bool) Decision {
	clean, multi, err := stripSQL(statement)
	if err != nil {
		return Decision{Reason: err.Error()}
	}
	if multi {
		return Decision{Reason: "multiple statements are not allowed"}
	}
	tokens := sqlTokens(clean)
	if len(tokens) == 0 {
		return Decision{Reason: "statement is empty"}
	}
	if !readOnly {
		return Decision{Allowed: true}
	}
	for _, token := range tokens {
		if sqlWriteTokens[token] {
			return Decision{Reason: "write statements are blocked in read-only mode", ModeHint: true}
		}
	}
	if !sqlReadStarts[tokens[0]] {
		return Decision{Reason: "not a read-only statement", ModeHint: true}
	}
	return Decision{Allowed: true}
}

// stripSQL removes comments, quoted strings, and quoted identifiers while
// detecting a second non-empty statement. A single trailing semicolon is fine.
func stripSQL(statement string) (string, bool, error) {
	var b strings.Builder
	runes := []rune(statement)
	statements := 0
	hasContent := false
	for i := 0; i < len(runes); {
		r := runes[i]
		switch {
		case r == '-' && i+1 < len(runes) && runes[i+1] == '-':
			for i < len(runes) && runes[i] != '\n' {
				i++
			}
		case r == '/' && i+1 < len(runes) && runes[i+1] == '*':
			i += 2
			closed := false
			for i+1 < len(runes) {
				if runes[i] == '*' && runes[i+1] == '/' {
					i += 2
					closed = true
					break
				}
				i++
			}
			if !closed {
				return "", false, fmt.Errorf("unterminated comment")
			}
			b.WriteRune(' ')
		case r == '\'' || r == '"' || r == '`':
			quote := r
			closed := false
			i++
			for i < len(runes) {
				if runes[i] == '\\' {
					i += 2
					continue
				}
				if runes[i] == quote {
					if i+1 < len(runes) && runes[i+1] == quote {
						i += 2
						continue
					}
					i++
					closed = true
					break
				}
				i++
			}
			if !closed {
				return "", false, fmt.Errorf("unterminated quoted text")
			}
			b.WriteRune(' ')
		case r == ';':
			if hasContent {
				statements++
				hasContent = false
			}
			i++
		default:
			if !unicode.IsSpace(r) {
				hasContent = true
			}
			b.WriteRune(r)
			i++
		}
	}
	if hasContent {
		statements++
	}
	return b.String(), statements > 1, nil
}

func sqlTokens(statement string) []string {
	fields := strings.FieldsFunc(statement, func(r rune) bool {
		return !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_')
	})
	for i := range fields {
		fields[i] = strings.ToLower(fields[i])
	}
	return fields
}

var mongoWriteStages = map[string]bool{
	"$out": true, "$merge": true,
}

var mongoScriptTokens = map[string]bool{
	"$where": true, "$function": true, "$accumulator": true,
}

// CheckMongo parses the statement as one of the four read commands and rejects
// write stages and arbitrary JavaScript.
func CheckMongo(statement string) Decision {
	return CheckMongoMode(statement, true)
}

// CheckMongoMode always enforces the restricted four-command surface and the
// JavaScript ban. Write stages are additionally blocked while read-only.
func CheckMongoMode(statement string, readOnly bool) Decision {
	command, err := mql.Parse(statement)
	if err != nil {
		return Decision{Reason: err.Error()}
	}
	if reason, modeHint := mongoValueReason(readOnly, command.Filter, command.Pipeline, command.Projection, command.Sort); reason != "" {
		return Decision{Reason: reason, ModeHint: modeHint}
	}
	return Decision{Allowed: true}
}

func mongoValueReason(readOnly bool, values ...any) (string, bool) {
	for _, value := range values {
		if reason, modeHint := walkMongoValue(value, readOnly); reason != "" {
			return reason, modeHint
		}
	}
	return "", false
}

func walkMongoValue(value any, readOnly bool) (string, bool) {
	switch v := value.(type) {
	case mql.Object:
		for _, pair := range v {
			if readOnly && mongoWriteStages[pair.Key] {
				return "write stages are blocked in read-only mode", true
			}
			if mongoScriptTokens[pair.Key] {
				return "arbitrary JavaScript is not allowed", false
			}
			if reason, modeHint := walkMongoValue(pair.Value, readOnly); reason != "" {
				return reason, modeHint
			}
		}
	case mql.Array:
		for _, item := range v {
			if reason, modeHint := walkMongoValue(item, readOnly); reason != "" {
				return reason, modeHint
			}
		}
	case []any:
		for _, item := range v {
			if reason, modeHint := walkMongoValue(item, readOnly); reason != "" {
				return reason, modeHint
			}
		}
	}
	return "", false
}
