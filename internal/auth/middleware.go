// Package auth checks the API key on every request that carries one.
package auth

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/MarkProminic/disk-agent/internal/keys"
	"github.com/MarkProminic/disk-agent/internal/logging"
	"github.com/MarkProminic/disk-agent/internal/problem"
)

func alog() *slog.Logger {
	return logging.Category("auth")
}

// ErrAmbiguous is two Authorization headers on one request.
var ErrAmbiguous = errors.New("multiple Authorization headers")

// ErrScheme is an Authorization scheme the agent does not read.
var ErrScheme = errors.New("unsupported Authorization scheme")

// Credential reads the request's credential: X-API-Key, else Authorization as Bearer.
func Credential(r *http.Request) (string, error) {
	if key := r.Header.Get("X-API-Key"); key != "" {
		return key, nil
	}
	values := r.Header.Values("Authorization")
	if len(values) > 1 {
		return "", ErrAmbiguous
	}
	if len(values) == 0 {
		return "", nil
	}
	name, token, found := strings.Cut(values[0], " ")
	token = strings.TrimSpace(token)
	if !found || token == "" || !strings.EqualFold(name, "Bearer") {
		return "", ErrScheme
	}
	return token, nil
}

func unauthorized(w http.ResponseWriter, detail string) {
	w.Header().Set("WWW-Authenticate", "Bearer")
	problem.Detail(w, http.StatusUnauthorized, detail)
}

// Middleware refuses every request without a valid API key: 401 missing, 403 unknown.
func Middleware(store *keys.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			credential, cerr := Credential(r)
			switch {
			case errors.Is(cerr, ErrAmbiguous):
				problem.Detail(w, http.StatusBadRequest, "Multiple methods used to include an access token")
				return
			case errors.Is(cerr, ErrScheme):
				unauthorized(w, "Authorization scheme must be Bearer")
				return
			case credential == "":
				unauthorized(w, "API key required - provide either X-API-Key header or Authorization: Bearer header")
				return
			}
			match, err := store.Verify(credential)
			if err != nil {
				alog().Error("api key validation failed", "error", err, "path", r.URL.Path)
				problem.Detail(w, http.StatusInternalServerError, "API key validation failed")
				return
			}
			if match == nil {
				problem.Detail(w, http.StatusForbidden, "Invalid API key")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
