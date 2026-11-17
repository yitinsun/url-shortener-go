package web

import "time"

// ShortenRequest is the create body. ttlInDays is optional; omitted means 730 days.
type ShortenRequest struct {
	LongURL   string `json:"longUrl" example:"https://example.com/api-design/api-design-url-shortening-service/"`
	TTLInDays *int64 `json:"ttlInDays" example:"365" minimum:"1" maximum:"3650"`
}

// ShortenResponse is the 201 body.
type ShortenResponse struct {
	LongURL   string    `json:"longUrl" example:"https://example.com/api-design/api-design-url-shortening-service/"`
	ShortURL  string    `json:"shortUrl" example:"http://localhost:8080/short/b9K2x1"`
	ShortCode string    `json:"shortCode" example:"b9K2x1" minLength:"5" maxLength:"7"`
	ExpiresAt time.Time `json:"expiresAt" example:"2027-11-12T09:25:00Z"`
}

// ErrorResponse is the JSON body for 400 and 503. shortCode is null on those responses.
type ErrorResponse struct {
	Status    int     `json:"status" example:"400"`
	Error     string  `json:"error" example:"Bad Request"`
	Message   string  `json:"message" example:"longUrl is not a valid URI"`
	ShortCode *string `json:"shortCode" swaggertype:"string"`
}
