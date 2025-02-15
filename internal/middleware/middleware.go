package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/basedalex/merch-shop/internal/auth"
)

func Authentication(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "api/auth" || path == "/api/auth" {
			next.ServeHTTP(w, r)
			return
		}
		tokenString := r.Header.Get("Authorization")
		authInfo := strings.Split(tokenString, " ")
		token := authInfo[1]
		if tokenString == "" {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprint(w, "Missing authorization header")
			return
		}

		if err := auth.VerifyToken(token); err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprint(w, "Invalid token", token)
			return
		}
		next.ServeHTTP(w, r)
	})
}

