package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Port           string
	MongoURI       string
	DBName         string
	JWTSecret      string
	JWTExpiryHours int
	Environment    string
	AllowedOrigins []string
}

func LoadConfig() *Config {
	_ = godotenv.Load()

	port := getEnv("PORT", "8080")
	mongoURI := getEnv("MONGO_URI", "mongodb://localhost:27017")
	dbName := getEnv("DB_NAME", "debt_crm")
	jwtSecret := getEnv("JWT_SECRET", "super-secret-debt-crm-jwt-key-2026-production")
	env := getEnv("ENV", "development")

	expiryHoursStr := getEnv("JWT_EXPIRY_HOURS", "24")
	expiryHours, err := strconv.Atoi(expiryHoursStr)
	if err != nil {
		expiryHours = 24
	}

	return &Config{
		Port:           port,
		MongoURI:       mongoURI,
		DBName:         dbName,
		JWTSecret:      jwtSecret,
		JWTExpiryHours: expiryHours,
		Environment:    env,
		AllowedOrigins: []string{"http://localhost:3000", "http://127.0.0.1:3000"},
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
