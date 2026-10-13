package middleware

import (
	"net/http"

	"paluketok/server/pkg/response"
)

// Admin menuntut Actor.IsAdmin (dari profiles.is_admin lewat JWT middleware);
// selain itu 403 (openapi.md bagian admin, security.md bagian 3).
func Admin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a, ok := ActorFrom(r.Context())
		if !ok || !a.IsAdmin {
			response.JSON(w, http.StatusForbidden, response.ErrorBody{
				Code: response.CodeForbidden, Message: "khusus admin",
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}
