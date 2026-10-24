package service

import (
	"context"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"regexp"
	"testing"
	"time"
)

func TestVisitorAnalyticsDisabledDoesNotInsert(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	svc := NewVisitorAnalyticsService(db)
	svc.lastCleanupTime = time.Now()
	mock.ExpectQuery("SELECT enabled, retention_days, updated_at").WillReturnRows(sqlmock.NewRows([]string{"enabled", "retention_days", "updated_at"}).AddRow(false, 90, time.Now()))
	require.NoError(t, svc.Track(context.Background(), VisitorTrackInput{}))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestVisitorAnalyticsTrackUnknownChannelAndSanitizeURLs(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	svc := NewVisitorAnalyticsService(db)
	svc.lastCleanupTime = time.Now()
	mock.ExpectQuery("SELECT enabled, retention_days, updated_at").WillReturnRows(sqlmock.NewRows([]string{"enabled", "retention_days", "updated_at"}).AddRow(true, 90, time.Now()))
	mock.ExpectQuery("SELECT id FROM visitor_channels").WithArgs("unknown").WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectExec("INSERT INTO visitor_events").WithArgs(nil, "direct", "visitor", "session", int64(42), "203.0.113.1", "", "/keys", "https://example.com/home?ref=reddit", "https://example.com/keys", "Browser", "en", "100x100", false).WillReturnResult(sqlmock.NewResult(1, 1))
	require.NoError(t, svc.Track(context.Background(), VisitorTrackInput{UserID: 42, ChannelCode: "unknown", VisitorID: "visitor", SessionID: "session", IP: "203.0.113.1", Path: "/keys?token=secret#secret", Referrer: "https://example.com/home?ref=reddit&token=secret", LandingURL: "https://user:pass@example.com/keys?key=secret#secret", UserAgent: "Browser", Language: "en", Screen: "100x100"}))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestVisitorAnalyticsRetentionCleanupWhileDisabled(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	svc := NewVisitorAnalyticsService(db)
	mock.ExpectQuery("SELECT enabled, retention_days, updated_at").WillReturnRows(sqlmock.NewRows([]string{"enabled", "retention_days", "updated_at"}).AddRow(false, 30, time.Now()))
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM visitor_events WHERE occurred_at < NOW() - ($1 * interval '1 day')")).WithArgs(30).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("DELETE FROM visitor_ip_geolocation_cache").WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, svc.Track(context.Background(), VisitorTrackInput{}))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestVisitorAnalyticsURLSanitization(t *testing.T) {
	require.Empty(t, sanitizeVisitorURL("javascript:alert(1)"))
	require.Empty(t, sanitizeVisitorURL("//example.com/path"))
	require.Equal(t, "https://example.com/path?utm_source=reddit", sanitizeVisitorURL("https://name:secret@example.com/path?api_key=secret&utm_source=reddit#token"))
}
