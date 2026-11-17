package web

import (
	"strings"

	"github.com/microcosm-cc/bluemonday"
)

// strictPolicy is the empty safelist. It strips every HTML element.
var strictPolicy = bluemonday.StrictPolicy()

// LooksUnsafe reports markup in longUrl. bluemonday rewrites a query "&" to "&amp;";
// that rewrite alone is not an HTML element, so a normal URL stays allowed.
func LooksUnsafe(value string) bool {
	if strings.TrimSpace(value) == "" {
		return false
	}
	sanitized := strictPolicy.Sanitize(value)
	if sanitized == value {
		return false
	}
	return strings.ReplaceAll(sanitized, "&amp;", "&") != value
}
