package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/BurrowedInCode/cannabis_coa_analyzer/db"
	"github.com/BurrowedInCode/cannabis_coa_analyzer/internal/auth"
	"github.com/BurrowedInCode/cannabis_coa_analyzer/internal/coa"
	"github.com/BurrowedInCode/cannabis_coa_analyzer/middleware"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	db, err := db.ConnectToDB(os.Getenv("DATABASE_URL"))
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}

	stats := db.Stat()

	logger.Info("database connected",
		"total_conns", stats.TotalConns(),
		"max_conns", stats.MaxConns(),
		"idle_conns", stats.IdleConns(),
	)

	coaStore := coa.NewStore(db)
	coaSvc, err := coa.NewService("prompts/extract_coa_v3.md")
	if err != nil {
		logger.Error("failed to load COA prompt", "error", err)
		os.Exit(1)
	}

	loginStore := auth.NewStore(db)

	mux := http.NewServeMux()
	mux.Handle("POST /coa/analyze", coa.AnalyzeCOAHandler(logger, coaSvc, coaStore))
	mux.Handle("GET /coa/analyses", coa.GetAllCOAAnalysesHandler(logger, coaStore))
	mux.Handle("GET /coa/analyses/{id}", coa.GetCOAAnalysisHandler(logger, coaStore))
	mux.Handle("POST /user/register", auth.RegisterUserHandler(logger, loginStore))
	mux.Handle("POST /user/login", auth.LoginUserHandler(logger, loginStore, os.Getenv("JWT_SECRET")))
	server := &http.Server{
		Handler:      middleware.Cors(mux),
		Addr:         ":8080",
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	serverErr := make(chan error, 1)
	go func() {
		logger.Info("Server starting", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	select {
	case err := <-serverErr:
		logger.Error("Server error", "error", err)
		os.Exit(1)
	case <-ctx.Done():
		logger.Info("Shutdown signal received")
	}

	stop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP shutdown error", "error", err)
		os.Exit(1)
	}

	logger.Info("Shutdown complete")
}
