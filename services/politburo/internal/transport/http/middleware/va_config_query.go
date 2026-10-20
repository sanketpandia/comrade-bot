package middleware

import (
	"net/http"
)

// VAConfigQueryAlias maps legacy ?type= to ?configType= for /api/v1/va/config.
func VAConfigQueryAlias() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/api/v1/va/config" {
				q := r.URL.Query()
				if q.Get("configType") == "" {
					if legacy := q.Get("type"); legacy != "" {
						q.Set("configType", legacy)
						r.URL.RawQuery = q.Encode()
					}
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
