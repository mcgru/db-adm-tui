package profile

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mcgru/db-adm-tui/internal/domain"
)

// CredentialTarget describes one native account whose password should be
// written to a local credential source. Password is never persisted anywhere
// except the source file.
type CredentialTarget struct {
	Engine            domain.Engine
	Host              string
	Port              int
	Database          string
	User              string
	Password          string
	MongoHosts        string
	MongoAuthDatabase string
	MongoOptions      string
	// SourcePath is an optional profile source path (for example a custom
	// PGPASSFILE). It is expanded and used instead of the default path.
	SourcePath string
}

// SaveCredential writes or rotates one account in the engine's credential file
// and returns the source pointer for the corresponding profile.
func (r *Resolver) SaveCredential(target CredentialTarget) (domain.Source, error) {
	if strings.TrimSpace(target.User) == "" {
		return domain.Source{}, credentialError("the user name is empty")
	}
	if target.Password == "" {
		return domain.Source{}, credentialError("the password is empty")
	}
	if strings.ContainsAny(target.Password, "\n\r") ||
		strings.ContainsAny(target.Host, "\n\r") ||
		strings.ContainsAny(target.Database, "\n\r") {
		return domain.Source{}, credentialError("credential fields must not contain line breaks")
	}

	switch target.Engine {
	case domain.EnginePostgreSQL:
		path, err := r.credentialPath(r.pgpassPath, target.SourcePath)
		if err != nil {
			return domain.Source{}, err
		}
		return r.savePasswordLine(path, target, domain.SourcePgpass)
	case domain.EngineMySQL:
		path, err := r.credentialPath(r.mypassPath, target.SourcePath)
		if err != nil {
			return domain.Source{}, err
		}
		return r.savePasswordLine(path, target, domain.SourceMySQLPass)
	case domain.EngineMongoDB:
		return r.saveMongoCredential(target)
	default:
		return domain.Source{}, credentialError("this engine has no credential file")
	}
}

func (r *Resolver) credentialPath(defaultPath, sourcePath string) (string, error) {
	if strings.TrimSpace(sourcePath) == "" {
		return defaultPath, nil
	}
	path, err := r.expand(sourcePath)
	if err != nil {
		return "", credentialError("the credential source path is not valid")
	}
	return path, nil
}

func (r *Resolver) savePasswordLine(path string, target CredentialTarget, kind domain.SourceKind) (domain.Source, error) {
	entry, err := formatPasswordLine(target)
	if err != nil {
		return domain.Source{}, err
	}
	lines, err := readCredentialLines(path)
	if err != nil {
		return domain.Source{}, err
	}
	match := -1
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		fields, reason := splitPgpassLine(line)
		if reason != "" || len(fields) != 5 {
			continue
		}
		if fields[0] == target.Host &&
			fields[1] == strconv.Itoa(target.Port) &&
			fields[2] == target.Database &&
			fields[3] == target.User {
			match = i
			break
		}
	}
	lines = rotateLine(lines, match, entry)
	if err := writeCredentialLines(path, lines); err != nil {
		return domain.Source{}, err
	}
	return domain.Source{Kind: kind, Path: r.display(path)}, nil
}

func formatPasswordLine(target CredentialTarget) (string, error) {
	fields := []string{
		target.Host,
		strconv.Itoa(target.Port),
		target.Database,
		target.User,
		target.Password,
	}
	for i, field := range fields {
		escaped, err := escapePasswordField(field)
		if err != nil {
			return "", err
		}
		fields[i] = escaped
	}
	return strings.Join(fields, ":"), nil
}

func escapePasswordField(value string) (string, error) {
	if strings.ContainsAny(value, "\n\r") {
		return "", credentialError("credential fields must not contain line breaks")
	}
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, ":", `\:`)
	return value, nil
}

