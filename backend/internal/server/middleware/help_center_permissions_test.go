//go:build unit

package middleware

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestHelpCenterAdminMenuMapping(t *testing.T) {
	for _, path := range []string{"/api/v1/admin/help-center", "/api/v1/admin/help-center/publish", "/api/v1/admin/settings/help-center/attachments"} {
		require.Equal(t, "admin_settings", adminMenuKeyForAPIPath(path))
	}
}
