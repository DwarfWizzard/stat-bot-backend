package service

import (
	"bytes"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/DwarfWizzard/stat-bot-backend/internal/repository"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
)

// TODO: add work with many databases
const (
	DB_NAME = "stat-db"

	IDLE_CONNS_PERCANTAGE = 70
	DISK_USAGE_PERCANTAGE = 80
	LOCK_IDLE_PERCANTAGE  = 80
	ROLLBACK_PRECANTAGE   = 80
	CONN_TTL              = 15 * time.Second
	QUERY_TTL             = 10 * time.Second
)

type Service struct {
	logger *zap.Logger

	monitoredDB *repository.Repo
	sshClient   *ssh.Client
}

func NewService(logger *zap.Logger, monitoredDB *repository.Repo, sshCleint *ssh.Client) *Service {
	return &Service{
		logger:      logger,
		monitoredDB: monitoredDB,
		sshClient:   sshCleint,
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

	conns, err := s.monitoredDB.ListConns(ctx)
	if err != nil {
		s.logger.Error("Get conns by db error", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, &Response{Error: err})
	}

	totalIdleConns, err := s.monitoredDB.TotalIdleConns(ctx)
	if err != nil {
		s.logger.Error("Get total idle calls error", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, &Response{Error: err})
	}

	diskUsage, err := s.monitoredDB.TotalDiskUsageByDB(ctx, DB_NAME)
	if err != nil {
		s.logger.Error("Get longest call error", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, &Response{Error: err})
	}

	longestConn, err := s.monitoredDB.LongestQuery(ctx)
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

	metrics.ConnsNum = len(conns)

	if metrics.ConnsNum != 0 {
		metrics.IdleConns = (float64(totalIdleConns) * 100) / float64(metrics.ConnsNum)
		if metrics.IdleConns > IDLE_CONNS_PERCANTAGE {
			Allerts = append(Allerts, NewAllert(AllertManyIdleConn, nil))
		}
	}

	metrics.DiskUsage = diskUsage / 1 >> 20

	allSpace, err := s.getPostgresContainerTotalSpace()
	if err != nil {
		s.logger.Error("Get total space error", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, &Response{Error: err})
	}
	if allSpace != 0 {
		metrics.DiskUsagePercantage = (float64(diskUsage) * 100) / float64(allSpace)
		if metrics.DiskUsagePercantage > DISK_USAGE_PERCANTAGE {
			Allerts = append(Allerts, NewAllert(AllertLowMemory, nil))
		}
	}

	for _, conn := range conns {
		metrics.Conns = append(metrics.Conns, Conn{
			LastQuery:     conn.LastQuery,
			WaitEvent:     conn.WaitEvent,
			WaitEventType: conn.WaitEventType,
			TxnStart:      conn.TxnStart,
			QueryStart:    conn.QueryStart,
			State:         conn.State,
			PID:           conn.PID,
		})
		if time.Since(conn.QueryStart) > CONN_TTL {
			Allerts = append(Allerts, NewAllert(AllertTooLongIdleConn, conn.PID))
		}
	}

	if longestConn != nil {
		metrics.LongestActiveConn = Conn{
			LastQuery:     longestConn.LastQuery,
			WaitEvent:     longestConn.WaitEvent,
			WaitEventType: longestConn.WaitEventType,
			TxnStart:      longestConn.TxnStart,
			QueryStart:    longestConn.QueryStart,
			State:         longestConn.State,
			PID:           longestConn.PID,
		}
		if longestConn.TxnStart != nil && time.Since(*longestConn.TxnStart) > CONN_TTL {
			Allerts = append(Allerts, NewAllert(AllertTooLongQuery, []any{longestConn.PID, longestConn.LastQuery}))
		}
	}

	metrics.Allerts = append(metrics.Allerts, Allerts...)

	return c.JSON(http.StatusOK, metrics)
}

func (s *Service) getPostgresContainerTotalSpace() (int64, error) {
	session, err := s.sshClient.NewSession()
	if err != nil {
		return 0, err
	}
	defer session.Close()

	var buff bytes.Buffer
	session.Stdout = &buff
	if err := session.Run("df | awk 'NR==2{print $2}'"); err != nil {
		return 0, err
	}

	allSpaceStr := buff.String()
	allSpaceStr = strings.TrimSpace(allSpaceStr)
	allSpace, err := strconv.ParseInt(allSpaceStr, 10, 64)
	if err != nil {
		return 0, err
	}

	return allSpace, nil
}
