package repository

import (
	"context"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestVideoTaskAdmissionSerializesPerUserGroup(t *testing.T) {
	db, m, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	r := NewVideoTaskRepository(db)
	m.ExpectBegin()
	m.ExpectExec("SELECT pg_advisory_xact_lock").WithArgs("video:1").WillReturnResult(sqlmock.NewResult(0, 1))
	m.ExpectQuery("SELECT COUNT").WithArgs(int64(1), int64(3)).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))
	m.ExpectRollback()
	err = r.Create(context.Background(), &service.VideoTask{ID: "a", UserID: 1, GroupID: 3}, 5)
	require.ErrorIs(t, err, service.ErrVideoTaskLimit)
	require.NoError(t, m.ExpectationsWereMet())
}
func TestVideoTaskStaleLeaseCannotOverwriteSuccess(t *testing.T) {
	db, m, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	r := NewVideoTaskRepository(db)
	m.ExpectExec("UPDATE video_tasks SET upstream_id=.*lease_token=.*lease_until>NOW").WillReturnResult(sqlmock.NewResult(0, 0))
	err = r.Save(context.Background(), &service.VideoTask{ID: "a", LeaseToken: "old", NextPollAt: time.Now()})
	require.ErrorContains(t, err, "lease lost")
	require.NoError(t, m.ExpectationsWereMet())
}

func TestVideoTaskAdmissionUserOverrideCountsAcrossGroups(t *testing.T) {
	db, m, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	r := NewVideoTaskRepository(db)
	m.ExpectBegin()
	m.ExpectExec("SELECT pg_advisory_xact_lock").WithArgs("video:1").WillReturnResult(sqlmock.NewResult(0, 1))
	m.ExpectQuery(`SELECT COUNT\(\*\) FROM video_tasks WHERE user_id=\$1 AND active$`).WithArgs(int64(1)).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(12))
	m.ExpectRollback()
	err = r.Create(context.Background(), &service.VideoTask{ID: "a", UserID: 1, GroupID: 9, GlobalPendingLimit: true}, 12)
	require.ErrorIs(t, err, service.ErrVideoTaskLimit)
	require.NoError(t, m.ExpectationsWereMet())
}
