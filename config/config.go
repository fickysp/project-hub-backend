package config

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv         string
	AppPort        string
	DatabaseURL    string
	JWTSecret      string
	RefreshSecret  string
	SupabaseURL    string
	SupabaseKey    string
	SupabaseBucket string
	FrontendURL    string
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		slog.Warn("File .env not found, using system environment variables")
	}

	return &Config{
		AppEnv:         getEnv("APP_ENV", "development"),
		AppPort:        getEnv("APP_PORT", "8080"),
		DatabaseURL:    getEnvOrPanic("DATABASE_URL"),
		JWTSecret:      getEnvOrPanic("JWT_SECRET"),
		RefreshSecret:  getEnvOrPanic("REFRESH_SECRET"),
		SupabaseURL:    getEnvOrPanic("SUPABASE_URL"),
		SupabaseKey:    getEnvOrPanic("SUPABASE_KEY"),
		SupabaseBucket: getEnv("SUPABASE_BUCKET", "uploads"),
		FrontendURL:    getEnv("FRONTEND_URL", "http://localhost:3000"),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvOrPanic(key string) string {
	value := os.Getenv(key)
	if value == "" {
		panic(fmt.Sprintf("Environment variable %s empty!", key))
	}
	return value
}
