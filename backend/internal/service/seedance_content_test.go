//go:build unit

package service

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func seedanceContentTestConfig() *config.Config {
	cfg := &config.Config{}
	cfg.Security.URLAllowlist.AllowInsecureHTTP = true
	return cfg
}

func seedanceContentTestAccount() *Account {
	return &Account{
		ID:       15,
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"api_key":             "sk-platform",
			"base_url":            "http://127.0.0.1:18082/ark",
			"openai_capabilities": []any{"seedance"},
		},
	}
}

func TestForwardSeedanceContentFetchesTaskThenStreamsWithAccountKey(t *testing.T) {
	upstream := &grokMediaContentUpstreamStub{
		responses: []*http.Response{
			grokMediaContentStatusResponse(`{"id":"task-1","status":"succeeded","actualDuration":1,"content":{"video_url":"http://127.0.0.1:18082/v1/videos/task-1/content"},"video_url":"http://127.0.0.1:18082/v1/videos/task-1/content"}`),
			{
				StatusCode: http.StatusPartialContent,
				Header: http.Header{
					"Content-Type":   []string{"video/mp4"},
					"Content-Length": []string{"13"},
					"Content-Range":  []string{"bytes 0-12/100"},
					"Accept-Ranges":  []string{"bytes"},
				},
				Body: io.NopCloser(strings.NewReader("video-payload")),
			},
		},
	}
	svc := &OpenAIGatewayService{cfg: seedanceContentTestConfig(), httpUpstream: upstream}
	c, recorder := grokMediaContentTestContext(http.MethodGet, "https://api.example/v1/videos/task-1/content", map[string]string{
		"Range": "bytes=0-12",
	})

	result, err := svc.ForwardSeedance(
		context.Background(), c, seedanceContentTestAccount(),
		SeedanceEndpointContent, "seedance:task-1", nil,
	)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, http.StatusPartialContent, recorder.Code)
	require.Equal(t, "video-payload", recorder.Body.String())
	require.Len(t, upstream.requests, 2)
	require.Equal(t, "http://127.0.0.1:18082/ark/api/v3/contents/generations/tasks/task-1", upstream.requests[0].URL.String())
	require.Equal(t, "Bearer sk-platform", upstream.requests[0].Header.Get("Authorization"))
	require.Equal(t, "http://127.0.0.1:18082/v1/videos/task-1/content", upstream.requests[1].URL.String())
	require.Equal(t, "Bearer sk-platform", upstream.requests[1].Header.Get("Authorization"))
	require.Equal(t, "bytes=0-12", upstream.requests[1].Header.Get("Range"))
	require.Equal(t, "video/mp4", recorder.Header().Get("Content-Type"))
	require.True(t, IsResponseCommitted(c))
}

func TestForwardSeedanceContentDoesNotLeakAccountKeyToSignedHost(t *testing.T) {
	upstream := &grokMediaContentUpstreamStub{
		responses: []*http.Response{
			grokMediaContentStatusResponse(`{"id":"task-1","status":"succeeded","content":{"video_url":"https://cdn.example/asset.mp4?sign=1"}}`),
			{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"video/mp4"}},
				Body:       io.NopCloser(strings.NewReader("signed")),
			},
		},
	}
	svc := &OpenAIGatewayService{cfg: seedanceContentTestConfig(), httpUpstream: upstream}
	account := seedanceContentTestAccount()
	account.Credentials["base_url"] = "https://ark.example/api/v3"
	c, recorder := grokMediaContentTestContext(http.MethodGet, "https://api.example/v1/videos/task-1/content", nil)

	_, err := svc.ForwardSeedance(
		context.Background(), c, account,
		SeedanceEndpointContent, "seedance:task-1", nil,
	)

	require.NoError(t, err)
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "signed", recorder.Body.String())
	require.Len(t, upstream.requests, 2)
	require.Equal(t, "https://cdn.example/asset.mp4?sign=1", upstream.requests[1].URL.String())
	require.Empty(t, upstream.requests[1].Header.Get("Authorization"))
}

