package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"lead-crm/backend/database"
	"lead-crm/backend/internal/routes"
	"lead-crm/backend/utils"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	commonsmw "github.com/kamlesh-Sahani/go-commons/middleware"
	"go.mongodb.org/mongo-driver/mongo"
)

type App struct {
	Router *gin.Engine
	DB     *mongo.Client
	Ctx    context.Context
	Cancel context.CancelFunc
}

func New() (*App, error) {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on system environment variables")
	}

	ctx, cancel := context.WithCancel(context.Background())

	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017/debt_crm"
	}

	mongoClient, err := database.ConnectMongoDB(mongoURI)
	if err != nil {
		cancel()
		return nil, err
	}

	utils.EnsureCollectionIndexes()

	if os.Getenv("GIN_MODE") == "release" {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.New()

	router.Use(gin.Logger())
	router.Use(commonsmw.Recovery())
	router.Use(commonsmw.CORS())
	router.Use(commonsmw.MaxBodySize(10 * 1024 * 1024)) // 10MB limit

	routes.SetupRoutes(router)

	app := &App{
		Router: router,
		DB:     mongoClient,
		Ctx:    ctx,
		Cancel: cancel,
	}

	return app, nil
}

func (a *App) Run() error {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: a.Router,
	}

	serverErr := make(chan error, 1)
	go func() {
		fmt.Printf("🚀 DebtCare CRM Backend running on port %s\n", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	return a.GracefulShutdown(srv, serverErr)
}
