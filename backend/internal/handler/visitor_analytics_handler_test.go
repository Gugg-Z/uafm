package handler

import (
	clientip "github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"net/http/httptest"
	"testing"
)

func TestVisitorClientIPUsesConfiguredProxyPolicy(t *testing.T) {
	for _, trust := range []bool{false, true} {
		r := gin.New()
		require.NoError(t, r.SetTrustedProxies(nil))
		var resolved string
		r.POST("/", func(c *gin.Context) { clientip.SetForwardedIPSettings(c, trust, nil); resolved = visitorClientIP(c) })
		req := httptest.NewRequest("POST", "/", nil)
		req.RemoteAddr = "203.0.113.20:1234"
		req.Header.Set("X-Real-IP", "198.51.100.9")
		r.ServeHTTP(httptest.NewRecorder(), req)
		if trust {
			require.Equal(t, "198.51.100.9", resolved)
		} else {
			require.Equal(t, "203.0.113.20", resolved)
		}
	}
}

func TestVisitorDateRangeRejectsReversedAndInvalidDates(t *testing.T) {
	for _, query := range []string{"start_date=2026-10-25&end_date=2026-10-24", "timezone=invalid", "start_date=2020-01-01&end_date=2026-10-25"} {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("GET", "/?"+query, nil)
		_, _, _, err := visitorDateRange(c)
		require.Error(t, err)
	}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/?start_date=2026-10-25&end_date=2026-10-25&timezone=Asia%2FShanghai", nil)
	start, end, _, err := visitorDateRange(c)
	require.NoError(t, err)
	require.Equal(t, int64(86400), int64(end.Sub(start).Seconds()))
}
