package repository

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestOpsRepositoryListRequestDetails_LatencySort(t *testing.T) {
	for _, tc := range []struct {
		name  string
		sort  string
		order string
	}{
		{name: "TTFT", sort: "ttft_desc", order: "first_token_ms DESC NULLS LAST, created_at DESC"},
		{name: "duration", sort: "duration_desc", order: "duration_ms DESC NULLS LAST, created_at DESC"},
		{name: "default", order: "created_at DESC"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, mock := newSQLMock(t)
			repo := &opsRepository{db: db}
			start := time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC)
			end := start.Add(time.Hour)
			filter := &service.OpsRequestDetailFilter{
				StartTime: &start,
				EndTime:   &end,
				Sort:      tc.sort,
				Page:      2,
				PageSize:  10,
			}

			mock.ExpectQuery(`SELECT COUNT\(1\) FROM combined`).
				WithArgs(start, end).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(13))
			rows := sqlmock.NewRows([]string{
				"kind", "created_at", "request_id", "platform", "model", "duration_ms", "first_token_ms",
				"status_code", "error_id", "phase", "severity", "message", "user_id", "api_key_id", "account_id", "group_id",
				"user_email", "username", "api_key_name", "account_name", "group_name",
				"stream", "openai_ws_mode", "request_type", "inbound_endpoint", "upstream_endpoint",
				"requested_model", "upstream_model", "request_params",
			}).
				AddRow("success", start, "req-slow", "openai", "gpt-5.5", 12000, 800, nil, nil, nil, nil, nil, 1, 2, 3, 4,
					"user@example.com", "alice", "key-a", "account-a", "group-a",
					true, false, 2, "/v1/chat/completions", "/chat/completions",
					"gpt-5.5", "gpt-5.5-upstream", `{"model":"gpt-5.5","stream":true,"prompt_preview":"secret","image_count":2}`).
				AddRow("error", start, "req-zero", "openai", "gpt-5.5", 9000, 0, 502, 5, "upstream", "error", "failed", 1, 2, 3, 4,
					"user@example.com", "alice", "key-a", "account-a", "group-a",
					true, false, nil, "/v1/chat/completions", "/chat/completions",
					"gpt-5.5", "", nil).
				AddRow("success", start, "req-missing", "openai", "gpt-5.5", 5000, nil, nil, nil, nil, nil, nil, 1, 2, 3, 4,
					"", "", "", "", "",
					false, false, nil, "", "",
					"gpt-5.5", "", nil)
			mock.ExpectQuery(`(?s)ul\.first_token_ms AS first_token_ms.*o\.time_to_first_token_ms AS first_token_ms.*SELECT.*duration_ms,\s+first_token_ms,.*ORDER BY `+tc.order+`\s+LIMIT \$3 OFFSET \$4`).
				WithArgs(start, end, 10, 10).
				WillReturnRows(rows)

			items, total, err := repo.ListRequestDetails(context.Background(), filter)
			require.NoError(t, err)
			require.EqualValues(t, 13, total)
			require.Len(t, items, 3)
			require.NotNil(t, items[0].FirstTokenMs)
			require.Equal(t, 800, *items[0].FirstTokenMs)
			require.Equal(t, 12000, *items[0].DurationMs)
			require.NotNil(t, items[1].FirstTokenMs)
			require.Zero(t, *items[1].FirstTokenMs)
			require.Nil(t, items[2].FirstTokenMs)
			require.Equal(t, "user@example.com", items[0].UserEmail)
			require.Equal(t, "key-a", items[0].APIKeyName)
			require.Equal(t, "stream", items[0].RequestType)
			require.Equal(t, "gpt-5.5-upstream", items[0].UpstreamModel)
			require.Equal(t, "stream", items[1].RequestType)
			require.Equal(t, "sync", items[2].RequestType)
			require.Nil(t, items[1].RequestParams)
			require.Nil(t, items[2].RequestParams)
			require.Equal(t, map[string]any{"model": "gpt-5.5", "stream": true, "image_count": float64(2)}, items[0].RequestParams)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestOpsRepositoryListRequestFilterOptions(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &opsRepository{db: db}
	start := time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	filter := &service.OpsRequestDetailFilter{StartTime: &start, EndTime: &end, Kind: "success"}

	optionRows := func(value, label string) *sqlmock.Rows {
		return sqlmock.NewRows([]string{"value", "label"}).AddRow(value, label)
	}
	for _, expectation := range []struct {
		value string
		label string
	}{
		{value: "openai", label: "OPENAI"},
		{value: "gpt-5.5", label: "gpt-5.5"},
		{value: "7", label: "user@example.com (#7)"},
		{value: "9", label: "key-a (#9)"},
		{value: "11", label: "account-a (#11)"},
		{value: "13", label: "group-a (#13)"},
	} {
		mock.ExpectQuery(`SELECT DISTINCT[\s\S]*FROM combined\s+WHERE kind = \$3`).
			WithArgs(start, end, "success").
			WillReturnRows(optionRows(expectation.value, expectation.label))
	}

	out, err := repo.ListRequestFilterOptions(context.Background(), filter)
	require.NoError(t, err)
	require.NotNil(t, out)
	require.Equal(t, []service.OpsRequestFilterOption{{Value: "openai", Label: "OPENAI"}}, out.Platforms)
	require.Equal(t, []service.OpsRequestFilterOption{{Value: "gpt-5.5", Label: "gpt-5.5"}}, out.Models)
	require.Equal(t, []service.OpsRequestFilterOption{{Value: "7", Label: "user@example.com (#7)"}}, out.Users)
	require.Equal(t, []service.OpsRequestFilterOption{{Value: "9", Label: "key-a (#9)"}}, out.APIKeys)
	require.Equal(t, []service.OpsRequestFilterOption{{Value: "11", Label: "account-a (#11)"}}, out.Accounts)
	require.Equal(t, []service.OpsRequestFilterOption{{Value: "13", Label: "group-a (#13)"}}, out.Groups)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestOpsRepositoryListRequestFilterOptionsInvalidKind(t *testing.T) {
	db, _ := newSQLMock(t)
	repo := &opsRepository{db: db}

	_, err := repo.ListRequestFilterOptions(context.Background(), &service.OpsRequestDetailFilter{Kind: "bogus"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid kind")
}
