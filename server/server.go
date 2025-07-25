package server

import (
	"asset/providers"
	"asset/providers/configProvider"
	"asset/providers/databaseProvider"
	"asset/providers/loggerProvider"
	"asset/providers/middlewareprovider"
	"asset/services/asset"
	"asset/services/user"
	"context"
	"fmt"
	"log"
	"net/http"
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
}

func ServerInit() *Server {
	cfg := configprovider.NewConfigProvider()
	cfg.LoadEnv()

	//zap logger
	logs := loggerProvider.NewLogProvider()
	logs.InitLogger()
	logs.GetLogger().Info("inside serverInit")

	db := databaseProvider.NewDBProvider(cfg.GetDatabaseString())
	middleware := middlewareprovider.NewAuthMiddlewareService(db.DB())

	//repositories
	userRepo := userservice.NewUserRepository(db.DB(), logs)
	assetRepo := assetservice.NewAssetRepository(db.DB())

	//services
	userService := userservice.NewUserService(userRepo, db.DB(), logs)
	assetService := assetservice.NewAssetService(assetRepo, db.DB())

	//handlers
	userHandler := userservice.NewUserHandler(userService, middleware, logs)
	assetHandler := assetservice.NewAssetHandler(assetService, middleware)

	return &Server{
		Config:       cfg,
		DB:           db,
		Middleware:   middleware,
		UserHandler:  userHandler,
		AssetHandler: assetHandler,
		Logger:       logs,
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
		log.Fatalf("server error: %v", err)
	}
}

func (s *Server) Stop() {
	fmt.Println("shutting down server...")
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

	fmt.Println("Server shutdown complete.")
}
