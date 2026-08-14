package server

import (
	"crypto/sha256"
	"crypto/subtle"
	"net"
	"net/http"
	"net/url"
	"strings"
)

const webUser = "pi"

// Security enforces host allow-listing and optional Basic Auth. When no web
// password is configured, only the host check applies.
func Security(password string, allowedHosts []string) func(http.Handler) http.Handler {
	allow := make(map[string]bool, len(allowedHosts))
	for _, h := range allowedHosts {
		allow[strings.ToLower(strings.TrimSpace(h))] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if origin := r.Header.Get("Origin"); origin != "" && loopbackOrigin(origin) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
				if r.Method == http.MethodOptions {
					w.WriteHeader(http.StatusNoContent)
					return
				}
			}
			if !hostAllowed(r.Host, allow) {
				http.Error(w, "Untrusted request", http.StatusForbidden)
				return
			}
			if password != "" && !validBasicAuth(r.Header.Get("Authorization"), password) {
				w.Header().Set("WWW-Authenticate", `Basic realm="Pi Web Go", charset="UTF-8"`)
				http.Error(w, "Authentication required", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func loopbackOrigin(origin string) bool {
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	hostname := u.Hostname()
	return hostname == "127.0.0.1" || hostname == "localhost" || hostname == "::1"
}

func hostAllowed(host string, allow map[string]bool) bool {
	hostname := host
	if h, _, err := net.SplitHostPort(host); err == nil {
		hostname = h
	}
	hostname = strings.ToLower(hostname)
	if len(allow) == 0 {
		return hostname == "127.0.0.1" || hostname == "localhost" || hostname == "::1"
	}
	// Check both bare hostname and host:port (e.g. "localhost" and "localhost:5173")
	if allow[hostname] {
		return true
	}
	if _, port, err := net.SplitHostPort(host); err == nil {
		return allow[hostname+":"+port]
	}
	return false
}

func validBasicAuth(header, password string) bool {
	const prefix = "Basic "
	if !strings.HasPrefix(header, prefix) {
		return false
	}
	decoded, err := base64Decode(strings.TrimPrefix(header, prefix))
	if err != nil {
		return false
	}
	user, pass, ok := strings.Cut(decoded, ":")
	if !ok {
		return false
	}
	userHash := sha256.Sum256([]byte(user))
	passHash := sha256.Sum256([]byte(pass))
	wantPass := sha256.Sum256([]byte(password))
	wantUser := sha256.Sum256([]byte(webUser))
	return subtle.ConstantTimeCompare(userHash[:], wantUser[:]) == 1 &&
		subtle.ConstantTimeCompare(passHash[:], wantPass[:]) == 1
}

func base64Decode(s string) (string, error) {
	b, err := decodeBase64(s)
	return string(b), err
}
