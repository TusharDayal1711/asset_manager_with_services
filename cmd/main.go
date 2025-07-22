package main

import (
	"asset/config"
	"asset/database"
	"asset/handler/assetHandler"
	"asset/handler/userHandler"
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
	// 1. Load environment variables
	config.LoadEnv()

	// 2. Initialize database
	dbConnectionString := config.GetDatabaseString()
	database.Init(dbConnectionString)
	defer database.DB.Close()

	// 3. Setup repositories
	assetRepo := asset.NewAssetRepository(database.DB)
	userRepo := user.NewUserRepository(database.DB)

	// 4. Setup services
	assetService := assetservice.NewAssetService(assetRepo, database.DB)
	userService := userservice.NewUserService(userRepo, database.DB)

	// 5. Setup handlers
	assetHandler := assethandler.NewAssetHandler(assetService)
	userHandler := userhandler.NewUserHandler(userService)

	routeHandler := routes.RouteHandler{
		UserHandler:  userHandler,
		AssetHandler: assetHandler,
	}
	router := routes.RegisterRoutes(routeHandler)

	// 7. Setup HTTP server
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
