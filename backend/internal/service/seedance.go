package service

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const (
	SeedanceEndpointCreate           GrokMediaEndpoint        = "seedance_create"
	SeedanceEndpointStatus           GrokMediaEndpoint        = "seedance_status"
	SeedanceEndpointDelete           GrokMediaEndpoint        = "seedance_delete"
	SeedanceEndpointContent          GrokMediaEndpoint        = "seedance_content"
	OpenAIEndpointCapabilitySeedance OpenAIEndpointCapability = "seedance"
)

// SeedanceContentProxyURLContextKey carries the gateway-side content URL that
// status responses should advertise instead of the upstream asset URL.
const SeedanceContentProxyURLContextKey = "seedance_content_proxy_url"

// Seedance polling policy used by the first, client-driven lifecycle. The
// gateway may return the cached task state when a client polls sooner than the
// minimum interval; no background goroutine is created per task.
const (
	SeedancePollMinimumInterval = 3 * time.Second
	SeedanceTaskSlowAfter       = 60 * time.Minute
	SeedanceTaskHardTimeout     = 24 * time.Hour
)

func SeedanceRecommendedPollInterval(age time.Duration) time.Duration {
	switch {
	case age < 2*time.Minute:
		return 30 * time.Second
	case age < 5*time.Minute:
		return 15 * time.Second
	case age < 20*time.Minute:
		return 10 * time.Second
	default:
		return time.Minute
	}
}

func (e GrokMediaEndpoint) IsSeedance() bool {
	return e == SeedanceEndpointCreate || e == SeedanceEndpointStatus || e == SeedanceEndpointDelete || e == SeedanceEndpointContent
}

// SeedanceTaskKey isolates ownership and billing keys from other video providers.
func SeedanceTaskKey(id string) string { return "seedance:" + strings.TrimSpace(id) }

func ParseSeedanceRequest(body []byte) (GrokMediaRequestInfo, error) {
	var info GrokMediaRequestInfo
	if !gjson.ValidBytes(body) || !gjson.ParseBytes(body).IsObject() {
		return info, fmt.Errorf("request body must be a JSON object")
	}
	model := gjson.GetBytes(body, "model")
	if model.Type != gjson.String || strings.TrimSpace(model.String()) == "" {
		return info, fmt.Errorf("model is required")
	}
	info.Model = strings.TrimSpace(model.String())
	content := gjson.GetBytes(body, "content")
	if !content.IsArray() || len(content.Array()) == 0 {
		prompt := strings.TrimSpace(gjson.GetBytes(body, "prompt").String())
		if prompt == "" {
			return info, fmt.Errorf("content or prompt is required")
		}
		info.Prompt = prompt
	} else {
		for _, item := range content.Array() {
			switch item.Get("type").String() {
			case "text":
				info.Prompt = strings.TrimSpace(info.Prompt + "\n" + item.Get("text").String())
			case "image_url":
				info.InputImageURLs = append(info.InputImageURLs, item.Get("image_url.url").String())
			}
		}
	}
	if d := gjson.GetBytes(body, "duration"); d.Exists() && d.Type == gjson.Number {
		info.DurationSeconds = int(d.Int())
	}
	if info.DurationSeconds <= 0 {
		if d := gjson.GetBytes(body, "seconds"); d.Exists() {
			if d.Type == gjson.Number {
				info.DurationSeconds = int(d.Int())
			}
			if d.Type == gjson.String {
				info.DurationSeconds, _ = strconv.Atoi(strings.TrimSpace(d.String()))
			}
		}
	}
	info.Resolution = NormalizeVideoBillingResolutionOrDefault(gjson.GetBytes(body, "resolution").String())
	return info, nil
}

func buildSeedanceURL(base string, endpoint GrokMediaEndpoint, taskID string) (string, error) {
	base = strings.TrimRight(base, "/")
	// Accept an origin, a proxy prefix, or the full Ark API base.
	if !strings.HasSuffix(base, "/api/v3") && !strings.HasSuffix(base, "/v3") {
		base += "/api/v3"
	}
	base += "/contents/generations/tasks"
	if endpoint != SeedanceEndpointCreate {
		if err := validateUpstreamPathSegment("Seedance task ID", taskID); err != nil || strings.TrimSpace(taskID) == "" {
			return "", fmt.Errorf("invalid Seedance task ID")
		}
		base += "/" + taskID
	}
	return base, nil
}

