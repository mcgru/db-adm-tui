package profile

import (
	"os"
	"strconv"
	"strings"
)

// PgpassEntry is one parsed, non-secret view of a ~/.pgpass line.
type PgpassEntry struct {
	Host     string
	Port     string
	Database string
	User     string
	Password string
	Line     int
}

// PgpassWarning describes a line that psql-compatible parsing rejects. It
// never contains the password.
type PgpassWarning struct {
	Line   int
	Reason string
}

// PgpassFile is the result of reading one password file.
type PgpassFile struct {
	Path         string
	Entries      []PgpassEntry
	Warnings     []PgpassWarning
	Missing      bool
	Ignored      bool
	IgnoreReason string
	ReadError    string
}

// ParsePgpass reads path with psql semantics: a missing file is empty, a file
// with group/other permissions is ignored, # comments are skipped, backslash
// escapes the next character, and every other line must have five fields.
func ParsePgpass(path string) PgpassFile {
	result := PgpassFile{Path: path}
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			result.Missing = true
			return result
		}
		result.ReadError = "file is unreadable"
		return result
	}
	if info.IsDir() {
		result.ReadError = "path is a directory, not a file"
		return result
	}
	if info.Mode().Perm()&0o077 != 0 {
		result.Ignored = true
		result.IgnoreReason = "permissions are not 0600"
		return result
	}
	data, err := os.ReadFile(path)
	if err != nil {
		result.ReadError = "file is unreadable"
		return result
	}
	result.Entries, result.Warnings = ParsePgpassLines(string(data))
	return result
}

// ParsePgpassLines parses password-file content. It returns valid entries and
// per-line warnings for malformed lines. Password values are present in the
// entries because connection resolution needs them; callers must never persist
// or render them.
func ParsePgpassLines(data string) ([]PgpassEntry, []PgpassWarning) {
	lines := strings.Split(data, "\n")
	var entries []PgpassEntry
	var warnings []PgpassWarning
	for i, raw := range lines {
		lineNo := i + 1
		line := strings.TrimSuffix(raw, "\r")
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") {
			continue
		}
		fields, reason := splitPgpassLine(line)
		if reason != "" {
			warnings = append(warnings, PgpassWarning{Line: lineNo, Reason: reason})
			continue
		}
		port := fields[1]
		if port != "*" {
			n, err := strconv.Atoi(port)
			if err != nil || n < 1 || n > 65535 {
				warnings = append(warnings, PgpassWarning{Line: lineNo, Reason: "port is not a number or *"})
				continue
			}
		}
		entries = append(entries, PgpassEntry{
			Host:     fields[0],
			Port:     port,
			Database: fields[2],
			User:     fields[3],
			Password: fields[4],
			Line:     lineNo,
		})
	}
	return entries, warnings
}

// splitPgpassLine implements psql's colon/backslash field parsing. It returns
// exactly five fields or a safe reason.
func splitPgpassLine(line string) ([]string, string) {
	fields := make([]string, 0, 5)
	var current strings.Builder
	escaped := false
	flush := func() {
		fields = append(fields, current.String())
		current.Reset()
	}
	for _, r := range line {
		if escaped {
			current.WriteRune(r)
			escaped = false
			continue
		}
		switch r {
		case '\\':
			escaped = true
		case ':':
			flush()
			if len(fields) > 5 {
				return nil, "too many fields"
			}
		default:
			current.WriteRune(r)
		}
	}
	if escaped {
		return nil, "trailing backslash"
	}
	flush()
	if len(fields) != 5 {
		return nil, "expected 5 fields"
	}
	return fields, ""
}

// MatchPgpass returns the first entry matching the connection tuple. Host,
// database, user, and port accept "*".
func MatchPgpass(entries []PgpassEntry, host string, port int, database, user string) (PgpassEntry, bool) {
	portText := strconv.Itoa(port)
	for _, entry := range entries {
		if !pgpassMatch(entry.Host, host) || !pgpassMatch(entry.Port, portText) ||
			!pgpassMatch(entry.Database, database) || !pgpassMatch(entry.User, user) {
			continue
		}
		return entry, true
	}
	return PgpassEntry{}, false
}

func pgpassMatch(pattern, value string) bool {
	return pattern == "*" || pattern == value
}
