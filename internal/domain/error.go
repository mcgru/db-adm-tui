package domain

import (
	"errors"
	"net/url"
	"regexp"
	"strings"
)

// ErrorKind classifies an engine or application error. The TUI maps the kind
// to copy; it never branches on error strings.
type ErrorKind string

const (
	KindAuth       ErrorKind = "auth"
	KindPermission ErrorKind = "permission"
	KindNotFound   ErrorKind = "not_found"
	KindTimeout    ErrorKind = "timeout"
	KindCancelled  ErrorKind = "cancelled"
	KindGuard      ErrorKind = "guard"
	KindEngine     ErrorKind = "engine"
	KindConfig     ErrorKind = "config"
)

// Error is the safe error value crossing adapter boundaries. Message is always
// safe to render: adapters redact driver text and secrets before creating it.
type Error struct {
	Kind    ErrorKind
	Op      string
	Profile string
	Source  string
	Message string
	Err     error
}

// Error implements error.
func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

// Unwrap exposes the wrapped cause for errors.Is/errors.As.
func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// NewError builds a safe domain error.
func NewError(kind ErrorKind, op, message string) *Error {
	return &Error{Kind: kind, Op: op, Message: message}
}

// KindOf returns the domain error kind of err, or KindEngine when it is not a
// domain error.
func KindOf(err error) ErrorKind {
	var de *Error
	if errors.As(err, &de) && de.Kind != "" {
		return de.Kind
	}
	return KindEngine
}

var (
	uriUserinfo      = regexp.MustCompile(`([A-Za-z][A-Za-z0-9+.-]*://[^/@\s]*):[^@/\s]*@`)
	passwordKV       = regexp.MustCompile(`(?i)(password|passwd|pwd)\s*=\s*[^\s&;]+`)
	passwordJSONPair = regexp.MustCompile(`(?i)("(?:password|passwd|pwd)"\s*:\s*)"(?:[^"\\]|\\.)*"`)
	passwordLiteral  = regexp.MustCompile(`(?i)\b(password|passwd|pwd|identified\s+by)\s*(?:=|:)?\s*'(?:[^'\\]|\\.)*'`)
	passwordDouble   = regexp.MustCompile(`(?i)(^|[^"])\b(password|passwd|pwd|identified\s+by)\s*(?:=|:)\s*"(?:[^"\\]|\\.)*"`)
)

// Redact removes URI userinfo passwords and password key/value pairs from s.
func Redact(s string) string {
	s = uriUserinfo.ReplaceAllString(s, "$1:***@")
	s = passwordKV.ReplaceAllString(s, "$1=***")
	return s
}

// RedactSensitive also removes quoted password literals and JSON password
// pairs. It is used for history, query errors, and driver error text.
func RedactSensitive(s string) string {
	s = Redact(s)
	s = passwordJSONPair.ReplaceAllString(s, `$1"***"`)
	s = passwordLiteral.ReplaceAllString(s, "$1 '***'")
	s = passwordDouble.ReplaceAllString(s, `$1$2: "***"`)
	return s
}

// RedactSecrets applies RedactSensitive and then removes every occurrence of a
// known connection secret, including its URL-encoded forms.
func RedactSecrets(s, secret string) string {
	return RedactSecret(RedactSensitive(s), secret)
}

// RedactSecret removes every occurrence of secret from s, including its
// URL-encoded forms. It is the last line of defence for driver error text.
func RedactSecret(s, secret string) string {
	if secret == "" {
		return s
	}
	s = strings.ReplaceAll(s, secret, "***")
	for _, encoded := range []string{url.QueryEscape(secret), url.PathEscape(secret)} {
		if encoded != secret {
			s = strings.ReplaceAll(s, encoded, "***")
		}
	}
	return s
}
