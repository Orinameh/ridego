package middleware

import (
	"net/http"

	"github.com/ridego/pkg/response"
)

// GetCallerInfo extracts user ID and role from request headers
func GetCallerInfo(r *http.Request) (userID, role string) {
	return r.Header.Get("X-User-ID"), r.Header.Get("X-User-Role")
}

// AuthorizeUser checks if the caller can access the target user
func AuthorizeUser(r *http.Request, targetUserID string) bool {
	callerID, callerRole := GetCallerInfo(r)
	return callerID == targetUserID || callerRole == "admin"
}

// RequireUserAccess is a middleware that ensures the caller can access the target user
func RequireUserAccess(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		targetUserID := r.PathValue("id")
		if targetUserID == "" {
			response.Error(w, http.StatusBadRequest, "user ID required")
			return
		}

		callerID, callerRole := GetCallerInfo(r)
		if callerID != targetUserID && callerRole != "admin" {
			response.Error(w, http.StatusForbidden, "forbidden")
			return
		}

		next(w, r)
	}
}

// RequireAdmin is a middleware that ensures the caller is an admin
func RequireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, callerRole := GetCallerInfo(r)
		if callerRole != "admin" {
			response.Error(w, http.StatusForbidden, "admin access required")
			return
		}
		next(w, r)
	}
}

// HasRole checks if the caller has any of the specified roles
func HasRole(r *http.Request, roles ...string) bool {
	_, callerRole := GetCallerInfo(r)
	for _, role := range roles {
		if callerRole == role {
			return true
		}
	}
	return false
}
