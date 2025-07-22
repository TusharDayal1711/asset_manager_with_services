package main

import (
	"asset/config"
	"asset/database"
	"asset/handler/assetHandler"
	"asset/handler/userHandler"
	"asset/middlewareprovider"
	"asset/repository/asset"
	"asset/repository/user"
	"asset/routes"
	assetservice "asset/serviceprovider/asset"
	userservice "asset/serviceprovider/user"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const shutdownTimeout = 10 * time.Second

func main() {
	config.LoadEnv()
	dbConnectionString := config.GetDatabaseString()

	database.Init(dbConnectionString)
	defer database.DB.Close()

	assetRepo := asset.NewAssetRepository(database.DB)
	userRepo := user.NewUserRepository(database.DB)

	assetService := assetservice.NewAssetService(assetRepo, database.DB)
	userService := userservice.NewUserService(userRepo, database.DB)

	authMiddleware := middlewareprovider.NewAuthMiddlewareService()

	assetHandler := assethandler.NewAssetHandler(assetService, authMiddleware)
	userHandler := userhandler.NewUserHandler(userService, authMiddleware)

	routeHandler := routes.RouteHandler{
		UserHandler:    userHandler,
		AssetHandler:   assetHandler,
		AuthMiddleware: authMiddleware,
	}
	router := routes.RegisterRoutes(routeHandler)

	server := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		fmt.Println("Server running on port 8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Could not listen on :8080: %v\n", err)
		}
	}()

	<-done
	fmt.Println("\nShutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}
	fmt.Println("Server exited gracefully.")
}
