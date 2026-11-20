package profile

import (
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/mcgru/db-adm-tui/internal/domain"
)

// UpdateCredential moves a file-backed credential from the old profile tuple to
// the edited one while preserving the password. The source file is rewritten
// atomically; the old file is kept when MongoDB needs a different canonical
// path. A prompt-backed profile is unchanged.
func (r *Resolver) UpdateCredential(old, next domain.Profile) (*domain.Source, error) {
	if old.Source == nil || old.Source.Kind == domain.SourcePrompt {
		return cloneSource(old.Source), nil
	}
	if sameConnection(old, next) {
		return cloneSource(old.Source), nil
	}
	switch old.Source.Kind {
	case domain.SourcePgpass, domain.SourceMySQLPass:
		return r.updatePasswordCredential(old, next)
	case domain.SourceMongoFile:
		return r.updateMongoCredential(old, next)
	default:
		return cloneSource(old.Source), nil
	}
}

func cloneSource(source *domain.Source) *domain.Source {
	if source == nil {
		return nil
	}
	copied := *source
	return &copied
}

func sameConnection(a, b domain.Profile) bool {
	return a.Engine == b.Engine &&
		strings.EqualFold(a.Host, b.Host) &&
		a.Port == b.Port &&
		a.Database == b.Database &&
		a.User == b.User
}

func (r *Resolver) updatePasswordCredential(old, next domain.Profile) (*domain.Source, error) {
	secret, _, err := r.ResolveSource(old)
	if err != nil {
		return nil, err
	}
	path, err := r.expand(old.Source.Path)
	if err != nil {
		return nil, credentialError("the credential source path is not valid")
	}
	lines, err := readCredentialLines(path)
	if err != nil {
		return nil, err
	}
	match := -1
	for i, line := range lines {
		fields, reason := splitPgpassLine(line)
		if reason != "" || len(fields) != 5 {
			continue
		}
		if passwordTupleMatches(fields, old.Host, old.Port, old.Database, old.User) {
			match = i
			break
		}
	}
	if match < 0 {
		return nil, credentialError("the source entry for the old connection was not found")
	}
	newLine, err := formatPasswordLine(CredentialTarget{
		Engine: old.Engine, Host: next.Host, Port: next.Port,
		Database: next.Database, User: next.User, Password: secret,
	})
	if err != nil {
		return nil, err
	}
	rewritten := make([]string, 0, len(lines)+2)
	inserted := false
	for i, line := range lines {
		if i == match {
			rewritten = append(rewritten, "#"+line, newLine)
			inserted = true
			continue
		}
		fields, reason := splitPgpassLine(line)
		if reason == "" && len(fields) == 5 &&
			passwordTupleMatches(fields, next.Host, next.Port, next.Database, next.User) {
			rewritten = append(rewritten, "#"+line)
			continue
		}
		rewritten = append(rewritten, line)
	}
	if !inserted {
		rewritten = append(rewritten, newLine)
	}
	if err := writeCredentialLines(path, rewritten); err != nil {
		return nil, err
	}
	return cloneSource(old.Source), nil
}

func passwordTupleMatches(fields []string, host string, port int, database, user string) bool {
	return fields[0] == host &&
		fields[1] == strconv.Itoa(port) &&
		fields[2] == database &&
		fields[3] == user
}

func (r *Resolver) updateMongoCredential(old, next domain.Profile) (*domain.Source, error) {
	secret, selector, err := r.ResolveSource(old)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(next.Database) == "" || strings.TrimSpace(next.User) == "" {
		return nil, credentialError("the MongoDB user or database is empty")
	}
	hosts := MongoSelectorHosts(selector)
	if len(hosts) == 0 {
		hosts = MongoSelectorHosts(old.Source.Selector)
	}
	hostChanged := !strings.EqualFold(next.Host, old.Host) || next.Port != old.Port
	if hostChanged || len(hosts) == 0 {
		if strings.Contains(next.Host, ",") {
			parsed, err := ParseMongoHosts(next.Host)
			if err != nil {
				return nil, credentialError("the MongoDB host list is not valid")
			}
			hosts = parsed
		} else {
			hosts = []string{net.JoinHostPort(next.Host, strconv.Itoa(next.Port))}
		}
	}
	host, port := splitMongoAddress(hosts[0])
	options := mongoEditOptions(selector, old.Source.Selector, next.Database)
	source, err := r.SaveCredential(CredentialTarget{
		Engine: domain.EngineMongoDB, Host: host, Port: port,
		Database: next.Database, User: next.User, Password: secret,
		MongoHosts: strings.Join(hosts, ","), MongoOptions: options,
	})
	if err != nil {
		return nil, err
	}
	oldPath, err := r.expand(old.Source.Path)
	if err != nil {
		return nil, credentialError("the credential source path is not valid")
	}
	lines, err := readCredentialLines(oldPath)
	if err != nil {
		return nil, err
	}
	changed := false
	for i, line := range lines {
		raw, ok := extractMongoURI(line)
		if !ok {
			continue
		}
		entry, err := parseMongoURI(raw, i+1)
		if err != nil {
			continue
		}
		if entry.User == old.User && entry.Database == old.Database &&
			strings.EqualFold(entry.Host, old.Host) && entry.Port == old.Port {
			lines[i] = "#" + line
			changed = true
			break
		}
	}
	if changed {
		if err := writeCredentialLines(oldPath, lines); err != nil {
			return nil, err
		}
	}
	return &source, nil
}

func splitMongoAddress(address string) (string, int) {
	if host, portText, err := net.SplitHostPort(address); err == nil {
		if port, convErr := strconv.Atoi(portText); convErr == nil {
			return host, port
		}
	}
	return address, 27017
}

// mongoEditOptions keeps the non-secret query options and rebinds authSource to
// the profile's selected authentication database.
func mongoEditOptions(selector, fallback, database string) string {
	values := url.Values{}
	for _, raw := range []string{selector, fallback} {
		raw = strings.TrimSpace(raw)
		index := strings.Index(raw, "?")
		if index < 0 || index+1 >= len(raw) {
			continue
		}
		parsed, err := url.ParseQuery(raw[index+1:])
		if err != nil {
			continue
		}
		values = parsed
		break
	}
	for key := range values {
		switch strings.ToLower(key) {
		case "password", "passwd", "pwd":
			values.Del(key)
		}
	}
	if database != "" {
		values.Set("authSource", database)
	}
	return values.Encode()
}
