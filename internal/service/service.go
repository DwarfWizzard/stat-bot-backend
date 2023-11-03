package service

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"syscall"
	"time"

	"github.com/DwarfWizzard/stat-bot-backend/internal/repository"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

// TODO: add work with many databases
const (
	DB_NAME = "stat-db"

	IDLE_CONNS_PERCANTAGE = 80
	DISK_USAGE_PERCANTAGE = 80
	LOCK_IDLE_PERCANTAGE  = 80
	ROLLBACK_PRECANTAGE   = 80
	CONN_TTL              = 15 * time.Second
)

type Service struct {
	logger *zap.Logger

	monitoredDB *repository.Repo
}

func NewService(logger *zap.Logger, monitoredDB *repository.Repo) *Service {
	return &Service{
		logger:      logger,
		monitoredDB: monitoredDB,
	}
}

// TODO: add more allerts
func (s *Service) CollectMetrics(c echo.Context) error {
	metrics := Metrics{}

	ctx := c.Request().Context()

	err := s.monitoredDB.Ping(ctx)
	if err != nil {
		s.logger.Error("Ping database error", zap.Error(err))
		metrics.ServerActive = false
		metrics.Allerts = append(metrics.Allerts, &Allert{Type: AllertUnattainableDB, Description: AllertUnattainableDB.Description()})
		return c.JSON(http.StatusOK, &Response{Data: metrics})
	}
	metrics.ServerActive = true

	//metrics collection
	_, rollbacks, err := s.monitoredDB.TransactionsNumber(ctx)
	if err != nil {
		s.logger.Error("Get number of transactions error", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, &Response{Error: err})
	}

	totalExecTime, err := s.monitoredDB.TotalExecutionTime(ctx)
	if err != nil {
		s.logger.Error("Get total execution time error", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, &Response{Error: err})
	}

	totalCalls, err := s.monitoredDB.TotalCalls(ctx)
	if err != nil {
		s.logger.Error("Get total calls error", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, &Response{Error: err})
	}

	totalConns, err := s.monitoredDB.TotalConns(ctx)
	if err != nil {
		s.logger.Error("Get total conns error", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, &Response{Error: err})
	}

	conns, err := s.monitoredDB.ListConnsByDatabase(ctx, DB_NAME)
	if err != nil {
		s.logger.Error("Get conns by db error", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, &Response{Error: err})
	}

	totalIdleConns, err := s.monitoredDB.TotalIdleConns(ctx)
	if err != nil {
		s.logger.Error("Get total idle calls error", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, &Response{Error: err})
	}

	// totalLockIdleCalls, err := s.monitoredDB.TotalIdleConns(ctx)
	// if err != nil {
	// 	s.logger.Error("Get total idle awaiting for unlock calls error", zap.Error(err))
	// 	return c.JSON(http.StatusInternalServerError, &Response{Error: err})
	// }

	diskUsage, err := s.monitoredDB.TotalDiskUsageByDB(ctx, DB_NAME)
	if err != nil {
		s.logger.Error("Get longest call error", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, &Response{Error: err})
	}

	var Allerts []*Allert
	//capsulate values
	if totalCalls != 0 {
		metrics.Rollbacks = (float64(rollbacks) * 100) / float64(totalCalls)
		if metrics.Rollbacks > ROLLBACK_PRECANTAGE {
			Allerts = append(Allerts, NewAllert(AllertManyRollbacks, nil))
		}

		metrics.MeanResponseTime = totalExecTime / float64(totalCalls)
	}

	metrics.Operations = totalCalls

	metrics.ConnsNum = totalConns

	if totalConns != 0 {
		metrics.IdleConns = (float64(totalIdleConns) * 100) / float64(totalConns)
		if metrics.IdleConns > IDLE_CONNS_PERCANTAGE {
			Allerts = append(Allerts, NewAllert(AllertManyIdleConn, nil))
		}
	}

	metrics.DiskUsage = diskUsage / 1 >> 20

	allSpace := s.getTotalSpace()
	if allSpace != 0 {
		metrics.DiskUsagePercantage = (float64(diskUsage) * 100) / float64(allSpace)
		if metrics.DiskUsagePercantage > DISK_USAGE_PERCANTAGE {
			Allerts = append(Allerts, NewAllert(AllertLowMemory, nil))
		}
	}

	for _, conn := range conns {
		metrics.Conns = append(metrics.Conns, Conn{LastQuery: conn.LastQuery, QueryStart: conn.QuertStart, PID: conn.PID})
		if time.Since(conn.QuertStart) > CONN_TTL {
			Allerts = append(Allerts, NewAllert(AllertTooLongIdleConn, conn.PID))
		}
	}

	metrics.Allerts = append(metrics.Allerts, Allerts...)

	return c.JSON(http.StatusOK, metrics)
}

func (s *Service) TerminateConn(c echo.Context) error {
	pidSrt := c.Param("pid")

	ctx := c.Request().Context()

	pid, err := strconv.Atoi(pidSrt)
	if err != nil {
		return c.JSON(http.StatusBadRequest, &Response{Error: errors.New("invalid input")}) //TODO: add errors handling
	}

	success, err := s.monitoredDB.TerminateConnByPid(ctx, pid)
	if err != nil {
		s.logger.Error("Terminate connection by PID error", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, &Response{Error: err})
	}

	if !success {
		return c.JSON(http.StatusOK, &Response{Data: NewAllert(AllertUnknown, fmt.Sprintf("Соединение %d не закрыто", pid))})
	}

	return c.JSON(http.StatusOK, &Response{Data: success})
}

func (s *Service) getTotalSpace() uint64 {
	fs := syscall.Statfs_t{}
	err := syscall.Statfs("/bitnami/postgresql", &fs)
	if err != nil {
		s.logger.Warn("Get space form container error", zap.Error(err))
	}
	return fs.Blocks * uint64(fs.Bsize)
}
