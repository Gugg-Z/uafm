//go:build unit

package middleware

import (
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
)

func TestVisitorAnalyticsSubAdminPermissions(t *testing.T) {
	for _, tc := range []struct {
		method, path string
		permissions  []string
		status       int
	}{
		{http.MethodGet, "/api/v1/admin/visitor-analytics/events", []string{"admin_visitor_analytics"}, 200},
		{http.MethodGet, "/api/v1/admin/visitor-analytics/events", []string{"admin_dashboard"}, 403},
		{http.MethodPost, "/api/v1/admin/visitor-analytics/ip-lookup", []string{"admin_visitor_analytics"}, 200},
		{http.MethodPost, "/api/v1/admin/visitor-analytics/channels", []string{"admin_visitor_analytics"}, 403},
		{http.MethodPut, "/api/v1/admin/visitor-analytics/settings", []string{"admin_visitor_analytics"}, 403},
	} {
		status, _ := runSubAdminGuardRequest(SubAdminAdminPermissionGuard(), service.RoleSubAdmin, tc.permissions, tc.method, tc.path)
		require.Equal(t, tc.status, status, tc.path)
	}
}
