//go:build unit

package service

import (
	"context"
	"encoding/json"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSupportContactConfigNormalization(t *testing.T) {
	raw := NormalizeSupportContactConfigJSON(`{"enabled":false,"title":" Support ","contacts":[{}, {"name":" A ","qr_image":" /a.png "},{"name":"B"},{"name":"C"},{"name":"D"}]}`)
	var cfg SupportContactConfig
	require.NoError(t, json.Unmarshal([]byte(raw), &cfg))
	require.False(t, cfg.Enabled)
	require.Equal(t, "Support", cfg.Title)
	require.Len(t, cfg.Contacts, 3)
	require.Equal(t, "/a.png", cfg.Contacts[0].QRImage)
	require.JSONEq(t, defaultSupportContactConfigJSON, NormalizeSupportContactConfigJSON(`invalid`))
}

func TestSupportContactSettingsRoundTrip(t *testing.T) {
	repo := &settingUpdateRepoStub{}
	svc := NewSettingService(repo, &config.Config{})
	settings := &SystemSettings{SupportContactConfig: `{"enabled":true,"contacts":[{"name":"Agent","qr_image":"/qr.png"}]}`, UserMenuItems: `["api_keys"]`}
	require.NoError(t, svc.UpdateSettings(context.Background(), settings))
	require.Contains(t, repo.updates[SettingKeySupportContactConfig], `"Agent"`)
	require.Equal(t, `["api_keys"]`, repo.updates[SettingKeyUserMenuItems])
}
