package handler

import (
	"context"
	"mime"
	"net/http"
	"strings"
	"time"

	middleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// SeedanceTasks exposes Ark's native asynchronous video task protocol.
func (h *OpenAIGatewayHandler) SeedanceTasks(c *gin.Context) {
	c.Set("seedance_billing_mode", string(service.BillingModeToken))
	h.seedanceTaskRequest(c)
}

// SeedanceVideos exposes the generic /v1/videos surface while retaining the
// OpenAI-platform Seedance account selection and Ark forwarding.
func (h *OpenAIGatewayHandler) SeedanceVideos(c *gin.Context) {
	c.Set("seedance_billing_mode", string(service.BillingModeVideo))
	h.seedanceTaskRequest(c)
}

func (h *OpenAIGatewayHandler) seedanceTaskRequest(c *gin.Context) {
	if c.Request.Method == http.MethodPost && h != nil && h.gatewayService != nil {
		if idem := strings.TrimSpace(c.GetHeader("Idempotency-Key")); idem != "" {
			if key, ok := middleware.GetAPIKeyFromContext(c); ok {
				if existing, err := h.gatewayService.LoadSeedanceIdempotentTask(c.Request.Context(), idem, key.User.ID, key.ID); err == nil && existing != nil && existing.TaskID != "" {
					c.JSON(http.StatusOK, gin.H{"id": strings.TrimPrefix(existing.TaskID, "seedance:"), "status": existing.TerminalStatus})
					return
				}
			}
		}
	}
	if (c.Request.Method == http.MethodGet || c.Request.Method == http.MethodDelete) && !isSeedanceContentRequest(c) && h != nil && h.gatewayService != nil && !h.gatewayService.VideoTasksEnabled() {
		if key, ok := middleware.GetAPIKeyFromContext(c); ok {
			taskID := service.SeedanceTaskKey(c.Param("task_id"))
			if taskID == "seedance:" {
				taskID = service.SeedanceTaskKey(c.Param("request_id"))
			}
			if pending, err := h.gatewayService.LoadGrokVideoPendingBilling(c.Request.Context(), taskID, key.User.ID, key.ID); err == nil && pending != nil && pending.TerminalStatus != "" {
				c.JSON(http.StatusOK, gin.H{"id": strings.TrimPrefix(taskID, "seedance:"), "status": pending.TerminalStatus, "error": pending.TerminalError})
				return
			}
		}
	}
	if c.Request.Method == http.MethodPost && c.GetHeader("Content-Type") != "" {
		mediaType, _, err := mime.ParseMediaType(c.GetHeader("Content-Type"))
		if err != nil || mediaType != "application/json" {
			h.errorResponse(c, http.StatusUnsupportedMediaType, "invalid_request_error", "Seedance requires application/json")
			return
		}
	}
	key, ok := middleware.GetAPIKeyFromContext(c)
	if !ok || key.Group == nil || (key.Group.Platform != service.PlatformOpenAI && key.Group.Platform != service.PlatformComposite) {
		h.errorResponse(c, http.StatusForbidden, "permission_error", "Seedance requires an OpenAI or composite group")
		return
	}
	endpoint := service.SeedanceEndpointCreate
	setActualUpstreamEndpoint(c, EndpointSeedanceTasks)
	taskID := ""
	if c.Request.Method != http.MethodPost {
		rawTaskID := c.Param("task_id")
		if rawTaskID == "" {
			rawTaskID = c.Param("request_id")
		}
		taskID = service.SeedanceTaskKey(rawTaskID)
		endpoint = service.SeedanceEndpointStatus
		if c.Request.Method == http.MethodDelete {
			endpoint = service.SeedanceEndpointDelete
		} else if isSeedanceContentRequest(c) {
			endpoint = service.SeedanceEndpointContent
		}
		if endpoint == service.SeedanceEndpointStatus {
			if proxyURL := service.SeedanceContentProxyURL(c, rawTaskID); proxyURL != "" {
				c.Set(service.SeedanceContentProxyURLContextKey, proxyURL)
			}
		}
	}
	h.handleGrokMedia(c, endpoint, taskID)
}

func isSeedanceContentRequest(c *gin.Context) bool {
	if c == nil || c.Request == nil || c.Request.URL == nil {
		return false
	}
	return strings.HasSuffix(strings.TrimRight(c.Request.URL.Path, "/"), "/content")
}

// Ark reports actual completion tokens. Never infer tokens from duration or use
// Grok's per-second video tariff. Repeated polls share the durable task dedup key.
func prepareSeedanceCompletionBilling(ctx context.Context, h *OpenAIGatewayHandler, key *service.APIKey, subject middleware.AuthSubject, taskID string, result *service.OpenAIForwardResult) *service.OpenAIForwardResult {
	if result == nil || (result.Usage.OutputTokens <= 0 && result.VideoCount <= 0) {
		return nil
	}
	pending, err := h.gatewayService.LoadGrokVideoPendingBilling(ctx, taskID, subject.UserID, key.ID)
	if err != nil || pending == nil {
		return nil
	}
	claimed, err := h.gatewayService.ClaimGrokVideoBilling(ctx, taskID, subject.UserID, key.ID)
	if err != nil || !claimed {
		return nil
	}
	merged := *result
	merged.Model = pending.Model
	merged.BillingModel = firstNonEmptyString(pending.BillingModel, pending.Model)
	merged.UpstreamModel = firstNonEmptyString(pending.UpstreamModel, result.UpstreamModel)
	// Restore creation-time mode; a different polling endpoint must not alter it.
	if pending.BillingMode != "" {
		merged.BillingModeOverride = pending.BillingMode
	}
	if merged.BillingModeOverride == string(service.BillingModeVideo) {
		merged.VideoCount = 1
		if strings.TrimSpace(merged.VideoResolution) == "" {
			merged.VideoResolution = pending.VideoResolution
		}
		if merged.VideoDurationSeconds <= 0 {
			merged.VideoDurationSeconds = pending.VideoDurationSeconds
		}
	}
	merged.RequestID = service.StableGrokVideoBillingRequestID(taskID)
	merged.ResponseID = taskID
	merged.Duration = service.GrokVideoE2EDuration(pending.CreatedAt, time.Now())
	return &merged
}
