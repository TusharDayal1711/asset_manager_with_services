package providers

import (
	"asset/models"
	"context"
	firebaseauth "firebase.google.com/go/v4/auth"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
	"net/http"
	"time"
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

type FirebaseProvider interface {
	VerifyIDToken(ctx context.Context, idToken string) (*firebaseauth.Token, error)
	GetUserByUID(ctx context.Context, uid string) (*firebaseauth.UserRecord, error)
	GetUserByEmail(ctx context.Context, email string) (*firebaseauth.UserRecord, error)
	CreateUser(ctx context.Context, email string) (*firebaseauth.UserRecord, error)
	DeleteAuthUser(ctx context.Context, uid string) error
	GetAuthUserID(ctx context.Context, email string) (string, error)
}

type RedisProvider interface {
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Get(ctx context.Context, key string) (string, error)
	Ping(ctx context.Context) error
	Close() error
}
