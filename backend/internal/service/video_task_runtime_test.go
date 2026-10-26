//go:build unit

package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/stretchr/testify/require"
)

type memoryVideoTasks struct {
	mu    sync.Mutex
	task  *VideoTask
	token string
	until time.Time
	limit int
}

func (m *memoryVideoTasks) Create(_ context.Context, t *VideoTask, limit int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.task = copyVideoTask(t)
	m.limit = limit
	return nil
}
func copyVideoTask(t *VideoTask) *VideoTask {
	b, _ := json.Marshal(t)
	var out VideoTask
	_ = json.Unmarshal(b, &out)
	return &out
}
func (m *memoryVideoTasks) Get(context.Context, string) (*VideoTask, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return copyVideoTask(m.task), nil
}
func (m *memoryVideoTasks) Find(ctx context.Context, _, _, _ int64, _ string) (*VideoTask, error) {
	return m.Get(ctx, "")
}
func (m *memoryVideoTasks) Due(context.Context, int) ([]string, error) {
	return []string{m.task.ID}, nil
}
func (m *memoryVideoTasks) Claim(_ context.Context, _, token string, ttl time.Duration) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.token != "" && time.Now().Before(m.until) {
		return false, nil
	}
	if m.task.Closed || m.task.Settled {
		return false, nil
	}
	m.token = token
	m.until = time.Now().Add(ttl)
	return true, nil
}
func (m *memoryVideoTasks) Save(_ context.Context, t *VideoTask) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if t.LeaseToken != m.token {
		return errors.New("lost lease")
	}
	m.task = copyVideoTask(t)
	return nil
}
func (m *memoryVideoTasks) Release(_ context.Context, _, token string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if token == m.token {
		m.token = ""
	}
	return nil
}

type videoTaskUsers struct{ UserRepository }

func (videoTaskUsers) GetByID(context.Context, int64) (*User, error) { return &User{ID: 1}, nil }

func videoTaskTestService(t *testing.T) (*OpenAIGatewayService, *memoryVideoTasks, *openAIRecordUsageBillingRepoStub) {
	t.Helper()
	logs := &openAIRecordUsageLogRepoStub{inserted: true}
	s := newOpenAIRecordUsageServiceForTest(logs, videoTaskUsers{}, nil, nil)
	account := seedanceTestAccount()
	s.accountRepo = &openAIRecordUsageAccountRepoStub{account: account}
	b := &openAIRecordUsageBillingRepoStub{}
	s.usageBillingRepo = b
	price := 0.1
	group := &Group{ID: 3, Platform: PlatformOpenAI, RateMultiplier: 1, VideoRateMultiplier: 1}
	gid := int64(3)
	task := &VideoTask{ID: "local-1", UserID: 1, APIKeyID: 2, GroupID: 3, AccountID: account.ID, Provider: "seedance", Mode: string(BillingModeVideo), Model: "custom-video", Resolution: "720p", UpstreamID: "seedance:task-1", UnitPrice: &price, Prices: map[string]float64{"720p": price}, Multiplier: 2, Group: group, Key: &APIKey{ID: 2, UserID: 1, GroupID: &gid}, CreatedAt: time.Now(), Deadline: time.Now().Add(time.Hour)}
	store := &memoryVideoTasks{task: task}
	s.ConfigureVideoTasks(store)
	return s, store, b
}

func TestVideoTaskBackgroundSettlementWithoutClient(t *testing.T) {
	s, store, b := videoTaskTestService(t)
	up := &grokMediaContentUpstreamStub{response: grokMediaContentStatusResponse(`{"id":"task-1","status":"succeeded","actualDuration":16.5,"resolution":"720p","usage":{"completion_tokens":999}}`)}
	s.httpUpstream = up
	result, err := s.PollVideoTask(context.Background(), store.task.ID)
	require.NoError(t, err)
	require.True(t, result.Settled)
	require.Equal(t, 1, b.calls)
	require.InDelta(t, 3.3, b.lastCmd.BalanceCost, 1e-8)
	_, err = s.PollVideoTask(context.Background(), store.task.ID)
	require.NoError(t, err)
	require.Len(t, up.requests, 1)
	require.Equal(t, 1, b.calls)
}
func TestVideoTaskSettlementRetryDoesNotQueryUpstream(t *testing.T) {
	s, store, b := videoTaskTestService(t)
	up := &grokMediaContentUpstreamStub{response: grokMediaContentStatusResponse(`{"id":"task-1","status":"succeeded","actualDuration":5,"resolution":"720p"}`)}
	s.httpUpstream = up
	b.err = errors.New("database offline")
	result, err := s.PollVideoTask(context.Background(), store.task.ID)
	require.Error(t, err)
	require.NotNil(t, result.Result)
	require.False(t, result.Settled)
	// Simulate restart: the next invocation only has durable result state.
	b.err = nil
	_, err = s.PollVideoTask(context.Background(), store.task.ID)
	require.NoError(t, err)
	require.Len(t, up.requests, 1)
	deadline := time.Now().Add(time.Second)
	for b.calls < 2 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	require.Equal(t, 2, b.calls)
}
func TestVideoTaskTimeoutStopsUpstreamQueries(t *testing.T) {
	s, store, b := videoTaskTestService(t)
	store.task.Deadline = time.Now().Add(-time.Second)
	cache := newMemInflightCache(10)
	s.billingCacheService = &BillingCacheService{cache: cache}
	store.task.HoldID = "video:local-1"
	cache.res[store.task.HoldID] = 3
	result, err := s.PollVideoTask(context.Background(), store.task.ID)
	require.NoError(t, err)
	require.True(t, result.Closed)
	require.Equal(t, "timeout", result.Status)
	require.Zero(t, b.calls)
	require.EqualValues(t, 1, cache.releaseCt.Load())
	_, err = s.PollVideoTask(context.Background(), store.task.ID)
	require.NoError(t, err)
}

