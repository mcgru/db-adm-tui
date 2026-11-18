package app

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/mcgru/db-adm-tui/internal/domain"
)

// ParseURI validates a no-password connection URI and returns an unsaved
// profile. It rejects userinfo passwords, requires a host, user, and explicit
// database path, and fills the engine default port when none is given.
func ParseURI(raw, label string) (domain.Profile, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return domain.Profile{}, uriError("the URI is not valid")
	}
	u, err := url.Parse(raw)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "port") {
			return domain.Profile{}, uriError("the port is not valid")
		}
		return domain.Profile{}, uriError("the URI is not valid")
	}
	engine, ok := domain.ParseScheme(u.Scheme)
	if !ok {
		return domain.Profile{}, uriError("the engine is not supported")
	}
	if u.User != nil {
		if _, hasPassword := u.User.Password(); hasPassword {
			return domain.Profile{}, uriError("the URI must not contain a password")
		}
	}
	for key := range u.Query() {
		switch strings.ToLower(key) {
		case "password", "passwd", "pwd":
			return domain.Profile{}, uriError("the URI must not contain a password")
		}
	}
	if u.RawQuery != "" {
		// Query options are not persisted in schema v1. Rejecting them is
		// safer than silently dropping a TLS or authSource setting.
		return domain.Profile{}, uriError("URI options are not supported in this story")
	}
	host := u.Hostname()
	if host == "" {
		return domain.Profile{}, uriError("the URI is not valid")
	}
	port := engine.DefaultPort()
	if portText := u.Port(); portText != "" {
		n, convErr := strconv.Atoi(portText)
		if convErr != nil || n < 1 || n > 65535 {
			return domain.Profile{}, uriError("the port is not valid")
		}
		port = n
	}
	database := strings.TrimPrefix(u.Path, "/")
	if database == "" {
		return domain.Profile{}, uriError("the URI needs a database path")
	}
	if strings.Contains(database, "/") {
		return domain.Profile{}, uriError("the URI is not valid")
	}
	user := ""
	if u.User != nil {
		user = u.User.Username()
	}
	if user == "" {
		return domain.Profile{}, uriError("the URI needs a user")
	}
	label = strings.TrimSpace(label)
	if label == "" {
		label = domain.DefaultLabel(database, host)
	}
	return domain.Profile{
		Label:    label,
		Engine:   engine,
		Host:     host,
		Port:     port,
		Database: database,
		User:     user,
	}.Normalize(), nil
}

func uriError(reason string) error {
	return &domain.Error{Kind: domain.KindConfig, Op: "parse-uri", Message: reason}
}

// URIErrorMessage renders a URI validation error for the new-connection modal.
// It never includes the URI itself.
func URIErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	message := strings.TrimSpace(err.Error())
	if strings.HasPrefix(message, "Could not connect") {
		return message
	}
	message = strings.TrimSuffix(message, ".")
	return fmt.Sprintf("Could not connect: %s.", message)
}