// ForwardSeedance preserves the Ark protocol, including multimodal content and
// future fields. Only model is rewritten using the account's configured mapping.
func (s *OpenAIGatewayService) ForwardSeedance(ctx context.Context, c *gin.Context, account *Account, endpoint GrokMediaEndpoint, taskID string, body []byte) (*OpenAIForwardResult, error) {
	if !account.SupportsOpenAIEndpointCapability(OpenAIEndpointCapabilitySeedance) || !endpoint.IsSeedance() {
		return nil, fmt.Errorf("seedance requires an OpenAI API key account with a custom base URL")
	}
	if endpoint == SeedanceEndpointContent {
		return s.forwardSeedanceVideoContent(ctx, c, account, taskID)
	}
	base, err := s.validateUpstreamBaseURL(account.GetCredential("base_url"))
	if err != nil {
		return nil, err
	}
	target, err := buildSeedanceURL(base, endpoint, strings.TrimPrefix(taskID, "seedance:"))
	if err != nil {
		return nil, err
	}
	model, upstreamModel := "", ""
	createInfo := GrokMediaRequestInfo{}
	method := http.MethodGet
	switch endpoint {
	case SeedanceEndpointCreate:
		info, parseErr := ParseSeedanceRequest(body)
		if parseErr != nil {
			return nil, parseErr
		}
		createInfo = info
		model = info.Model
		upstreamModel = account.GetMappedModel(model)
		body, err = sjson.SetBytes(body, "model", upstreamModel)
		if err != nil {
			return nil, err
		}
		if !gjson.GetBytes(body, "duration").Exists() && gjson.GetBytes(body, "seconds").Exists() {
			body, err = sjson.SetBytes(body, "duration", info.DurationSeconds)
			if err != nil {
				return nil, err
			}
		}
		if !gjson.GetBytes(body, "content").IsArray() {
			body, err = sjson.SetBytes(body, "content", []map[string]any{{"type": "text", "text": info.Prompt}})
			if err != nil {
				return nil, err
			}
		}
		method = http.MethodPost
	case SeedanceEndpointDelete:
		method = http.MethodDelete
	}
	token := strings.TrimSpace(account.GetCredential("api_key"))
	if token == "" {
		return nil, fmt.Errorf("seedance account missing api_key")
	}
	started := time.Now()
	req, err := http.NewRequestWithContext(ctx, method, target, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	account.ApplyHeaderOverrides(req.Header)
	proxy := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxy = account.Proxy.URL()
	}
	resp, err := s.httpUpstream.Do(req, proxy, account.ID, account.Concurrency)
	SetOpsLatencyMs(c, OpsUpstreamLatencyMsKey, time.Since(started).Milliseconds())
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	responseBody, err := ReadUpstreamResponseBody(resp.Body, s.cfg, c, openAITooLargeError)
	if err != nil {
		return nil, err
	}
	// Do not retry ambiguous asynchronous creates: the upstream may already have
	// accepted a billable job. Preserve native error codes and response bodies.
	if resp.StatusCode >= 300 {
		writeGrokMediaResponse(c, resp, responseBody, s.responseHeaderFilter)
		return nil, fmt.Errorf("seedance upstream status %d", resp.StatusCode)
	}
	result := &OpenAIForwardResult{Model: model, BillingModel: model, UpstreamModel: upstreamModel, Duration: time.Since(started), ResponseHeaders: resp.Header.Clone(), VideoResolution: createInfo.Resolution, VideoDurationSeconds: createInfo.DurationSeconds}
	result.VideoTaskBody = append([]byte(nil), responseBody...)
	// Billing mode is decided at task creation and carried through status polls.
	// Native Ark tasks use token billing; /videos explicitly uses per-second billing.
	result.BillingModeOverride = string(BillingModeToken)
	if mode, ok := c.Get("seedance_billing_mode"); ok && mode == string(BillingModeVideo) {
		result.BillingModeOverride = string(BillingModeVideo)
	}
	if endpoint == SeedanceEndpointCreate {
		id := strings.TrimSpace(gjson.GetBytes(responseBody, "id").String())
		if id == "" {
			return nil, fmt.Errorf("seedance create response missing task ID")
		}
		result.ResponseID = SeedanceTaskKey(id)
	}
	if endpoint == SeedanceEndpointStatus {
		result.ResponseID = taskID
		result.Status = strings.ToLower(strings.TrimSpace(gjson.GetBytes(responseBody, "status").String()))
		result.StatusError = strings.TrimSpace(gjson.GetBytes(responseBody, "error.message").String())
		if result.StatusError == "" {
			result.StatusError = strings.TrimSpace(gjson.GetBytes(responseBody, "error").String())
		}
		result.UpstreamModel = gjson.GetBytes(responseBody, "model").String()
		if result.Status == "failed" || result.Status == "cancelled" || result.Status == "error" {
			writeGrokMediaResponse(c, resp, responseBody, s.responseHeaderFilter)
			return result, nil
		}
		if result.Status == "succeeded" || result.Status == "completed" {
			result.Usage.OutputTokens = max(0, int(gjson.GetBytes(responseBody, "usage.completion_tokens").Int()))
			if result.BillingModeOverride == string(BillingModeVideo) {
				result.VideoCount = 1
			}
			// Seedance's billing contract uses the upstream-confirmed actualDuration.
			// Do not substitute Grok's video.duration or the requested duration here;
			// the creation snapshot is merged by the deferred billing path only when
			// the provider omits actualDuration.
			if actual := gjson.GetBytes(responseBody, "actualDuration"); actual.Exists() {
				result.VideoActualSeconds = actual.Float()
				if actual.Type == gjson.Number {
					result.VideoDurationSeconds = int(actual.Int())
					result.VideoDurationActual = result.VideoDurationSeconds > 0
				} else if actual.Type == gjson.String {
					if parsed, parseErr := strconv.Atoi(strings.TrimSpace(actual.String())); parseErr == nil {
						result.VideoDurationSeconds = parsed
						result.VideoDurationActual = parsed > 0
					}
				}
			}
			if result.VideoActualSeconds > 0 {
				result.VideoDurationActual = true
			}
			if resolution := strings.TrimSpace(gjson.GetBytes(responseBody, "resolution").String()); resolution != "" {
				result.VideoResolution = NormalizeVideoBillingResolutionOrDefault(resolution)
			}
		}
	}
	if endpoint == SeedanceEndpointCreate {
		if err := acceptDurableVideoTask(c, result); err != nil {
			return nil, err
		}
	}
	if endpoint == SeedanceEndpointStatus {
		if proxyURL, ok := c.Get(SeedanceContentProxyURLContextKey); ok {
			if value, ok := proxyURL.(string); ok {
				responseBody = RewriteSeedanceVideoURLs(responseBody, value)
			}
		}
	}
	writeGrokMediaResponse(c, resp, responseBody, s.responseHeaderFilter)
	return result, nil
}

