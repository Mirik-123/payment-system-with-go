package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request){
		logger.Info("health check", "remote", r.RemoteAddr)
		w.Header().Set("Content-Type", "application/json")
		if _,err := w.Write([]byte(`{"status":"ok"}` + "\n")); err != nil{
			logger.Error("Failed to write health response", "err", err)
		}
	})

	srv := &http.Server{
		Addr:           	":8081",
		Handler:			mux,
		ReadHeaderTimeout:	5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Info("wallet service listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("Server failed", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	logger.Info("Shutting down gracefully")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10 * time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("Graceful shutdown failed", "err", err)

		if closeErr := srv.Close(); closeErr != nil {
			logger.Error("Force close faild", "err", closeErr)
		}
	}
	
	logger.Info("Server stopped")
}
