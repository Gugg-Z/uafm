package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type videoTaskRepository struct{ db *sql.DB }

func NewVideoTaskRepository(db *sql.DB) service.VideoTaskRepository { return &videoTaskRepository{db} }

func (r *videoTaskRepository) Create(ctx context.Context, t *service.VideoTask, limit int) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// Serialize admissions across all groups and instances for this user.
	if _, err = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, fmt.Sprintf("video:%d", t.UserID)); err != nil {
		return err
	}
	var count int
	query := `SELECT COUNT(*) FROM video_tasks WHERE user_id=$1 AND active`
	args := []any{t.UserID}
	if !t.GlobalPendingLimit {
		query += ` AND group_id=$2`
		args = append(args, t.GroupID)
	}
	if err = tx.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return err
	}
	if count >= limit {
		return service.ErrVideoTaskLimit
	}
	b, err := json.Marshal(t)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO video_tasks(id,user_id,api_key_id,group_id,upstream_id,next_poll_at,payload) VALUES($1,$2,$3,$4,$5,$6,$7)`, t.ID, t.UserID, t.APIKeyID, t.GroupID, t.UpstreamID, t.NextPollAt, string(b))
	if err != nil {
		return err
	}
	return tx.Commit()
}
func scanVideoTask(row interface{ Scan(...any) error }) (*service.VideoTask, error) {
	var b []byte
	if err := row.Scan(&b); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	var t service.VideoTask
	if err := json.Unmarshal(b, &t); err != nil {
		return nil, err
	}
	return &t, nil
}
func (r *videoTaskRepository) Get(ctx context.Context, id string) (*service.VideoTask, error) {
	return scanVideoTask(r.db.QueryRowContext(ctx, `SELECT payload FROM video_tasks WHERE id=$1`, id))
}
func (r *videoTaskRepository) Find(ctx context.Context, u, k, g int64, id string) (*service.VideoTask, error) {
	return scanVideoTask(r.db.QueryRowContext(ctx, `SELECT payload FROM video_tasks WHERE user_id=$1 AND api_key_id=$2 AND group_id=$3 AND (upstream_id=$4 OR upstream_id='seedance:'||$4 OR upstream_id='grok-video:'||$4)`, u, k, g, id))
}
func (r *videoTaskRepository) Due(ctx context.Context, n int) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id FROM video_tasks WHERE active AND next_poll_at<=NOW() AND (lease_until IS NULL OR lease_until<NOW()) ORDER BY next_poll_at LIMIT $1`, n)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
func (r *videoTaskRepository) Claim(ctx context.Context, id, token string, ttl time.Duration) (bool, error) {
	result, err := r.db.ExecContext(ctx, `UPDATE video_tasks SET lease_until=NOW()+($3 * INTERVAL '1 second'),lease_token=$2 WHERE id=$1 AND active AND (lease_until IS NULL OR lease_until<NOW())`, id, token, ttl.Seconds())
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	return n == 1, err
}
func (r *videoTaskRepository) Save(ctx context.Context, t *service.VideoTask) error {
	b, err := json.Marshal(t)
	if err != nil {
		return err
	}
	result, err := r.db.ExecContext(ctx, `UPDATE video_tasks SET upstream_id=$2,active=$3,next_poll_at=$4,payload=$5 WHERE id=$1 AND lease_token=$6 AND lease_until>NOW()`, t.ID, t.UpstreamID, !t.Closed && !t.Settled, t.NextPollAt, string(b), t.LeaseToken)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err == nil && n != 1 {
		return errors.New("video task lease lost")
	}
	return err
}

func (r *videoTaskRepository) Release(ctx context.Context, id, token string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE video_tasks SET lease_until=NULL,lease_token='' WHERE id=$1 AND lease_token=$2`, id, token)
	return err
}
