package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tidwall/gjson"
)

// ConfigureVideoTasks is called at application assembly, before requests are served.
func (s *OpenAIGatewayService) ConfigureVideoTasks(repo VideoTaskRepository) { s.videoTasks = repo }
func (s *OpenAIGatewayService) VideoTasksEnabled() bool                      { return s != nil && s.videoTasks != nil }
func (s *OpenAIGatewayService) StartVideoTasks() {
	if !s.VideoTasksEnabled() {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.videoTaskCancel = cancel
	s.videoTaskWG.Add(1)
	go func() {
		defer s.videoTaskWG.Done()
		tick := time.NewTicker(5 * time.Second)
		defer tick.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
				dueCtx, dueCancel := context.WithTimeout(ctx, 5*time.Second)
				ids, err := s.videoTasks.Due(dueCtx, 16)
				dueCancel()
				if err != nil {
					slog.Warn("video_tasks_due_failed", "error", err)
					continue
				}
				var wg sync.WaitGroup
				slots := make(chan struct{}, 4)
				for _, id := range ids {
					select {
					case slots <- struct{}{}:
					case <-ctx.Done():
						wg.Wait()
						return
					}
					wg.Add(1)
					go func(id string) {
						defer wg.Done()
						defer func() { <-slots }()
						if _, err := s.PollVideoTask(ctx, id); err != nil {
							slog.Warn("video_task_poll_failed", "task_id", id, "error", err)
						}
					}(id)
				}
				wg.Wait()
			}
		}
	}()
}
func (s *OpenAIGatewayService) StopVideoTasks() {
	if s.videoTaskCancel != nil {
		s.videoTaskCancel()
		s.videoTaskWG.Wait()
	}
}