func TestForwardSeedanceContentRejectsPrivateSignedHost(t *testing.T) {
	upstream := &grokMediaContentUpstreamStub{
		responses: []*http.Response{
			grokMediaContentStatusResponse(`{"id":"task-1","status":"succeeded","content":{"video_url":"http://169.254.169.254/latest/meta-data"}}`),
		},
	}
	svc := &OpenAIGatewayService{cfg: seedanceContentTestConfig(), httpUpstream: upstream}
	c, _ := grokMediaContentTestContext(http.MethodGet, "https://api.example/v1/videos/task-1/content", nil)

	_, err := svc.ForwardSeedance(
		context.Background(), c, seedanceContentTestAccount(),
		SeedanceEndpointContent, "seedance:task-1", nil,
	)

	require.ErrorContains(t, err, "unsupported seedance content URL")
	require.Len(t, upstream.requests, 1)
}

func TestForwardSeedanceStatusRewritesAssetURLsToGateway(t *testing.T) {
	statusBody := `{"id":"task-1","status":"succeeded","actualDuration":1,"video_url":"http://127.0.0.1:18082/v1/videos/task-1/content","result_url":"http://127.0.0.1:18082/v1/videos/task-1/content","video":{"url":"http://127.0.0.1:18082/v1/videos/task-1/content","duration":1},"content":{"video_url":"http://127.0.0.1:18082/v1/videos/task-1/content","duration":1},"usage":{"completion_tokens":42}}`
	upstream := &grokMediaContentUpstreamStub{response: grokMediaContentStatusResponse(statusBody)}
	svc := &OpenAIGatewayService{cfg: seedanceContentTestConfig(), httpUpstream: upstream}
	c, recorder := grokMediaContentTestContext(http.MethodGet, "https://api.example/v1/videos/task-1", nil)
	c.Set(SeedanceContentProxyURLContextKey, "https://api.example/v1/videos/task-1/content")

	_, err := svc.ForwardSeedance(
		context.Background(), c, seedanceContentTestAccount(),
		SeedanceEndpointStatus, "seedance:task-1", nil,
	)

	require.NoError(t, err)
	require.Equal(t, http.StatusOK, recorder.Code)
	for _, path := range []string{"video_url", "result_url", "video.url", "content.video_url"} {
		require.Equal(t, "https://api.example/v1/videos/task-1/content", gjson.GetBytes(recorder.Body.Bytes(), path).String(), path)
	}
	require.Equal(t, "42", gjson.GetBytes(recorder.Body.Bytes(), "usage.completion_tokens").String())
}

func TestRewriteSeedanceVideoURLsLeavesOtherFieldsIntact(t *testing.T) {
	body := []byte(`{"id":"task-1","status":"succeeded","video_url":"http://upstream/content","result_url":"","content":{"video_url":"http://upstream/content","duration":5},"counter":9007199254740993}`)

	rewritten := RewriteSeedanceVideoURLs(body, "https://api.example/v1/videos/task-1/content")

	require.Equal(t, "https://api.example/v1/videos/task-1/content", gjson.GetBytes(rewritten, "video_url").String())
	require.Equal(t, "", gjson.GetBytes(rewritten, "result_url").String())
	require.Equal(t, "https://api.example/v1/videos/task-1/content", gjson.GetBytes(rewritten, "content.video_url").String())
	require.Equal(t, "5", gjson.GetBytes(rewritten, "content.duration").String())
	require.Equal(t, "9007199254740993", gjson.GetBytes(rewritten, "counter").String())
}

func TestSeedanceContentProxyURLUsesRequestHost(t *testing.T) {
	c, _ := grokMediaContentTestContext(http.MethodGet, "https://api.example/api/v3/contents/generations/tasks/task-1", nil)

	require.Equal(t, "https://api.example/v1/videos/task-1/content", SeedanceContentProxyURL(c, "seedance:task-1"))
}