// forwardSeedanceVideoContent relays the finished asset through the gateway. The
// upstream asset is fetched with the platform account's key, so clients never
// need (and never receive) a URL that only the task creator can open.
func (s *OpenAIGatewayService) forwardSeedanceVideoContent(ctx context.Context, c *gin.Context, account *Account, taskID string) (*OpenAIForwardResult, error) {
	rawID := strings.TrimPrefix(strings.TrimSpace(taskID), "seedance:")
	base, err := s.validateUpstreamBaseURL(account.GetCredential("base_url"))
	if err != nil {
		return nil, err
	}
	statusURL, err := buildSeedanceURL(base, SeedanceEndpointStatus, rawID)
	if err != nil {
		return nil, err
	}
	token := strings.TrimSpace(account.GetCredential("api_key"))
	if token == "" {
		return nil, fmt.Errorf("seedance account missing api_key")
	}
	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	started := time.Now()
	upstreamCtx, releaseUpstreamCtx := detachUpstreamContext(ctx)
	defer releaseUpstreamCtx()

	statusReq, err := http.NewRequestWithContext(upstreamCtx, http.MethodGet, statusURL, nil)
	if err != nil {
		return nil, err
	}
	statusReq.Header.Set("Authorization", "Bearer "+token)
	statusReq.Header.Set("Accept", "application/json")
	account.ApplyHeaderOverrides(statusReq.Header)
	statusResp, err := s.httpUpstream.Do(statusReq, proxyURL, account.ID, account.Concurrency)
	SetOpsLatencyMs(c, OpsUpstreamLatencyMsKey, time.Since(started).Milliseconds())
	if err != nil {
		return nil, err
	}
	statusBody, err := ReadUpstreamResponseBody(statusResp.Body, s.cfg, c, openAITooLargeError)
	_ = statusResp.Body.Close()
	if err != nil {
		return nil, err
	}
	if statusResp.StatusCode >= 300 {
		writeGrokMediaResponse(c, statusResp, statusBody, s.responseHeaderFilter)
		return nil, fmt.Errorf("seedance upstream status %d", statusResp.StatusCode)
	}
	contentURL := seedanceVideoContentURL(statusBody)
	if contentURL == "" {
		c.JSON(http.StatusConflict, gin.H{"error": gin.H{"type": "video_not_ready", "message": "Video is not ready yet"}})
		return nil, fmt.Errorf("seedance video content is not ready")
	}
	target, signed, err := resolveSeedanceContentURL(contentURL, base)
	if err != nil {
		return nil, err
	}

	contentReq, err := http.NewRequestWithContext(WithHTTPUpstreamRedirectsDisabled(upstreamCtx), http.MethodGet, target.String(), nil)
	if err != nil {
		return nil, err
	}
	contentReq.Header.Set("Accept", "*/*")
	if c != nil {
		if rangeHeader := strings.TrimSpace(c.GetHeader("Range")); rangeHeader != "" {
			contentReq.Header.Set("Range", rangeHeader)
		}
	}
	if !signed {
		contentReq.Header.Set("Authorization", "Bearer "+token)
		account.ApplyHeaderOverrides(contentReq.Header)
	}
	contentResp, err := s.httpUpstream.Do(contentReq, proxyURL, account.ID, account.Concurrency)
	SetOpsLatencyMs(c, OpsUpstreamLatencyMsKey, time.Since(started).Milliseconds())
	if err != nil {
		return nil, err
	}
	defer func() { _ = contentResp.Body.Close() }()
	if contentResp.StatusCode >= 300 && contentResp.StatusCode != http.StatusRequestedRangeNotSatisfiable {
		body, readErr := ReadUpstreamResponseBody(contentResp.Body, s.cfg, c, openAITooLargeError)
		if readErr != nil {
			return nil, readErr
		}
		writeGrokMediaResponse(c, contentResp, body, s.responseHeaderFilter)
		return nil, fmt.Errorf("seedance content upstream status %d", contentResp.StatusCode)
	}
	if err := writeGrokMediaContentResponse(c, contentResp); err != nil {
		return nil, err
	}
	return &OpenAIForwardResult{
		RequestID:       rawID,
		ResponseID:      SeedanceTaskKey(rawID),
		UpstreamHeaders: contentResp.Header,
		ResponseHeaders: contentResp.Header.Clone(),
		Duration:        time.Since(started),
	}, nil
}

