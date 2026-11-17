package web

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"url-shortener/apperror"
	"url-shortener/service"
)

// NewRouter is the greenfield HTTP surface: create and redirect.
func NewRouter(svc *service.Service) chi.Router {
	r := chi.NewRouter()
	r.NotFound(friendlyMiss)
	r.MethodNotAllowed(friendlyMiss)
	r.Post("/api/v1/shorten", shorten(svc))
	r.Get("/short/{shortCode}", redirect(svc))
	return r
}

// shorten creates a short URL.
//
// @Summary Create a short URL
// @Description Mapping created, or an existing active mapping returned
// @Tags Shorten
// @Accept json
// @Produce json
// @Param request body ShortenRequest true "Create a short URL"
// @Success 201 {object} ShortenResponse "Mapping created, or an existing active mapping returned"
// @Failure 400 {object} ErrorResponse "Invalid body (URL, TTL, script payload, or self-loop)"
// @Failure 503 {object} ErrorResponse "Database unavailable"
// @Router /api/v1/shorten [post]
func shorten(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req ShortenRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			apperror.Write(w, apperror.BadRequest("request body is not valid JSON"))
			return
		}
		created, err := svc.Create(r.Context(), req.LongURL, req.TTLInDays, time.Now().UTC())
		if err != nil {
			apperror.Write(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, ShortenResponse{
			LongURL:   created.LongURL,
			ShortURL:  created.ShortURL,
			ShortCode: created.ShortCode,
			ExpiresAt: created.ExpiresAt,
		})
	}
}

func redirect(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := chi.URLParam(r, "shortCode")
		longURL, err := svc.Resolve(r.Context(), code, time.Now().UTC())
		if err != nil {
			if apperror.IsNotFound(err) {
				apperror.WriteNotFound(w, r.URL.Path)
				return
			}
			apperror.Write(w, err)
			return
		}
		http.Redirect(w, r, longURL, http.StatusFound)
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(payload)
}
