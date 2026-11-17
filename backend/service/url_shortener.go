package service

import (
	"context"
	"net/url"
	"strings"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"url-shortener/apperror"
	"url-shortener/encoding"
	"url-shortener/repo"
)

const (
	maxURLLength   = 2048
	defaultTTLDays = 730
	minTTLDays     = 1
	maxTTLDays     = 3650
)

// Result is a successful create response before JSON encoding.
type Result struct {
	LongURL   string
	ShortURL  string
	ShortCode string
	ExpiresAt time.Time
}

// Service is the write and redirect path.
type Service struct {
	repo    repo.Repository
	baseURL string
}

func New(store repo.Repository, baseURL string) *Service {
	return &Service{repo: store, baseURL: strings.TrimRight(baseURL, "/")}
}

func (s *Service) BaseURL() string { return s.baseURL }

func (s *Service) Create(ctx context.Context, longURL string, ttlInDays *int64, now time.Time) (Result, error) {
	longURL, err := validateLongURL(longURL, s.baseURL)
	if err != nil {
		return Result{}, err
	}
	ttlDays, err := resolveTTL(ttlInDays)
	if err != nil {
		return Result{}, err
	}
	now = now.UTC()
	prefix := prefixOf(longURL)
	existing, err := s.repo.FindByLongURLPrefix(ctx, prefix)
	if err != nil {
		return Result{}, err
	}
	for _, row := range existing {
		if row.LongURL == longURL && row.ExpiresAt.After(now) {
			return s.toResult(row), nil
		}
	}
	expiresAt := now.Add(time.Duration(ttlDays) * 24 * time.Hour)
	mapping, err := s.repo.InsertWithEncodedID(ctx, longURL, placeholderCode(), now, expiresAt, encoding.Encode)
	if err != nil {
		return Result{}, err
	}
	return s.toResult(mapping), nil
}

func (s *Service) Resolve(ctx context.Context, shortCode string, now time.Time) (string, error) {
	mapping, err := s.repo.FindByShortCode(ctx, shortCode)
	if err != nil {
		return "", err
	}
	if mapping == nil || !mapping.ExpiresAt.After(now.UTC()) {
		return "", apperror.NotFound()
	}
	return mapping.LongURL, nil
}

func (s *Service) toResult(mapping repo.URLMapping) Result {
	return Result{
		LongURL:   mapping.LongURL,
		ShortURL:  s.baseURL + "/short/" + mapping.ShortCode,
		ShortCode: mapping.ShortCode,
		ExpiresAt: mapping.ExpiresAt.UTC(),
	}
}

func validateLongURL(longURL string, baseURL string) (string, error) {
	if strings.TrimSpace(longURL) == "" {
		return "", apperror.BadRequest("longUrl is required")
	}
	trimmed := strings.TrimSpace(longURL)
	if utf8.RuneCountInString(trimmed) > maxURLLength {
		return "", apperror.BadRequest("longUrl exceeds 2048 characters")
	}
	lower := strings.ToLower(trimmed)
	if strings.Contains(lower, "<script") || strings.HasPrefix(lower, "javascript:") || strings.HasPrefix(lower, "data:") {
		return "", apperror.BadRequest("longUrl contains a disallowed payload")
	}
	parsed, err := url.ParseRequestURI(trimmed)
	if err != nil {
		return "", apperror.BadRequest("longUrl is not a valid URI")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", apperror.BadRequest("longUrl must be HTTP or HTTPS")
	}
	host := parsed.Hostname()
	if host == "" {
		return "", apperror.BadRequest("longUrl must include a host")
	}
	base, err := url.ParseRequestURI(baseURL)
	if err != nil {
		return "", apperror.Unavailable("APP_BASE_URL is invalid")
	}
	if appHost := base.Hostname(); appHost != "" && strings.EqualFold(host, appHost) {
		return "", apperror.BadRequest("longUrl must not point at this service")
	}
	return trimmed, nil
}

func resolveTTL(ttlInDays *int64) (int64, error) {
	if ttlInDays == nil {
		return defaultTTLDays, nil
	}
	days := *ttlInDays
	if days < minTTLDays || days > maxTTLDays {
		return 0, apperror.BadRequest("ttlInDays must be between 1 and 3650")
	}
	return days, nil
}

func prefixOf(longURL string) string {
	runes := []rune(longURL)
	if len(runes) > 255 {
		return string(runes[:255])
	}
	return longURL
}

var placeholderSeq atomic.Uint64

func placeholderCode() string {
	n := placeholderSeq.Add(1)
	mixed := n * 0x9E3779B97F4A7C15
	code := encoding.Encode(mixed)
	if len(code) > 10 {
		code = code[:10]
	}
	for len(code) < 10 {
		code = "0" + code
	}
	return code
}
