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

func (a *App) GracefulShutdown(server *http.Server, serverErr <-chan error) error {
	quit := make(chan os.Signal, 1)

	signal.Notify(
		quit,
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer signal.Stop(quit)

	select {
	case err := <-serverErr:
		log.Printf("[FATAL] Server failed to start: %v", err)
		return err
	case sig := <-quit:
		log.Printf("Shutdown signal received: %v", sig)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	defer func() {
		if a.Cancel != nil {
			a.Cancel()
		}
		if a.DB != nil {
			disconnectCtx, dCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer dCancel()
			_ = a.DB.Disconnect(disconnectCtx)
		}
	}()

	if err := server.Shutdown(ctx); err != nil {
		return err
	}

	log.Println("Server gracefully stopped")
	return nil
}
