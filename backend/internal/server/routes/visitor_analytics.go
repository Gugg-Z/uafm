package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	appmiddleware "github.com/Wei-Shaw/sub2api/internal/middleware"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"time"
)

func RegisterVisitorAnalyticsRoutes(v1 *gin.RouterGroup, h *handler.Handlers, optionalJWT servermiddleware.OptionalJWTAuthMiddleware, redisClient *redis.Client) {
	limiter := appmiddleware.NewRateLimiter(redisClient)
	v1.POST("/analytics/visit", limiter.LimitWithOptions("visitor-analytics", 180, time.Minute, appmiddleware.RateLimitOptions{FailureMode: appmiddleware.RateLimitFailOpen}), gin.HandlerFunc(optionalJWT), h.VisitorAnalytics.Track)
}
