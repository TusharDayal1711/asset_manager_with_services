package middlewareprovider

import (
	"asset/models"
	"asset/providers"
	"asset/utils"
	"context"
	"errors"
	"github.com/jmoiron/sqlx"
	"net/http"
	"strings"
)

type contextKey string

const (
	userContextKey  contextKey = "user_key"
	rolesContextKey contextKey = "roles_key"
)

type DefaultAuthMiddleware struct {
	db *sqlx.DB
}

func NewAuthMiddlewareService(db *sqlx.DB) providers.AuthMiddlewareService {
	return &DefaultAuthMiddleware{
		db: db,
	}
}

func (a *DefaultAuthMiddleware) JWTAuthMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			accessToken := r.Header.Get("Authorization")

			if accessToken == "" {
				utils.RespondError(w, http.StatusUnauthorized, errors.New("missing access token"), "missing access token")
				return
			}

			userID, roles, err := ParseJWT(accessToken)
			if err != nil && strings.Contains(err.Error(), "invalid or expired token") {
				refreshToken := r.Header.Get("refresh_token")
				if refreshToken == "" {
					utils.RespondError(w, http.StatusUnauthorized, errors.New("missing refresh token"), "access token expired, and refresh token missing")
					return
				}
				userID, err = ParseRefreshToken(refreshToken)
				if err != nil {
					utils.RespondError(w, http.StatusUnauthorized, err, "invalid or expired refresh token")
					return
				}

				var dbRoles []string
				err = a.db.Select(&dbRoles, `SELECT role FROM user_roles WHERE user_id = $1 AND archived_at IS NULL`, userID)
				if err != nil {
					utils.RespondError(w, http.StatusInternalServerError, err, "failed to fetch roles")
					return
				}
				roles = dbRoles

				//generate new token
				newAccessToken, err := GenerateJWT(userID, roles)
				if err != nil {
					utils.RespondError(w, http.StatusInternalServerError, err, "failed to generate access token")
					return
				}
				//generate new refresh token
				newRefreshToken, err := GenerateRefreshToken(userID)
				if err != nil {
					utils.RespondError(w, http.StatusInternalServerError, err, "failed to generate refresh token")
					return
				}
				w.Header().Set("Authorization", newAccessToken)
				w.Header().Set("Refresh_token", newRefreshToken)
			} else if err != nil {
				utils.RespondError(w, http.StatusUnauthorized, err, "unauthorized")
				return
			}

			ctx := context.WithValue(r.Context(), userContextKey, userID)
			ctx = context.WithValue(ctx, rolesContextKey, roles)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func (a *DefaultAuthMiddleware) RequireRole(allowedRoles ...models.Role) func(http.Handler) http.Handler {
	allowed := make(map[models.Role]bool)
	for _, role := range allowedRoles {
		allowed[role] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, roles, err := a.GetUserAndRolesFromContext(r)
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			for _, role := range roles {
				if allowed[models.Role(role)] {
					next.ServeHTTP(w, r)
					return
				}
			}
			http.Error(w, "forbidden", http.StatusForbidden)
		})
	}
}

func (a *DefaultAuthMiddleware) GetUserAndRolesFromContext(r *http.Request) (string, []string, error) {
	userID, ok := r.Context().Value(userContextKey).(string)
	if !ok {
		return "", nil, errors.New("user ID not found in context")
	}
	roles, ok := r.Context().Value(rolesContextKey).([]string)
	if !ok {
		return "", nil, errors.New("roles not found in context")
	}
	return userID, roles, nil
}