func rotateLine(lines []string, match int, newLine string) []string {
	if len(lines) == 0 {
		return []string{newLine, ""}
	}
	if match >= 0 && match < len(lines) {
		lines[match] = "#" + lines[match]
		lines = append(lines[:match+1], append([]string{newLine}, lines[match+1:]...)...)
		return lines
	}
	if lines[len(lines)-1] == "" {
		lines = append(lines[:len(lines)-1], newLine, "")
		return lines
	}
	return append(lines, newLine, "")
}

func (r *Resolver) saveMongoCredential(target CredentialTarget) (domain.Source, error) {
	if strings.TrimSpace(target.Database) == "" {
		return domain.Source{}, credentialError("the MongoDB database is empty")
	}
	filenameDatabase := strings.TrimSpace(target.MongoAuthDatabase)
	if filenameDatabase == "" {
		filenameDatabase = target.Database
	}
	path := filepath.Join(r.home,
		".mongo."+safeCredentialName(target.User)+"-in-"+safeCredentialName(filenameDatabase))
	uri := mongoCredentialURI(target)
	lines, err := readCredentialLines(path)
	if err != nil {
		return domain.Source{}, err
	}
	match := -1
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		raw, ok := extractMongoURI(line)
		if !ok {
			continue
		}
		entry, err := parseMongoURI(raw, i+1)
		if err != nil {
			continue
		}
		if entry.User == target.User && entry.Database == target.Database &&
			strings.EqualFold(entry.Host, target.Host) && entry.Port == target.Port {
			match = i
		}
	}
	lines = rotateLine(lines, match, uri)
	if err := writeCredentialLines(path, lines); err != nil {
		return domain.Source{}, err
	}
	return domain.Source{Kind: domain.SourceMongoFile, Path: r.display(path)}, nil
}

func mongoCredentialURI(target CredentialTarget) string {
	authority := strings.TrimSpace(target.MongoHosts)
	if authority == "" {
		authority = net.JoinHostPort(target.Host, strconv.Itoa(target.Port))
	}
	u := url.URL{
		Scheme: "mongodb",
		Host:   authority,
		Path:   "/" + target.Database,
	}
	u.User = url.UserPassword(target.User, target.Password)
	if target.MongoOptions != "" {
		u.RawQuery = target.MongoOptions
	}
	return u.String()
}

func safeCredentialName(value string) string {
	value = strings.ReplaceAll(value, "/", "_")
	value = strings.ReplaceAll(value, `\`, "_")
	value = strings.ReplaceAll(value, "..", "_")
	if value == "" {
		return "unknown"
	}
	return value
}

func readCredentialLines(path string) ([]string, error) {
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, credentialError("the credential file is unreadable")
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, credentialError("the credential path is a symbolic link")
	}
	if info.IsDir() {
		return nil, credentialError("the credential path is a directory")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, credentialError("the credential file is unreadable")
	}
	return strings.Split(string(data), "\n"), nil
}

func writeCredentialLines(path string, lines []string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return credentialError("the credential directory cannot be created")
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return credentialError("the credential directory permissions cannot be set")
	}
	data := strings.Join(lines, "\n")
	if !strings.HasSuffix(data, "\n") {
		data += "\n"
	}
	tmp, err := os.CreateTemp(dir, ".credential-*.tmp")
	if err != nil {
		return credentialError("the credential file cannot be written")
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return credentialError("the credential file permissions cannot be set")
	}
	if _, err := tmp.WriteString(data); err != nil {
		tmp.Close()
		return credentialError("the credential file cannot be written")
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return credentialError("the credential file cannot be synced")
	}
	if err := tmp.Close(); err != nil {
		return credentialError("the credential file cannot be closed")
	}
	if err := os.Rename(tmpName, path); err != nil {
		return credentialError("the credential file cannot be replaced")
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return credentialError("the credential file permissions cannot be set")
	}
	return nil
}

func credentialError(reason string) error {
	return domain.NewError(domain.KindConfig, "credential", fmt.Sprintf("Could not save credentials: %s.", reason))
}