func (s *OpenAIGatewayService) PrepareVideoTask(ctx context.Context, key *APIKey, sub *UserSubscription, endpoint GrokMediaEndpoint, model, resolution, mode, inboundEndpoint, upstreamEndpoint string) (*VideoTask, error) {
	if !s.VideoTasksEnabled() {
		return nil, nil
	}
	timeout, limit := key.Group.VideoTaskTimeoutHours, 5
	if key.Group.VideoMaxPendingTasks > 0 {
		limit = key.Group.VideoMaxPendingTasks
	}
	now := time.Now()
	t := &VideoTask{ID: uuid.NewString(), UserID: key.User.ID, APIKeyID: key.ID, GroupID: *key.GroupID, Mode: mode, Model: model, Resolution: resolution, CreatedAt: now, NextPollAt: now.Add(2 * time.Minute), Status: "submitting", Subscription: sub, InboundEndpoint: inboundEndpoint, UpstreamEndpoint: upstreamEndpoint}
	if timeout > 0 {
		t.Deadline = now.Add(time.Duration(timeout) * time.Hour)
	}
	if endpoint.IsSeedance() {
		t.Provider = "seedance"
	} else {
		t.Provider = "grok"
	}
	// Snapshot pricing and quota metadata without persisting bearer keys/user secrets.
	clone := *key
	clone.Key = ""
	clone.User = nil
	clone.Group = nil
	t.Key = &clone
	groupCopy := *key.Group
	groupCopy.AccountGroups = nil
	t.Group = &groupCopy
	if sub != nil {
		subCopy := *sub
		subCopy.User = nil
		subCopy.Group = nil
		subCopy.AssignedByUser = nil
		t.Subscription = &subCopy
	}
	base := s.ResolveUserGroupRateMultiplier(ctx, t.UserID, t.GroupID, key.Group.RateMultiplier)
	t.Multiplier = resolveVideoRateMultiplier(key, base)
	if mode == string(BillingModeToken) && s.resolver != nil {
		t.TokenPricing = s.resolver.Resolve(ctx, PricingInput{Model: model, GroupID: key.GroupID, Group: key.Group})
		if t.TokenPricing == nil || t.TokenPricing.Mode != BillingModeToken || t.TokenPricing.BasePricing == nil {
			return nil, fmt.Errorf("token price not configured for video model")
		}
	}
	if mode == string(BillingModeVideo) {
		t.Prices = make(map[string]float64)
		for _, tier := range []string{"480p", "720p", "1080p"} {
			p := key.Group.GetVideoPriceForModel(model, tier)
			if p != nil {
				t.Prices[tier] = *p
			} else if t.Provider == "grok" {
				if v, ok := getDefaultGrokImagineVideoPrice(model, tier); ok {
					t.Prices[tier] = v
				}
			}
		}
		price := key.Group.GetVideoPriceForModel(model, resolution)
		if price == nil && t.Provider == "grok" {
			v, ok := getDefaultGrokImagineVideoPrice(model, resolution)
			if ok {
				price = &v
			}
		}
		if price == nil {
			return nil, fmt.Errorf("video price is not configured for model %s / %s", model, resolution)
		}
		t.UnitPrice = price
		t.HoldAmount = *price * 15 * t.Multiplier
	}
	// Load current settings instead of a potentially stale auth snapshot.
	user, err := s.userRepo.GetByID(ctx, t.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, fmt.Errorf("video task user unavailable")
	}
	if user.VideoMaxPendingTasks > 0 {
		limit = user.VideoMaxPendingTasks
		t.GlobalPendingLimit = true
	}
	if err := s.videoTasks.Create(ctx, t, limit); err != nil {
		return nil, err
	}
	// Claim the submitting row so the worker cannot observe a half-created task.
	t.LeaseToken = uuid.NewString()
	ok, err := s.videoTasks.Claim(ctx, t.ID, t.LeaseToken, 5*time.Minute)
	if err != nil || !ok {
		return nil, fmt.Errorf("claim new video task: %v", err)
	}
	if err = s.reserveVideoHold(ctx, t, key.User); err != nil {
		t.Closed = true
		t.Status = "failed"
		_ = s.videoTasks.Save(ctx, t)
		_ = s.videoTasks.Release(ctx, t.ID, t.LeaseToken)
		return nil, err
	}
	if err = s.videoTasks.Save(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *OpenAIGatewayService) reserveVideoHold(ctx context.Context, t *VideoTask, user *User) error {
	if t.HoldAmount <= 0 || s.billingCacheService == nil || t.Group.IsSubscriptionType() {
		return nil
	}
	cache, ok := s.billingCacheService.cache.(InflightBalanceReservationCache)
	if !ok {
		return fmt.Errorf("video balance reservation unavailable")
	}
	balance, err := s.billingCacheService.GetUserBalance(ctx, user.ID)
	if err != nil {
		return err
	}
	if balance < t.HoldAmount {
		return ErrInsufficientBalance
	}
	t.HoldID = "video:" + t.ID
	allowed, _, err := cache.ReserveInflightBalance(ctx, t.UserID, t.HoldID, t.HoldAmount, balance, videoHoldTTL(t))
	if err != nil {
		return err
	}
	if !allowed {
		return ErrInsufficientBalance
	}
	return nil
}
func (s *OpenAIGatewayService) releaseVideoHold(ctx context.Context, t *VideoTask) error {
	if t.HoldID == "" {
		return nil
	}
	cache, ok := s.billingCacheService.cache.(InflightBalanceReservationCache)
	if !ok {
		return fmt.Errorf("video balance reservation unavailable")
	}
	return cache.ReleaseInflightBalance(ctx, t.UserID, t.HoldID)
}
func (s *OpenAIGatewayService) AcceptVideoTask(ctx context.Context, t *VideoTask, accountID int64, r *OpenAIForwardResult) error {
	if t == nil {
		return nil
	}
	t.AccountID = accountID
	t.UpstreamID = r.ResponseID
	t.UpstreamModel = r.UpstreamModel
	t.Status = "pending"
	t.NextPollAt = time.Now().Add(15 * time.Second)
	t.Body = r.VideoTaskBody
	if len(t.Body) == 0 {
		t.Body, _ = json.Marshal(map[string]any{"id": strings.TrimPrefix(t.UpstreamID, "seedance:"), "status": "pending"})
	}
	if err := s.videoTasks.Save(ctx, t); err != nil {
		return err
	}
	return s.videoTasks.Release(ctx, t.ID, t.LeaseToken)
}

func acceptDurableVideoTask(c *gin.Context, r *OpenAIForwardResult) error {
	if value, ok := c.Get("video_task_accept"); ok {
		if accept, ok := value.(func(*OpenAIForwardResult) error); ok {
			return accept(r)
		}
	}
	return nil
}
func (s *OpenAIGatewayService) FailVideoSubmission(ctx context.Context, t *VideoTask) {
	if t == nil {
		return
	}
	t.Status = "submission_unknown"
	t.NextPollAt = nextUnknownVideoPoll(t)
	_ = s.videoTasks.Save(ctx, t)
	_ = s.videoTasks.Release(ctx, t.ID, t.LeaseToken)
}
func (s *OpenAIGatewayService) FindVideoTask(ctx context.Context, key *APIKey, id string) (*VideoTask, error) {
	if !s.VideoTasksEnabled() {
		return nil, nil
	}
	return s.videoTasks.Find(ctx, key.User.ID, key.ID, derefGroupID(key.GroupID), id)
}

// PollVideoTask is shared by the HTTP lookup and scheduler. A DB lease covers
// fetching, state transitions and settlement; old observations cannot overwrite success.
func (s *OpenAIGatewayService) PollVideoTask(parent context.Context, id string) (out *VideoTask, pollErr error) {
	ctx, cancel := context.WithTimeout(parent, 45*time.Second)
	defer cancel()
	t, err := s.videoTasks.Get(ctx, id)
	if err != nil || t == nil {
		return t, err
	}
	if t.Closed || t.Settled {
		return t, nil
	}
	// Merge bursts of client polls as well as worker/client overlap.
	if t.Result == nil && !videoTaskExpired(t, time.Now()) && !t.LastPolledAt.IsZero() && time.Since(t.LastPolledAt) < 3*time.Second {
		return t, nil
	}
	token := uuid.NewString()
	ok, err := s.videoTasks.Claim(ctx, id, token, 90*time.Second)
	if err != nil {
		return nil, err
	}
	if !ok {
		// A concurrent HTTP/background poll owns the task. Reuse its observation.
		for i := 0; i < 20; i++ {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(100 * time.Millisecond):
			}
			fresh, e := s.videoTasks.Get(ctx, id)
			if e != nil {
				return nil, e
			}
			if fresh != nil && (fresh.Closed || fresh.Settled || !fresh.NextPollAt.Equal(t.NextPollAt)) {
				return fresh, nil
			}
		}
		return t, nil
	}
	t, err = s.videoTasks.Get(ctx, id)
	if err != nil || t == nil {
		return t, err
	}
	t.LeaseToken = token
	defer func() {
		saveCtx, c := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer c()
		if saveErr := s.videoTasks.Save(saveCtx, t); saveErr != nil {
			slog.Error("video_task_save_failed", "task_id", id, "error", saveErr)
			pollErr = saveErr
		}
		_ = s.videoTasks.Release(saveCtx, id, token)
	}()
	t.NextPollAt = time.Now().Add(15 * time.Second)
	if time.Since(t.CreatedAt) > 20*time.Minute {
		t.NextPollAt = time.Now().Add(time.Minute)
	}
	if t.Result == nil && videoTaskExpired(t, time.Now()) {
		t.Status = "timeout"
		t.Closed = true
		t.Body, _ = json.Marshal(map[string]any{"id": strings.TrimPrefix(t.UpstreamID, "seedance:"), "status": "failed", "error": map[string]string{"code": "task_timeout", "message": "Video task timed out"}})
		if err = s.releaseVideoHold(ctx, t); err != nil {
			t.Closed = false
		}
		return t, err
	}
	// Keep the shared balance-inflight entry alive even while settlement retries.
	if t.HoldID != "" && s.billingCacheService != nil {
		if renewer, ok := s.billingCacheService.cache.(InflightBalanceReservationRenewer); ok {
			ttl := videoHoldTTL(t)
			_, _ = renewer.RenewInflightBalance(ctx, t.UserID, t.HoldID, ttl)
		}
	}
	if t.UpstreamID == "" {
		t.NextPollAt = nextUnknownVideoPoll(t)
		return t, nil
	}
	account, err := s.accountRepo.GetByID(ctx, t.AccountID)
	if err != nil {
		t.LastError = "upstream account unavailable"
		return t, err
	}
	if t.Result == nil {
		t.LastPolledAt = time.Now()
		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		c.Request, _ = http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost/v1/videos/"+strings.TrimPrefix(t.UpstreamID, "seedance:"), nil)
		c.Set("seedance_billing_mode", t.Mode)
		var result *OpenAIForwardResult
		if t.Provider == "seedance" {
			result, err = s.ForwardSeedance(ctx, c, account, SeedanceEndpointStatus, t.UpstreamID, nil)
		} else {
			result, err = s.ForwardGrokMedia(ctx, c, account, GrokMediaEndpointVideoStatus, t.UpstreamID, nil, "")
		}
		if err != nil {
			t.LastError = "upstream query failed"
			t.NextPollAt = time.Now().Add(time.Minute)
			return t, err
		}
		newBody := append(json.RawMessage(nil), recorder.Body.Bytes()...)
		status := strings.ToLower(gjson.GetBytes(newBody, "status").String())
		if videoTaskSucceeded(t.Provider, t.Status, t.Body) && !videoTaskSucceeded(t.Provider, status, newBody) {
			return t, nil
		}
		t.Body = newBody
		t.Status = status
		if videoTaskFailed(status) {
			t.Closed = true
			err = s.releaseVideoHold(ctx, t)
			if err != nil {
				t.Closed = false
			}
			return t, err
		}
		if !videoTaskSucceeded(t.Provider, status, t.Body) {
			return t, nil
		}
		// Only usage data is needed for retries, never upstream cookies/headers.
		result.ResponseHeaders = nil
		result.UpstreamHeaders = nil
		result.VideoTaskBody = nil
		result.Model = t.Model
		result.BillingModel = t.Model
		result.UpstreamModel = t.UpstreamModel
		result.ResponseID = t.UpstreamID
		result.RequestID = StableGrokVideoBillingRequestID(t.UpstreamID)
		result.BillingModeOverride = t.Mode
		result.Duration = time.Since(t.CreatedAt)
		if t.Provider == "grok" || result.VideoResolution == "" {
			result.VideoResolution = t.Resolution
		}
		if t.Mode == string(BillingModeVideo) {
			if p, ok := t.Prices[result.VideoResolution]; ok {
				t.UnitPrice = &p
			} else {
				t.LastError = "unpriced actual resolution"
				return t, nil
			}
		}
		if t.Mode == string(BillingModeVideo) {
			result.VideoCount = 1
			if t.Provider == "grok" && result.VideoDurationSeconds <= 0 {
				t.LastError = "missing video duration"
				return t, nil
			}
			if t.Provider == "seedance" && !result.VideoDurationActual {
				t.LastError = "missing actualDuration"
				return t, nil
			}
		}
		if t.Mode == string(BillingModeToken) && result.Usage.OutputTokens <= 0 {
			t.LastError = "missing completion tokens"
			return t, nil
		}
		t.Result = result
		// Persist the confirmed result before any money event; retry never needs upstream again.
		if err = s.videoTasks.Save(ctx, t); err != nil {
			return t, err
		}
	}
	user, err := s.userRepo.GetByID(ctx, t.UserID)
	if err != nil {
		return t, err
	}
	key := *t.Key
	key.User = user
	key.Group = t.Group
	if key.Group != nil {
		key.Group.VideoRateIndependent = true
		key.Group.VideoRateMultiplier = t.Multiplier
	}
	var keyUpdater APIKeyQuotaUpdater
	if s.videoTaskAPIKeys != nil {
		keyUpdater = s.videoTaskAPIKeys
	}
	if err = s.RecordUsage(ctx, &OpenAIRecordUsageInput{Result: t.Result, User: user, APIKey: &key, Account: account, Subscription: t.Subscription, APIKeyService: keyUpdater, QuotaPlatform: t.Group.Platform, RequestPayloadHash: HashUsageRequestPayload([]byte(t.ID)), InboundEndpoint: t.InboundEndpoint, UpstreamEndpoint: t.UpstreamEndpoint, VideoTask: t}); err != nil {
		t.LastError = "settlement failed"
		return t, err
	}
	t.Settled = true
	t.LastError = ""
	if err = s.releaseVideoHold(ctx, t); err != nil {
		t.Settled = false
	}
	return t, err
}
func videoTaskFailed(s string) bool {
	return s == "failed" || s == "cancelled" || s == "canceled" || s == "expired" || s == "error"
}
func videoTaskSucceeded(provider, status string, body []byte) bool {
	if provider == "grok" {
		return IsGrokVideoStatusBillable(body)
	}
	return status == "succeeded" || status == "completed"
}
func videoTaskCost(t *VideoTask, r *OpenAIForwardResult) (*CostBreakdown, error) {
	if t.UnitPrice == nil || !r.VideoDurationActual && t.Provider == "seedance" {
		return nil, fmt.Errorf("missing video price or actualDuration")
	}
	seconds := float64(r.VideoDurationSeconds)
	if r.VideoActualSeconds > 0 {
		seconds = r.VideoActualSeconds
	}
	if seconds <= 0 || math.IsNaN(seconds) || math.IsInf(seconds, 0) {
		return nil, fmt.Errorf("invalid video duration")
	}
	total := seconds * *t.UnitPrice
	return &CostBreakdown{TotalCost: total, ActualCost: total * t.Multiplier, BillingMode: string(BillingModeVideo)}, nil
}

func videoTaskExpired(t *VideoTask, now time.Time) bool {
	return !t.Deadline.IsZero() && !now.Before(t.Deadline)
}
func videoHoldTTL(t *VideoTask) time.Duration {
	if t.Deadline.IsZero() {
		return 25 * time.Hour
	}
	ttl := time.Until(t.Deadline) + time.Hour
	if ttl < time.Hour {
		return time.Hour
	}
	return ttl
}
func nextUnknownVideoPoll(t *VideoTask) time.Time {
	if t.Deadline.IsZero() {
		return time.Now().Add(time.Hour)
	}
	return t.Deadline
}
