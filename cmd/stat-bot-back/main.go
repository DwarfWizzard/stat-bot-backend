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
	"golang.org/x/crypto/ssh"
)

const (
	POSTGRES_CONN_STRING        = "POSTGRES_CONN_STRING"
	SSH_HOST     = "SSH_HOST"
	SSH_USER     = "SSH_USER"
	SSH_USER_PWD = "SSH_USER_PWD"
)

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	logger.Info("Server start")

	pool, err := pgxpool.New(context.Background(), os.Getenv(POSTGRES_CONN_STRING))
	if err != nil {
		logger.Fatal("Open pgx pool error", zap.Error(err))
	}

	config := &ssh.ClientConfig{
		User: os.Getenv(SSH_USER),
		Auth: []ssh.AuthMethod{
			ssh.Password(os.Getenv(SSH_USER_PWD)),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	conn, err := ssh.Dial("tcp", os.Getenv(SSH_HOST), config)
	if err != nil {
		logger.Fatal("Open ssh error", zap.Error(err))
	}
	defer conn.Close()

	repo := repository.NewRepo(pool)
	svc := service.NewService(logger, repo, conn)
	handler := handler.NewHandler(svc)

	go func() {
		if err := handler.InitRoutes().Start(":8008"); err != nil {
			logger.Warn("Server stopped with error", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	logger.Warn("OS signal received", zap.String("signal", sig.String()))

	pool.Close()

	logger.Info("Server stopped")
}
