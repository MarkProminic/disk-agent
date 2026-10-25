package auth

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/Makr91/disk-agent/internal/keys"
	"github.com/Makr91/disk-agent/internal/logging"
	"github.com/Makr91/disk-agent/internal/problem"
)

// alog is this package's category logger; logging.categories.auth overrides
// its level.
func alog() *slog.Logger {
	return logging.Category("auth")
}

// Role hierarchy for the authorization model. Unknown roles compare as 0 and
// only pass checks requiring nothing.
var roleLevels = map[string]int{"viewer": 1, "operator": 2, "admin": 3}

// Surfaces that are admin-only regardless of method: key management and the
// configuration files.
var adminAlwaysPrefixes = []string{"/api/api-keys", "/api/config"}

func underPrefix(path string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if path == prefix || strings.HasPrefix(path, prefix+"/") {
			return true
		}
	}
	return false
}

// RequiredRole is the central method+path policy.
func RequiredRole(method, path string) string {
	if path == "/api/api-keys/info" {
		return "viewer"
	}
	if underPrefix(path, adminAlwaysPrefixes) {
		return "admin"
	}
	if method == http.MethodGet || method == http.MethodHead {
		return "viewer"
	}
	return "operator"
}

type contextKey struct{}

// Identity is the authenticated key attached to the request context.
type Identity struct {
	ID          int64
	Name        string
	Description string
	Role        string
}

// FromContext returns the authenticated identity, or nil on unauthenticated
// requests (public routes).
func FromContext(ctx context.Context) *Identity {
	id, _ := ctx.Value(contextKey{}).(*Identity)
	return id
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

// Unauthorized answers 401 with the challenge header and a problem body.
func Unauthorized(w http.ResponseWriter, detail string) {
	w.Header().Set("WWW-Authenticate", "Bearer")
	problem.Detail(w, http.StatusUnauthorized, detail)
}

// WriteMsg writes a problem body of the status's registry type with msg as its detail.
func WriteMsg(w http.ResponseWriter, status int, msg string) {
	problem.Detail(w, status, msg)
}

func identityOf(k *keys.Key) *Identity {
	return &Identity{ID: k.ID, Name: k.Name, Description: k.Description, Role: k.Role}
}

// Middleware validates the credential and enforces the role policy: 401
// missing credential, 403 invalid, 403 insufficient role. On success the
// identity is attached to the context.
func Middleware(store *keys.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			credential, cerr := Credential(r)
			switch {
			case errors.Is(cerr, ErrAmbiguous):
				problem.Detail(w, http.StatusBadRequest, "Multiple methods used to include an access token")
				return
			case errors.Is(cerr, ErrScheme):
				Unauthorized(w, "Authorization scheme must be Bearer")
				return
			case credential == "":
				Unauthorized(w, "API key required - provide either X-API-Key header or Authorization: Bearer header")
				return
			}

			match, err := store.Verify(credential)
			if err != nil {
				alog().Error("api key validation failed", "error", err, "path", r.URL.Path)
				WriteMsg(w, http.StatusInternalServerError, "API key validation failed")
				return
			}
			if match == nil {
				WriteMsg(w, http.StatusForbidden, "Invalid API key")
				return
			}
			authorize(w, r, next, identityOf(match))
		})
	}
}

func authorize(w http.ResponseWriter, r *http.Request, next http.Handler, identity *Identity) {
	needed := RequiredRole(r.Method, r.URL.Path)
	if roleLevels[identity.Role] < roleLevels[needed] {
		alog().Warn("credential role insufficient for request",
			"entity_name", identity.Name,
			"role", identity.Role,
			"required_role", needed,
			"request_path", r.URL.Path,
			"request_method", r.Method,
		)
		WriteMsg(w, http.StatusForbidden,
			"Insufficient role: this operation requires '"+needed+"' (key role: '"+identity.Role+"')")
		return
	}
	next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), contextKey{}, identity)))
}
