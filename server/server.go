package server

import (
	"asset/providers"
	"asset/providers/configProvider"
	"asset/providers/databaseProvider"
	firebaseprovider "asset/providers/firebaseProvider"
	"asset/providers/loggerProvider"
	"asset/providers/middlewareprovider"
	redisprovider "asset/providers/redisProvider"
	"asset/services/asset"
	"asset/services/user"
	"context"
	"fmt"
	"go.uber.org/zap"
	"log"
	"net/http"
	"os"
	"time"
)

type Server struct {
	Config       providers.ConfigProvider
	DB           providers.DBProvider
	Middleware   providers.AuthMiddlewareService
	UserHandler  *userservice.UserHandler
	AssetHandler *assetservice.AssetHandler
	httpServer   *http.Server
	Logger       providers.ZapLoggerProvider
	Firebase     providers.FirebaseProvider
	Redis        providers.RedisProvider
}

func ServerInit() *Server {
	cfg := configprovider.NewConfigProvider()
	cfg.LoadEnv()

	//zap logger
	logs := loggerProvider.NewLogProvider()
	logs.InitLogger()
	logs.GetLogger().Info("inside serverInit")

	//firebase
	serviceAccountJSON, err := os.ReadFile(os.Getenv("FIREBASE_CONFIG"))
	if err != nil {
		logs.GetLogger().Error("failed to read service account json file ::", zap.Error(err))
	}
	firebase, err := firebaseprovider.NewFirebaseProvider(serviceAccountJSON)
	if err != nil {
		logs.GetLogger().Error("failed to initialize firebase provider ::", zap.Error(err))
	}

	//redis provider
	redisPort := ":" + os.Getenv("REDIS_PORT")
	redis := redisprovider.NewRedisProvider(redisPort)
	logs.GetLogger().Info("redis initialized")
	redis.Ping(context.Background())

	//database provider
	db := databaseProvider.NewDBProvider(cfg.GetDatabaseString())
	middleware := middlewareprovider.NewAuthMiddlewareService(db.DB())

	//repositories
	userRepo := userservice.NewUserRepository(db.DB(), logs, firebase, redis)
	assetRepo := assetservice.NewAssetRepository(db.DB())

	//services
	userService := userservice.NewUserService(userRepo, db.DB(), logs, firebase)
	assetService := assetservice.NewAssetService(assetRepo, db.DB())

	//handlers
	userHandler := userservice.NewUserHandler(userService, middleware, logs, firebase)
	assetHandler := assetservice.NewAssetHandler(assetService, middleware)

	logs.GetLogger().Info("\nall provider and services initialized...")
	return &Server{
		Config:       cfg,
		DB:           db,
		Middleware:   middleware,
		UserHandler:  userHandler,
		AssetHandler: assetHandler,
		Logger:       logs,
		Redis:        redis,
	}
}

func (s *Server) Start() {
	addr := ":" + s.Config.GetServerPort()

	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      s.InjectRoutes(),
		ReadTimeout:  2 * time.Minute,
		WriteTimeout: 2 * time.Minute,
		IdleTimeout:  2 * time.Minute,
	}

	fmt.Println("server running on", addr)
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		s.Logger.GetLogger().Fatal("failed to start server", zap.Error(err))
	}
}

func (s *Server) Stop() {
	s.Logger.GetLogger().Info("shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	s.Logger.SyncLogger()
	if err := s.httpServer.Shutdown(ctx); err != nil {
		log.Printf("error shutting down server: %v", err)
	}
	if err := s.DB.Close(); err != nil {
		log.Printf("error closing DB: %v", err)
	}
	if err := s.Redis.Close(); err != nil {
		s.Logger.GetLogger().Error("error closing redis connection")
	}
	s.Logger.GetLogger().Info("Server shutdown gracefully")
}
