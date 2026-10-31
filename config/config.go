package config

import (
	"os"
	"strconv"
	"sync"

	"github.com/joho/godotenv"
)

type Config struct {
	Port                string
	MongoURI            string
	DBName              string
	JWTSecret           string
	JWTExpiryHours      int
	Environment         string
	AllowedOrigins      []string
	AWSRegion           string
	AWSAccessKeyID      string
	AWSSecretAccessKey  string
	AWSS3Bucket         string
	AWSS3Endpoint       string
	AWSS3ForcePathStyle bool
	AWSS3PublicURL      string
}

var (
	appConfig *Config
	once      sync.Once
)

func Get() *Config {
	once.Do(func() {
		appConfig = LoadConfig()
	})
	return appConfig
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
		Port:                port,
		MongoURI:            mongoURI,
		DBName:              dbName,
		JWTSecret:           jwtSecret,
		JWTExpiryHours:      expiryHours,
		Environment:         env,
		AllowedOrigins:      []string{"http://localhost:3000", "http://127.0.0.1:3000"},
		AWSRegion:           getEnv("AWS_REGION", "ap-south-1"),
		AWSAccessKeyID:      getEnv("AWS_ACCESS_KEY_ID", ""),
		AWSSecretAccessKey:  getEnv("AWS_SECRET_ACCESS_KEY", ""),
		AWSS3Bucket:         getEnv("AWS_S3_BUCKET", ""),
		AWSS3Endpoint:       getEnv("AWS_S3_ENDPOINT", ""),
		AWSS3ForcePathStyle: getEnv("AWS_S3_FORCE_PATH_STYLE", "false") == "true",
		AWSS3PublicURL:      getEnv("AWS_S3_PUBLIC_URL", ""),
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

