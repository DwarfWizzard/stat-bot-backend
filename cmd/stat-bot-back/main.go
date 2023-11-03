package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/DwarfWizzard/stat-bot-backend/internal/handler"
	"github.com/DwarfWizzard/stat-bot-backend/internal/repository"
	"github.com/DwarfWizzard/stat-bot-backend/internal/service"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// TODO: move to configs
const (
	POSTGRES_CONN_STRING = "postgres://postgres:1234@localhost:5432/stat-db"
)

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	logger.Info("Server start")

	pool, err := pgxpool.New(context.Background(), POSTGRES_CONN_STRING)
	if err != nil {
		logger.Fatal("Open pgx pool error", zap.Error(err))
	}

	repo := repository.NewRepo(pool)
	svc := service.NewService(logger, repo)
	handler := handler.NewHandler(svc)

	go func() {
		if err := handler.InitRoutes().Start("localhost:8008"); err != nil {
			logger.Warn("Server stopped with error", zap.Error(err))
		} //TODO: move to configs
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	logger.Warn("OS signal received", zap.String("signal", sig.String()))

	pool.Close()

	logger.Info("Server stopped")
}
