//go:build unit

package handler

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	middleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type contentLookupVideoTasks struct {
	task *service.VideoTask
	err  error
}

func (s *contentLookupVideoTasks) Create(context.Context, *service.VideoTask, int) error { return nil }
func (s *contentLookupVideoTasks) Get(context.Context, string) (*service.VideoTask, error) {
	return s.task, s.err
}
func (s *contentLookupVideoTasks) Find(context.Context, int64, int64, int64, string) (*service.VideoTask, error) {
	return s.task, s.err
}
func (s *contentLookupVideoTasks) Due(context.Context, int) ([]string, error) { return nil, nil }
func (s *contentLookupVideoTasks) Claim(context.Context, string, string, time.Duration) (bool, error) {
	return true, nil
}
func (s *contentLookupVideoTasks) Save(context.Context, *service.VideoTask) error { return nil }
func (s *contentLookupVideoTasks) Release(context.Context, string, string) error { return nil }

func seedanceContentLookupContext() (*gin.Context, *httptest.ResponseRecorder) {
	c, w := grokMediaSlotContext(context.Background(), false)
	key, _ := middleware.GetAPIKeyFromContext(c)
	key.Group.Platform = service.PlatformOpenAI
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/videos/task-ark/content", nil)
	c.Params = gin.Params{{Key: "request_id", Value: "task-ark"}}
	return c, w
}

func seedanceContentLookupUpstream(t *testing.T) func(*http.Request, int64) (*http.Response, error) {
	t.Helper()
	return func(*http.Request, int64) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"id":"task-ark","status":"queued"}`)),
		}, nil
	}
}

func TestSeedanceContentLookupFallsBackToDurableTask(t *testing.T) {
	h, _, bindings, upstream := newGrokMediaSlotHandler(t, false, false, service.PlatformOpenAI)
	// The create-time binding is empty (as after its TTL); the durable task owns
	// the account and must keep the download reachable.
	bindings.key, bindings.owner = "", 0
	h.gatewayService.ConfigureVideoTasks(&contentLookupVideoTasks{task: &service.VideoTask{
		ID: "local-1", UserID: 10, APIKeyID: 20, GroupID: 24, AccountID: 1, UpstreamID: "seedance:task-ark",
	}})
	upstream.call = seedanceContentLookupUpstream(t)

	c, w := seedanceContentLookupContext()
	h.SeedanceVideos(c)

	require.Equal(t, http.StatusConflict, w.Code, w.Body.String())
	require.Equal(t, 1, upstream.calls)
}

func TestSeedanceContentLookupStoreFailureIsServiceUnavailable(t *testing.T) {
	h, _, bindings, upstream := newGrokMediaSlotHandler(t, false, false, service.PlatformOpenAI)
	bindings.key, bindings.owner = "", 0
	h.gatewayService.ConfigureVideoTasks(&contentLookupVideoTasks{err: errors.New("database offline")})
	upstream.call = seedanceContentLookupUpstream(t)

	c, w := seedanceContentLookupContext()
	h.SeedanceVideos(c)

	require.Equal(t, http.StatusServiceUnavailable, w.Code, w.Body.String())
	require.Zero(t, upstream.calls)
}
