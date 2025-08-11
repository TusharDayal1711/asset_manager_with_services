package configprovider

import (
	"asset/providers"
	"fmt"
	"github.com/joho/godotenv"
	"log"
	"os"
)

func NewConfigProvider() providers.ConfigProvider {
	return &EnvConfigProvider{}
}

func (e *EnvConfigProvider) LoadEnv() error {
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not loaded, using system envs")
	}

	e.dbUser = os.Getenv("DB_USER")
	e.dbPassword = os.Getenv("DB_PASSWORD")
	e.dbHost = os.Getenv("DB_HOST")
	e.dbPort = os.Getenv("DB_PORT")
	e.dbName = os.Getenv("DB_NAME")
	e.serverPort = os.Getenv("SERVER_PORT")
	return nil
}

func (e *EnvConfigProvider) GetServerPort() string {
	return e.serverPort
}

func (e *EnvConfigProvider) GetDatabaseString() string {
	return fmt.Sprintf("user=%s password=%s host=%s port=%s dbname=%s sslmode=disable",
		e.dbUser, e.dbPassword, e.dbHost, e.dbPort, e.dbName)
}
