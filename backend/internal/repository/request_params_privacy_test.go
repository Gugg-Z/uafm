package repository

import (
	"encoding/json"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestPrepareUsageLogInsertProtectsRequestParams(t *testing.T) {
	log := &service.UsageLog{
		UserID:    1,
		APIKeyID:  2,
		AccountID: 3,
		RequestID: "req-privacy",
		Model:     "gpt-5.5",
		RequestParams: map[string]any{
			"model":          "gpt-5.5",
			"prompt_preview": "private prompt",
		},
	}

	prepared := prepareUsageLogInsert(log)
	raw, ok := prepared.args[len(prepared.args)-1].(string)
	require.True(t, ok)
	require.NotContains(t, raw, "private prompt")

	var decoded map[string]any
	require.NoError(t, json.Unmarshal([]byte(raw), &decoded))
	require.Equal(t, "gpt-5.5", decoded["model"])
	require.NotContains(t, decoded, "prompt_preview")
	require.NotContains(t, decoded, "_encrypted_sensitive_request_params")
}

func TestPrepareUsageLogInsertNilRequestParams(t *testing.T) {
	prepared := prepareUsageLogInsert(&service.UsageLog{
		UserID:    1,
		APIKeyID:  2,
		AccountID: 3,
		RequestID: "req-no-params",
		Model:     "gpt-5.5",
	})

	require.Nil(t, prepared.args[len(prepared.args)-1])
}
