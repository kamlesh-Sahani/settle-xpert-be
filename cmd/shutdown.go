package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func (a *App) GracefulShutdown(server *http.Server) error {
	quit := make(chan os.Signal, 1)

	signal.Notify(
		quit,
		os.Interrupt,
		syscall.SIGTERM,
	)

	<-quit
	log.Println("Shutdown signal received")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		return err
	}

	if a.Cancel != nil {
		a.Cancel()
	}

	if a.DB != nil {
		_ = a.DB.Disconnect(ctx)
	}

	log.Println("Server gracefully stopped")
	return nil
}
