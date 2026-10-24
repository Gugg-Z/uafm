//go:build unit

package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestSubAdminAdminPermissionGuard(t *testing.T) {
	tests := []struct {
		name        string
		role        string
		permissions []string
		method      string
		path        string
		wantStatus  int
		wantBody    string
	}{
		{
			name:       "super_admin_not_limited",
			role:       service.RoleAdmin,
			method:     http.MethodPost,
			path:       "/api/v1/admin/users",
			wantStatus: http.StatusOK,
		},
		{
			name:        "sub_admin_authorized_get_allowed",
			role:        service.RoleSubAdmin,
			permissions: []string{"admin_users"},
			method:      http.MethodGet,
			path:        "/api/v1/admin/users",
			wantStatus:  http.StatusOK,
		},
		{
			name:        "sub_admin_batch_query_post_allowed",
			role:        service.RoleSubAdmin,
			permissions: []string{"admin_dashboard"},
			method:      http.MethodPost,
			path:        "/api/v1/admin/dashboard/users-usage",
			wantStatus:  http.StatusOK,
		},
		{
			name:        "sub_admin_non_whitelisted_post_denied",
			role:        service.RoleSubAdmin,
			permissions: []string{"admin_dashboard"},
			method:      http.MethodPost,
			path:        "/api/v1/admin/affiliates/settlements",
			wantStatus:  http.StatusForbidden,
			wantBody:    "READ_ONLY_ADMIN",
		},
		{
			name:        "sub_admin_ops_requests_uses_requests_permission",
			role:        service.RoleSubAdmin,
			permissions: []string{"admin_requests"},
			method:      http.MethodGet,
			path:        "/api/v1/admin/ops/requests",
			wantStatus:  http.StatusOK,
		},
		{
			name:        "sub_admin_ops_other_paths_need_ops_permission",
			role:        service.RoleSubAdmin,
			permissions: []string{"admin_requests"},
			method:      http.MethodGet,
			path:        "/api/v1/admin/ops/dashboard/overview",
			wantStatus:  http.StatusForbidden,
			wantBody:    "MENU_FORBIDDEN",
		},
		{
			name:        "sub_admin_channel_monitor_v2_allowed",
			role:        service.RoleSubAdmin,
			permissions: []string{"admin_channel_monitor"},
			method:      http.MethodGet,
			path:        "/api/v1/admin/channel-monitor-v2/matrix",
			wantStatus:  http.StatusOK,
		},
		{
			name:        "sub_admin_grok_accounts_use_accounts_permission",
			role:        service.RoleSubAdmin,
			permissions: []string{"admin_accounts"},
			method:      http.MethodGet,
			path:        "/api/v1/admin/grok/accounts",
			wantStatus:  http.StatusOK,
		},
		{
			name:        "sub_admin_affiliate_settlements_use_usage_permission",
			role:        service.RoleSubAdmin,
			permissions: []string{"admin_affiliate_usage"},
			method:      http.MethodGet,
			path:        "/api/v1/admin/affiliates/settlements",
			wantStatus:  http.StatusOK,
		},
		{
			name:        "sub_admin_affiliate_fallback_uses_usage_permission",
			role:        service.RoleSubAdmin,
			permissions: []string{"admin_affiliate_usage"},
			method:      http.MethodGet,
			path:        "/api/v1/admin/affiliates/partner-tiers",
			wantStatus:  http.StatusOK,
		},
		{
			name:        "sub_admin_audit_logs_covers_compliance",
			role:        service.RoleSubAdmin,
			permissions: []string{"admin_audit_logs"},
			method:      http.MethodGet,
			path:        "/api/v1/admin/compliance",
			wantStatus:  http.StatusOK,
		},
		{
			name:        "sub_admin_trial_cards_alias_maps_to_redeem",
			role:        service.RoleSubAdmin,
			permissions: []string{"admin_redeem"},
			method:      http.MethodGet,
			path:        "/api/v1/admin/trial-cards",
			wantStatus:  http.StatusOK,
		},
		{
			name:        "sub_admin_unmapped_path_denied",
			role:        service.RoleSubAdmin,
			permissions: []string{"admin_users"},
			method:      http.MethodGet,
			path:        "/api/v1/admin/unknown-feature",
			wantStatus:  http.StatusForbidden,
			wantBody:    "MENU_FORBIDDEN",
		},
		{
			name:        "sub_admin_missing_menu_denied",
			role:        service.RoleSubAdmin,
			permissions: []string{"admin_groups"},
			method:      http.MethodGet,
			path:        "/api/v1/admin/users",
			wantStatus:  http.StatusForbidden,
			wantBody:    "MENU_FORBIDDEN",
		},
		{
			name:        "sub_admin_write_denied",
			role:        service.RoleSubAdmin,
			permissions: []string{"admin_users"},
			method:      http.MethodPost,
			path:        "/api/v1/admin/users",
			wantStatus:  http.StatusForbidden,
			wantBody:    "READ_ONLY_ADMIN",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			code, body := runSubAdminGuardRequest(SubAdminAdminPermissionGuard(), tc.role, tc.permissions, tc.method, tc.path)
			require.Equal(t, tc.wantStatus, code)
			if tc.wantBody != "" {
				require.Contains(t, body, tc.wantBody)
			}
		})
	}
}

func runSubAdminGuardRequest(guard gin.HandlerFunc, role string, permissions []string, method string, path string) (int, string) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(ContextKeyUserRole), role)
		if permissions != nil {
			c.Set(string(ContextKeyAdminMenuPermissions), permissions)
		}
		c.Next()
	})
	router.Use(guard)
	router.Any("/*path", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, nil)
	router.ServeHTTP(w, req)
	return w.Code, w.Body.String()
}
