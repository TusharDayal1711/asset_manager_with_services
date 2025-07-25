package providers

import (
	"asset/models"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
	"net/http"
)

type AuthMiddlewareService interface {
	JWTAuthMiddleware() func(http.Handler) http.Handler
	RequireRole(roles ...models.Role) func(http.Handler) http.Handler
	GetUserAndRolesFromContext(r *http.Request) (string, []string, error)
}

type ConfigProvider interface {
	LoadEnv() error
	GetDatabaseString() string
	GetServerPort() string
}

type DBProvider interface {
	DB() *sqlx.DB
	Close() error
}

type ZapLoggerProvider interface {
	InitLogger()
	SyncLogger()
	GetLogger() *zap.Logger
}
