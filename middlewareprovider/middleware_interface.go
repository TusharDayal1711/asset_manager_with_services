package middlewareprovider

import (
	"asset/models"
	"net/http"
)

type AuthMiddlewareService interface {
	JWTAuthMiddleware() func(http.Handler) http.Handler
	RequireRole(roles ...models.Role) func(http.Handler) http.Handler
	GetUserAndRolesFromContext(r *http.Request) (string, []string, error)
}
