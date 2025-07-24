package server

import (
	"asset/handler/assetHandler"
	"asset/handler/userHandler"
	"asset/providers"
	"asset/providers/configProvider"
	"asset/providers/databaseProvider"
	"asset/providers/middlewareprovider"
	assetrepo "asset/repository/asset"
	userrepo "asset/repository/user"
	assetservice "asset/services/asset"
	userservice "asset/services/user"
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
	UserHandler  *userhandler.UserHandler
	AssetHandler *assethandler.AssetHandler
	httpServer   *http.Server
}

func SrvInit() *Server {
	cfg := configprovider.NewConfigProvider()
	cfg.LoadEnv()

	db := databaseProvider.NewDBProvider(cfg.GetDatabaseString())
	middleware := middlewareprovider.NewAuthMiddlewareService(db.DB())

	// repositories
	userRepo := userrepo.NewUserRepository(db.DB())
	assetRepo := assetrepo.NewAssetRepository(db.DB())

	// services
	userService := userservice.NewUserService(userRepo, db.DB())
	assetService := assetservice.NewAssetService(assetRepo, db.DB())

	// handlers
	userHandler := userhandler.NewUserHandler(userService, middleware)
	assetHandler := assethandler.NewAssetHandler(assetService, middleware)

	return &Server{
		Config:       cfg,
		DB:           db,
		Middleware:   middleware,
		UserHandler:  userHandler,
		AssetHandler: assetHandler,
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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		log.Printf("error shutting down server: %v", err)
	}

	if err := s.DB.Close(); err != nil {
		log.Printf("error closing DB: %v", err)
	}

	fmt.Println("Server shutdown complete.")
}
