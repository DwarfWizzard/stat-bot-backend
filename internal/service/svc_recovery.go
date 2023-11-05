package service

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

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

func (s *Service) ShutdownDatabase(c echo.Context) error {
	ctx := c.Request().Context()

	err := s.monitoredDB.CreateCheckpoint(ctx)
	if err != nil {
		s.logger.Error("Create checkpoint error", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, &Response{Error: err})
	}

	session, err := s.sshClient.NewSession()
	if err != nil {
		s.logger.Error("Create new ssh session error", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, &Response{Error: err})
	}
	defer session.Close()

	var stdErrBuff bytes.Buffer
	session.Stderr = &stdErrBuff
	if err := session.Run(`su - postgres -c "/usr/lib/postgresql/16/bin/pg_ctl stop -D /var/lib/postgresql/data"`); err != nil {
		s.logger.Error("Execute command error", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, &Response{Error: err})
	}

	errInfo := stdErrBuff.String()
	if len(errInfo) != 0 {
		s.logger.Error("Stop postgresql service error", zap.Error(errors.New(errInfo)))
		return c.JSON(http.StatusInternalServerError, &Response{Error: errors.New(errInfo)})
	}

	return c.NoContent(http.StatusOK)
}

func (s *Service) VaccumTable(c echo.Context) error {
	ctx := c.Request().Context()

	tableName := c.QueryParam("table_name")
	vacuumType := c.QueryParam("type")

	if len(tableName) == 0 {
		return c.JSON(http.StatusBadRequest, &Response{Error: errors.New("table name cant be empty")})
	}

	var err error
	switch vacuumType {
	case "standart":
		err = s.monitoredDB.Vacuum(ctx, tableName)
	case "full":
		err = s.monitoredDB.VacuumFull(ctx, tableName)
	default:
		return c.JSON(http.StatusBadRequest, &Response{Error: errors.New("vacuum type invalid")})
	}

	if err != nil {
		s.logger.Error("Vacuum error", zap.Error(err))
		return c.JSON(http.StatusBadRequest, &Response{Error: err})
	}

	return c.NoContent(http.StatusOK)
}