func seedanceVideoContentURL(body []byte) string {
	for _, path := range []string{"content.video_url", "video_url", "result_url", "video.url"} {
		if value := strings.TrimSpace(gjson.GetBytes(body, path).String()); value != "" {
			return value
		}
	}
	return ""
}

// resolveSeedanceContentURL accepts same-origin asset URLs (fetched with the
// account key) and public HTTPS signed URLs (fetched without credentials).
func resolveSeedanceContentURL(rawURL, base string) (*url.URL, bool, error) {
	baseURL, err := url.Parse(strings.TrimSpace(base))
	if err != nil || baseURL.Host == "" {
		return nil, false, fmt.Errorf("invalid seedance base URL")
	}
	ref, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || (ref.Host == "" && ref.Path == "") || ref.User != nil {
		return nil, false, fmt.Errorf("invalid seedance content URL")
	}
	target := ref
	if ref.Host == "" {
		target = baseURL.ResolveReference(ref)
	}
	if target.Scheme != "http" && target.Scheme != "https" {
		return nil, false, fmt.Errorf("invalid seedance content URL")
	}
	if strings.EqualFold(target.Scheme, baseURL.Scheme) && strings.EqualFold(target.Host, baseURL.Host) {
		return target, false, nil
	}
	if target.Scheme != "https" || isBlockedHostname(target.Hostname()) {
		return nil, false, fmt.Errorf("unsupported seedance content URL")
	}
	if ip := net.ParseIP(target.Hostname()); ip != nil && isPrivateIP(ip) {
		return nil, false, fmt.Errorf("unsupported seedance content URL")
	}
	return target, true, nil
}

// RewriteSeedanceVideoURLs replaces upstream asset URLs with the gateway
// content endpoint so clients can download with their own API key.
func RewriteSeedanceVideoURLs(body []byte, proxyURL string) []byte {
	if len(body) == 0 || strings.TrimSpace(proxyURL) == "" || !gjson.ValidBytes(body) {
		return body
	}
	out := body
	for _, path := range []string{"video_url", "result_url", "video.url", "content.video_url"} {
		if value := strings.TrimSpace(gjson.GetBytes(out, path).String()); value == "" {
			continue
		}
		next, err := sjson.SetBytes(out, path, proxyURL)
		if err != nil {
			return out
		}
		out = next
	}
	return out
}

// SeedanceContentProxyURL builds the gateway URL clients should use for the
// finished asset. Ark's native surface needs an absolute URL, so use the
// request host (never X-Forwarded-Host) and fall back to a relative path.
func SeedanceContentProxyURL(c *gin.Context, taskID string) string {
	if c == nil || c.Request == nil {
		return ""
	}
	taskID = strings.TrimPrefix(strings.TrimSpace(taskID), "seedance:")
	if taskID == "" {
		return ""
	}
	path := "/v1/videos/" + url.PathEscape(taskID) + "/content"
	host := strings.TrimSpace(c.Request.Host)
	if host == "" {
		return path
	}
	scheme := "http"
	if c.Request.TLS != nil || strings.EqualFold(strings.TrimSpace(c.GetHeader("X-Forwarded-Proto")), "https") {
		scheme = "https"
	}
	return scheme + "://" + host + path
}
