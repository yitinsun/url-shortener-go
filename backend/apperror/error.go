package apperror

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

type Kind int

const (
	KindBadRequest Kind = iota
	KindNotFound
	KindUnavailable
)

// Error is the single create-API failure type. NotFound is rendered as HTML by the redirect handler.
type Error struct {
	Kind Kind
	Msg  string
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Kind == KindNotFound {
		return "short link was not found"
	}
	return e.Msg
}

func BadRequest(message string) *Error {
	return &Error{Kind: KindBadRequest, Msg: message}
}

// DisallowedHTML is the 400 for HTML markup in longUrl.
func DisallowedHTML() *Error {
	return BadRequest("longUrl contains disallowed HTML")
}

func NotFound() *Error {
	return &Error{Kind: KindNotFound}
}

func Unavailable(message string) *Error {
	return &Error{Kind: KindUnavailable, Msg: message}
}

func IsBadRequest(err error) bool {
	var appErr *Error
	return errors.As(err, &appErr) && appErr.Kind == KindBadRequest
}

func IsNotFound(err error) bool {
	var appErr *Error
	return errors.As(err, &appErr) && appErr.Kind == KindNotFound
}

type body struct {
	Status    int     `json:"status"`
	Error     string  `json:"error"`
	Message   string  `json:"message"`
	ShortCode *string `json:"shortCode"`
}

// Write maps a service error onto the create-API JSON body.
func Write(w http.ResponseWriter, err error) {
	var appErr *Error
	if !errors.As(err, &appErr) {
		appErr = Unavailable(err.Error())
	}
	switch appErr.Kind {
	case KindBadRequest:
		writeJSON(w, http.StatusBadRequest, "Bad Request", appErr.Msg)
	case KindNotFound:
		WriteNotFound(w, "")
	default:
		writeJSON(w, http.StatusServiceUnavailable, "Service Unavailable", appErr.Msg)
	}
}

func writeJSON(w http.ResponseWriter, status int, kind string, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(body{Status: status, Error: kind, Message: message})
}

// WriteNotFound renders the friendly HTML page for a missing or expired code.
func WriteNotFound(w http.ResponseWriter, path string) {
	html := "<!DOCTYPE html><html lang=\"en\"><head><meta charset=\"utf-8\"><title>Link not found</title></head><body><h1>Link not found</h1><p>This short link is missing or has expired.</p><p>" +
		escapeHTML(path) +
		"</p><nav><a href=\"/api/v1/shorten\">POST /api/v1/shorten</a> <a href=\"/swagger-ui.html\">/swagger-ui.html</a> <a href=\"/short/{code}\">/short/{code}</a></nav></body></html>"
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write([]byte(html))
}

func escapeHTML(value string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
	)
	return replacer.Replace(value)
}
