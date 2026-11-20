package profile

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"unicode"

	"github.com/mcgru/db-adm-tui/internal/domain"
)

// MongoEntry is the non-secret view of a MongoDB URI discovered in a
// ~/.mongo.* file. Password is populated only for connection resolution and
// must never be persisted or rendered.
type MongoEntry struct {
	Host     string
	Port     int
	Database string
	User     string
	Password string
	Selector string
	Line     int
}

// MongoFile is the result of reading one MongoDB connection file. Any
// malformed mongodb:// line makes the file unusable; the user fixes the file.
type MongoFile struct {
	Path    string
	Entry   *MongoEntry
	Missing bool
	Err     string
}

// ParseMongoFile reads one ~/.mongo.* file. Comment lines starting with # are
// ignored and the last valid mongodb:// URI wins.
func ParseMongoFile(path string) MongoFile {
	result := MongoFile{Path: path}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			result.Missing = true
			result.Err = "file not found"
		} else {
			result.Err = "file is unreadable"
		}
		return result
	}
	return ParseMongoBytes(path, string(data))
}

// ParseMongoBytes parses MongoDB connection-file content.
func ParseMongoBytes(path, data string) MongoFile {
	result := MongoFile{Path: path}
	var last *MongoEntry
	var lastErr string
	for i, raw := range strings.Split(data, "\n") {
		lineNo := i + 1
		line := strings.TrimSpace(strings.TrimSuffix(raw, "\r"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		uri, ok := extractMongoURI(line)
		if !ok {
			continue
		}
		entry, err := parseMongoURI(uri, lineNo)
		if err != nil {
			lastErr = fmt.Sprintf("line %d: %s", lineNo, err)
			continue
		}
		last = &entry
	}
	if lastErr != "" {
		result.Err = lastErr
		return result
	}
	if last != nil {
		result.Entry = last
		return result
	}
	if strings.Contains(data, "mongodb+srv://") {
		result.Err = "unsupported scheme mongodb+srv"
		return result
	}
	result.Err = "no mongodb:// URI found"
	return result
}

// MatchMongo reports whether an entry describes the same connection tuple as a
// profile. Host comparison is case-insensitive; the password is not compared.
func MatchMongo(entry MongoEntry, host string, port int, database, user string) bool {
	return strings.EqualFold(entry.Host, host) &&
		entry.Port == port &&
		entry.Database == database &&
		entry.User == user
}

func extractMongoURI(line string) (string, bool) {
	idx := strings.Index(line, "mongodb://")
	if idx < 0 {
		return "", false
	}
	rest := line[idx:]
	for i, r := range rest {
		if unicode.IsSpace(r) || r == '\'' || r == '"' || r == '`' || r == ')' {
			return rest[:i], true
		}
	}
	return rest, true
}

func parseMongoURI(raw string, line int) (MongoEntry, error) {
	const prefix = "mongodb://"
	if !strings.HasPrefix(raw, prefix) {
		return MongoEntry{}, fmt.Errorf("invalid URI")
	}
	rest := strings.TrimPrefix(raw, prefix)

	userinfo := ""
	if at := strings.Index(rest, "@"); at >= 0 {
		if slash := strings.Index(rest, "/"); slash < 0 || at < slash {
			userinfo = rest[:at]
			rest = rest[at+1:]
		}
	}

	authority := rest
	tail := ""
	if i := strings.IndexAny(rest, "/?"); i >= 0 {
		authority = rest[:i]
		tail = rest[i:]
	}
	if authority == "" {
		return MongoEntry{}, fmt.Errorf("URI has no host")
	}
	host, port, err := splitMongoHost(strings.Split(authority, ",")[0])
	if err != nil {
		return MongoEntry{}, err
	}
	if host == "" {
		return MongoEntry{}, fmt.Errorf("URI has no host")
	}

	path := tail
	query := ""
	if i := strings.Index(path, "?"); i >= 0 {
		query = path[i:]
		path = path[:i]
	}
	database := strings.Trim(path, "/")
	if database == "" {
		return MongoEntry{}, fmt.Errorf("URI has no database")
	}
	if strings.Contains(database, "/") {
		return MongoEntry{}, fmt.Errorf("URI database path is not valid")
	}

	entry := MongoEntry{Host: host, Port: port, Database: database, Line: line}
	if userinfo != "" {
		userPart, passwordPart, hasPassword := strings.Cut(userinfo, ":")
		entry.User = decodeMongoUserInfo(userPart)
		if hasPassword {
			entry.Password = decodeMongoUserInfo(passwordPart)
		}
	}
	selector := prefix
	if entry.User != "" {
		selector += url.User(entry.User).String() + "@"
	}
	selector += authority + path + query
	entry.Selector = domain.SanitizeMongoSelector(selector)
	return entry, nil
}

// ParseMongoHosts accepts a comma-separated host[:port] list and returns
// canonical host:port entries. A missing port uses the MongoDB default 27017.
func ParseMongoHosts(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("the host list is empty")
	}
	parts := strings.Split(raw, ",")
	hosts := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, fmt.Errorf("the host list contains an empty host")
		}
		host, port, err := splitMongoHost(part)
		if err != nil {
			return nil, fmt.Errorf("the host list is not valid")
		}
		hosts = append(hosts, net.JoinHostPort(host, strconv.Itoa(port)))
	}
	return hosts, nil
}

// MongoSelectorHosts returns the canonical host:port list from a MongoDB
// selector URI, or nil when the selector is absent or malformed.
func MongoSelectorHosts(selector string) []string {
	raw := strings.TrimSpace(selector)
	if !strings.HasPrefix(raw, "mongodb://") {
		return nil
	}
	rest := strings.TrimPrefix(raw, "mongodb://")
	if at := strings.Index(rest, "@"); at >= 0 {
		if slash := strings.Index(rest, "/"); slash < 0 || at < slash {
			rest = rest[at+1:]
		}
	}
	if index := strings.IndexAny(rest, "/?"); index >= 0 {
		rest = rest[:index]
	}
	hosts, err := ParseMongoHosts(rest)
	if err != nil {
		return nil
	}
	return hosts
}

func decodeMongoUserInfo(value string) string {
	decoded, err := url.PathUnescape(value)
	if err != nil {
		return value
	}
	return decoded
}

func splitMongoHost(raw string) (string, int, error) {
	if h, p, err := net.SplitHostPort(raw); err == nil {
		n, convErr := strconv.Atoi(p)
		if convErr != nil || n < 1 || n > 65535 {
			return "", 0, fmt.Errorf("URI port is not valid")
		}
		return h, n, nil
	}
	if strings.Count(raw, ":") == 1 {
		parts := strings.SplitN(raw, ":", 2)
		if parts[0] == "" {
			return "", 0, fmt.Errorf("URI has no host")
		}
		n, convErr := strconv.Atoi(parts[1])
		if convErr != nil || n < 1 || n > 65535 {
			return "", 0, fmt.Errorf("URI port is not valid")
		}
		return parts[0], n, nil
	}
	if raw == "" {
		return "", 0, fmt.Errorf("URI has no host")
	}
	return strings.Trim(raw, "[]"), 27017, nil
}
