package middleware

import (
	"fmt"
	"net/http"

	"github.com/basedalex/merch-shop/internal/auth"
)

func IsAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "api/auth" || path == "/api/auth" {
			next.ServeHTTP(w, r)
			return
		}
		tokenString := r.Header.Get("Authorization")
		if tokenString == "" {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprint(w, "Missing authorization header")
			return
		}

		if err := auth.VerifyToken(tokenString); err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprint(w, "Invalid token")
			return
		}
		next.ServeHTTP(w, r)
	})
}