type blockingVideoUpstream struct {
	started chan struct{}
	finish  chan struct{}
	calls   atomic.Int32
}

func (u *blockingVideoUpstream) Do(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	u.calls.Add(1)
	close(u.started)
	select {
	case <-u.finish:
	case <-req.Context().Done():
		return nil, req.Context().Err()
	}
	return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"id":"task-1","status":"succeeded","actualDuration":5,"resolution":"720p"}`))}, nil
}
func (u *blockingVideoUpstream) DoWithTLS(req *http.Request, p string, id int64, n int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	return u.Do(req, p, id, n)
}
func TestVideoTaskConcurrentClientAndWorkerShareQuery(t *testing.T) {
	s, store, b := videoTaskTestService(t)
	up := &blockingVideoUpstream{started: make(chan struct{}), finish: make(chan struct{})}
	s.httpUpstream = up
	first := make(chan error, 1)
	go func() { _, e := s.PollVideoTask(context.Background(), store.task.ID); first <- e }()
	<-up.started
	second := make(chan *VideoTask, 1)
	go func() { v, _ := s.PollVideoTask(context.Background(), store.task.ID); second <- v }()
	time.Sleep(150 * time.Millisecond)
	close(up.finish)
	require.NoError(t, <-first)
	result := <-second
	require.True(t, result.Settled)
	require.EqualValues(t, 1, up.calls.Load())
	require.Equal(t, 1, b.calls)
}
func TestVideoTaskMissingActualDurationNeverBills(t *testing.T) {
	s, store, b := videoTaskTestService(t)
	s.httpUpstream = &grokMediaContentUpstreamStub{response: grokMediaContentStatusResponse(`{"id":"task-1","status":"succeeded","duration":15,"resolution":"720p"}`)}
	result, err := s.PollVideoTask(context.Background(), store.task.ID)
	require.NoError(t, err)
	require.Nil(t, result.Result)
	require.False(t, result.Settled)
	require.Zero(t, b.calls)
	require.Equal(t, "missing actualDuration", result.LastError)
}

func TestVideoTaskNeverTimeoutStillPollsAndSettles(t *testing.T) {
	s, store, b := videoTaskTestService(t)
	store.task.CreatedAt = time.Now().Add(-30 * 24 * time.Hour)
	store.task.Deadline = time.Time{}
	s.httpUpstream = &grokMediaContentUpstreamStub{response: grokMediaContentStatusResponse(`{"id":"task-1","status":"succeeded","actualDuration":10,"resolution":"720p"}`)}
	result, err := s.PollVideoTask(context.Background(), store.task.ID)
	require.NoError(t, err)
	require.True(t, result.Settled)
	require.Equal(t, 1, b.calls)
}

func TestVideoTaskUnknownSubmissionWithoutTimeoutRenewsLater(t *testing.T) {
	s, store, _ := videoTaskTestService(t)
	store.task.UpstreamID = ""
	store.task.Deadline = time.Time{}
	result, err := s.PollVideoTask(context.Background(), store.task.ID)
	require.NoError(t, err)
	require.False(t, result.Closed)
	require.True(t, result.NextPollAt.After(time.Now().Add(59*time.Minute)))
	require.Greater(t, videoHoldTTL(result), 24*time.Hour)
}

type videoLimitUsers struct {
	UserRepository
	limit int
}

func (u videoLimitUsers) GetByID(context.Context, int64) (*User, error) {
	return &User{ID: 1, VideoMaxPendingTasks: u.limit}, nil
}
func TestVideoTaskAdmissionPrefersFreshUserLimit(t *testing.T) {
	for _, limit := range []int{0, 2, 12} {
		s, store, _ := videoTaskTestService(t)
		s.userRepo = videoLimitUsers{limit: limit}
		gid := int64(3)
		key := &APIKey{ID: 2, User: &User{ID: 1, VideoMaxPendingTasks: 99}, GroupID: &gid, Group: &Group{ID: gid, VideoMaxPendingTasks: 5, VideoTaskTimeoutHours: 24}}
		task, err := s.PrepareVideoTask(context.Background(), key, nil, SeedanceEndpointCreate, "seedance-2.0", "720p", "")
		require.NoError(t, err)
		want := limit
		if want == 0 {
			want = 5
		}
		require.Equal(t, want, store.limit)
		require.Equal(t, limit > 0, task.GlobalPendingLimit)
	}
}
