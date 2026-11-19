package profile

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// HomeDir returns the current user's home directory.
func HomeDir() (string, error) {
	return os.UserHomeDir()
}

// ExpandPath expands a leading "~" using the current home directory.
func ExpandPath(path string) (string, error) {
	if path == "" {
		return "", nil
	}
	if path == "~" {
		return HomeDir()
	}
	if strings.HasPrefix(path, "~/") {
		home, err := HomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, path[2:]), nil
	}
	return path, nil
}

// DisplayPath replaces the home directory prefix with "~".
func DisplayPath(path string) string {
	if path == "" {
		return ""
	}
	home, err := HomeDir()
	if err != nil || home == "" {
		return path
	}
	if path == home {
		return "~"
	}
	if strings.HasPrefix(path, home+string(filepath.Separator)) {
		return "~" + path[len(home):]
	}
	return path
}

// ConfigDir returns $XDG_CONFIG_HOME/db-adm-tui, defaulting to
// ~/.config/db-adm-tui. Relative XDG values are ignored as the spec requires an
// absolute base directory.
func ConfigDir() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" && filepath.IsAbs(xdg) {
		return filepath.Join(xdg, "db-adm-tui"), nil
	}
	home, err := HomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "db-adm-tui"), nil
}

// PgpassPath returns the psql-compatible password file path. PGPASSFILE wins
// over ~/.pgpass.
func PgpassPath() (string, error) {
	if v := os.Getenv("PGPASSFILE"); v != "" {
		return ExpandPath(v)
	}
	home, err := HomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".pgpass"), nil
}

// MyPassPath returns the MySQL/MariaDB password file path. MYPASSFILE wins
// over ~/.mypass.
func MyPassPath() (string, error) {
	if v := os.Getenv("MYPASSFILE"); v != "" {
		return ExpandPath(v)
	}
	home, err := HomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".mypass"), nil
}

// MongoFiles returns every regular file matching ~/.mongo.* plus ~/.mongo.
// Results are sorted so discovery is deterministic.
func MongoFiles() ([]string, error) {
	home, err := HomeDir()
	if err != nil {
		return nil, err
	}
	return MongoFilesIn(home), nil
}

// MongoFilesIn returns every regular file matching <home>/.mongo.* plus
// <home>/.mongo, sorted.
func MongoFilesIn(home string) []string {
	seen := map[string]bool{}
	var files []string
	add := func(path string) {
		if seen[path] {
			return
		}
		info, statErr := os.Stat(path)
		if statErr != nil || info.IsDir() {
			return
		}
		seen[path] = true
		files = append(files, path)
	}
	matches, _ := filepath.Glob(filepath.Join(home, ".mongo.*"))
	for _, match := range matches {
		add(match)
	}
	add(filepath.Join(home, ".mongo"))
	sort.Strings(files)
	return files
}
