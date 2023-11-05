package repository

import "context"

func (r *Repo) CreateCheckpoint(ctx context.Context) error {
	_, err := r.db.Exec(ctx, `CHECKPOINT`)
	if err != nil {
		return err
	}

	return nil
}

func (r *Repo) TerminateConnByPid(ctx context.Context, pid int) (bool, error) {
	var success bool
	err := r.db.QueryRow(ctx, `SELECT pg_terminate_backend($1)`, pid).Scan(&success)
	if err != nil {
		return false, err
	}

	return success, err
}

func (r *Repo) Vacuum(ctx context.Context, tableName string) error {
	_, err := r.db.Exec(ctx, `VACUUM `+ tableName)
	if err != nil {
		return err
	}

	return nil
}

func (r *Repo) VacuumFull(ctx context.Context, tableName string) error {
	_, err := r.db.Exec(ctx, `VACUUM FULL `+ tableName)
	if err != nil {
		return err
	}

	return nil
}
