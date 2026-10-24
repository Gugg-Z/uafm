// Package middleware sub_admin_permissions enforces menu-level read-only access
// for delegated administrators (role=sub_admin).
//
// Ported from 58token commit 9c6e6d5 (final HEAD semantics: admin-scope guard
// only; user-scope guard was removed upstream in c886730). The path→menu-key
// table below is rebuilt against this codebase's actual /admin route tree.
package middleware

import (
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// SubAdminAdminPermissionGuard gates /admin APIs for sub_admin accounts:
//   - write methods are always denied (read-only by design);
//   - read methods require the matching menu key granted to the user;
//   - unmapped admin paths are denied (fail-closed).
func SubAdminAdminPermissionGuard() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := GetUserRoleFromContext(c)
		if role != service.RoleSubAdmin {
			c.Next()
			return
		}

		if !isReadOnlyMethod(c.Request.Method) && !isReadOnlyAdminQueryEndpoint(c.Request.Method, c.Request.URL.Path) {
			AbortWithError(c, http.StatusForbidden, "READ_ONLY_ADMIN", "This admin role has read-only access")
			return
		}

		key := adminMenuKeyForAPIPath(c.Request.URL.Path)
		if key == "" || !hasAdminMenuPermission(c, key) {
			AbortWithError(c, http.StatusForbidden, "MENU_FORBIDDEN", "Menu permission required")
			return
		}

		c.Next()
	}
}

func isReadOnlyMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	default:
		return false
	}
}

// isReadOnlyAdminQueryEndpoint whitelists POST endpoints that are semantically
// read queries (batch lookups), so read-only admins can still use them.
func isReadOnlyAdminQueryEndpoint(method, path string) bool {
	if method != http.MethodPost {
		return false
	}
	p := trimAPIV1(path)
	switch p {
	case "/admin/dashboard/users-usage",
		"/admin/dashboard/api-keys-usage",
		"/admin/accounts/today-stats/batch",
		"/admin/user-attributes/batch",
		"/admin/visitor-analytics/ip-lookup":
		return true
	default:
		return false
	}
}

func hasAdminMenuPermission(c *gin.Context, key string) bool {
	items, ok := GetAdminMenuPermissionsFromContext(c)
	if !ok || key == "" {
		return false
	}
	for _, item := range items {
		if item == key {
			return true
		}
	}
	return false
}

// adminMenuKeyForAPIPath maps an admin API path to the sidebar menu key that
// gates it. Longer prefixes must be matched before their shorter parents.
func adminMenuKeyForAPIPath(path string) string {
	p := trimAPIV1(path)
	if !strings.HasPrefix(p, "/admin/") {
		return ""
	}
	p = strings.TrimPrefix(p, "/admin")

	switch {
	case strings.HasPrefix(p, "/visitor-analytics"):
		return "admin_visitor_analytics"
	case strings.HasPrefix(p, "/download-resources"):
		return "admin_download_resources"
	case strings.HasPrefix(p, "/dashboard"):
		return "admin_dashboard"
	case strings.HasPrefix(p, "/ops/requests"):
		return "admin_requests"
	case strings.HasPrefix(p, "/ops"):
		return "admin_ops"
	case strings.HasPrefix(p, "/users"):
		return "admin_users"
	case strings.HasPrefix(p, "/user-attributes"):
		return "admin_users"
	case strings.HasPrefix(p, "/groups"):
		return "admin_groups"
	case strings.HasPrefix(p, "/channels"):
		return "admin_channel_pricing"
	case strings.HasPrefix(p, "/channel-monitors"),
		strings.HasPrefix(p, "/channel-monitor-templates"),
		strings.HasPrefix(p, "/channel-monitor-v2"):
		return "admin_channel_monitor"
	case strings.HasPrefix(p, "/subscriptions"):
		return "admin_subscriptions"
	case strings.HasPrefix(p, "/accounts"),
		strings.HasPrefix(p, "/openai"),
		strings.HasPrefix(p, "/gemini"),
		strings.HasPrefix(p, "/antigravity"),
		strings.HasPrefix(p, "/grok"),
		strings.HasPrefix(p, "/cn-providers"),
		strings.HasPrefix(p, "/api-keys"):
		return "admin_accounts"
	case strings.HasPrefix(p, "/announcements"):
		return "admin_announcements"
	case strings.HasPrefix(p, "/proxies"):
		return "admin_proxies"
	case strings.HasPrefix(p, "/plugins"):
		return "admin_plugins"
	case strings.HasPrefix(p, "/risk-control"):
		return "admin_risk_control"
	case strings.HasPrefix(p, "/prompt-audit"):
		return "admin_prompt_audit"
	case strings.HasPrefix(p, "/audit-logs"), strings.HasPrefix(p, "/compliance"):
		return "admin_audit_logs"
	case strings.HasPrefix(p, "/redeem-codes"),
		strings.HasPrefix(p, "/trial-cards"),
		strings.HasPrefix(p, "/free-quota"):
		return "admin_redeem"
	case strings.HasPrefix(p, "/promo-codes"):
		return "admin_promo_codes"
	case strings.HasPrefix(p, "/affiliates/usage"),
		strings.HasPrefix(p, "/affiliates/settlements"),
		strings.HasPrefix(p, "/affiliates/partner-tiers"),
		strings.HasPrefix(p, "/affiliates/users"):
		return "admin_affiliate_usage"
	case strings.HasPrefix(p, "/affiliates/partner-applications"):
		return "admin_affiliate_applications"
	case strings.HasPrefix(p, "/affiliates/invites"):
		return "admin_affiliate_invites"
	case strings.HasPrefix(p, "/affiliates/rebates"):
		return "admin_affiliate_rebates"
	case strings.HasPrefix(p, "/affiliates/transfers"):
		return "admin_affiliate_transfers"
	case strings.HasPrefix(p, "/affiliates"):
		return "admin_affiliate_usage"
	case strings.HasPrefix(p, "/payment/dashboard"),
		strings.HasPrefix(p, "/payment/config"),
		strings.HasPrefix(p, "/payment/providers"):
		return "admin_order_dashboard"
	case strings.HasPrefix(p, "/payment/orders"):
		return "admin_orders"
	case strings.HasPrefix(p, "/payment/plans"):
		return "admin_order_plans"
	case strings.HasPrefix(p, "/usage"):
		return "admin_usage"
	case strings.HasPrefix(p, "/help-center"), strings.HasPrefix(p, "/settings"),
		strings.HasPrefix(p, "/system"),
		strings.HasPrefix(p, "/data-management"),
		strings.HasPrefix(p, "/backups"),
		strings.HasPrefix(p, "/scheduled-test-plans"),
		strings.HasPrefix(p, "/error-passthrough-rules"),
		strings.HasPrefix(p, "/tls-fingerprint-profiles"):
		return "admin_settings"
	default:
		return ""
	}
}

func trimAPIV1(path string) string {
	if strings.HasPrefix(path, "/api/v1") {
		return strings.TrimPrefix(path, "/api/v1")
	}
	return path
}
