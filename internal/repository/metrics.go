package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repo struct {
	db *pgxpool.Pool
}

func NewRepo(db *pgxpool.Pool) *Repo {
	return &Repo{
		db: db,
	}
}

func (r *Repo) Ping(ctx context.Context) error {
	return r.db.Ping(ctx)
}

// TODO: add work with many databases
func (r *Repo) TransactionsNumber(ctx context.Context) (int, int, error) {
	var commits, rollbacks int
	err := r.db.QueryRow(ctx, `SELECT xact_commit, xact_rollback FROM pg_stat_database`).Scan(&commits, &rollbacks)
	if err != nil {
		return 0, 0, err
	}

	return commits, rollbacks, nil
}

func (r *Repo) TotalExecutionTime(ctx context.Context) (float64, error) {
	var total float64
	err := r.db.QueryRow(ctx, `SELECT SUM(total_exec_time) FROM pg_stat_statements`).Scan(&total)
	if err != nil {
		return 0, err
	}

	return total, nil
}

func (r *Repo) TotalCalls(ctx context.Context) (int, error) {
	var total int
	err := r.db.QueryRow(ctx, `SELECT SUM(calls) FROM pg_stat_statements`).Scan(&total)
	if err != nil {
		return 0, err
	}

	return total, nil
}

func (r *Repo) TotalConns(ctx context.Context) (int, error) {
	var conns int
	err := r.db.QueryRow(ctx, `SELECT count(datid) FROM pg_stat_activity`).Scan(&conns)
	if err != nil {
		return 0, err
	}

	return conns, nil
}

func (r *Repo) TotalIdleConns(ctx context.Context) (int, error) {
	var conns int
	err := r.db.QueryRow(ctx, `SELECT count(datid) FROM pg_stat_activity WHERE state LIKE '%idle%' `).Scan(&conns)
	if err != nil {
		return 0, err
	}

	return conns, nil
}

func (r *Repo) TotalLockIdleConns(ctx context.Context) (int, error) {
	var conns int
	err := r.db.QueryRow(ctx, `SELECT count(datid) FROM pg_stat_activity WHERE state LIKE '%idle%' AND wait_event_type LIKE '%idle%'`).Scan(&conns)
	if err != nil {
		return 0, err
	}

	return conns, nil
}

func (r *Repo) TotalDiskUsageByDB(ctx context.Context, dbName string) (uint64, error) {
	var diskUsage uint64
	err := r.db.QueryRow(ctx, `select pg_database_size($1)`, dbName).Scan(&diskUsage)
	if err != nil {
		return 0, err
	}

	return diskUsage, nil
}

func (r *Repo) ListConnsByDatabase(ctx context.Context, dbName string) ([]Conn, error) {
	var conns []Conn

	rows, err := r.db.Query(ctx, `SELECT query, query_start, pid FROM pg_stat_activity WHERE datname=$1`, dbName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		conn := &Conn{}
		err := rows.Scan(&conn.LastQuery, &conn.QuertStart, &conn.PID)
		if err != nil {
			return nil, err
		}

		conns = append(conns, *conn)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return conns, nil
}
