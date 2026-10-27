//go:build unit

package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func videoLookupTestKey(groupID int64) *APIKey {
	return &APIKey{ID: 2, UserID: 1, GroupID: &groupID, User: &User{ID: 1}}
}

func videoLookupTestTask(accountID int64) *VideoTask {
	return &VideoTask{ID: "local-1", UserID: 1, APIKeyID: 2, GroupID: 3, AccountID: accountID, UpstreamID: "seedance:task-1"}
}

func videoLookupTestService(t *testing.T, cache GatewayCache, tasks VideoTaskRepository) *OpenAIGatewayService {
	t.Helper()
	svc := &OpenAIGatewayService{cache: cache}
	if tasks != nil {
		svc.ConfigureVideoTasks(tasks)
	}
	return svc
}

type brokenVideoTasks struct{ memoryVideoTasks }

func (b *brokenVideoTasks) Find(context.Context, int64, int64, int64, string) (*VideoTask, error) {
	return nil, errors.New("database offline")
}

func TestResolveVideoLookupAccountUsesCreateTimeBinding(t *testing.T) {
	cache := &mockGatewayCacheForPlatform{sessionBindings: map[string]int64{}}
	svc := videoLookupTestService(t, cache, &memoryVideoTasks{task: videoLookupTestTask(7)})
	gid := int64(3)
	require.NoError(t, svc.BindGrokMediaVideoRequestAccount(context.Background(), &gid, "seedance:task-1", 1, 2, 9))

	accountID, err := svc.ResolveVideoLookupAccount(context.Background(), videoLookupTestKey(gid), 1, "seedance:task-1")

	require.NoError(t, err)
	require.Equal(t, int64(9), accountID)
}

func TestResolveVideoLookupAccountFallsBackToDurableTaskWhenBindingExpired(t *testing.T) {
	cache := &mockGatewayCacheForPlatform{sessionBindings: map[string]int64{}}
	tasks := &memoryVideoTasks{task: videoLookupTestTask(7)}
	svc := videoLookupTestService(t, cache, tasks)

	accountID, err := svc.ResolveVideoLookupAccount(context.Background(), videoLookupTestKey(3), 1, "seedance:task-1")

	require.NoError(t, err)
	require.Equal(t, int64(7), accountID)
}

func TestResolveVideoLookupAccountIgnoresTaskWithoutUpstreamID(t *testing.T) {
	cache := &mockGatewayCacheForPlatform{sessionBindings: map[string]int64{}}
	task := videoLookupTestTask(7)
	task.UpstreamID = ""
	svc := videoLookupTestService(t, cache, &memoryVideoTasks{task: task})

	accountID, err := svc.ResolveVideoLookupAccount(context.Background(), videoLookupTestKey(3), 1, "seedance:task-1")

	require.Error(t, err)
	require.Zero(t, accountID)
}

func TestResolveVideoLookupAccountWithoutDurableTasksKeepsBindingMiss(t *testing.T) {
	cache := &mockGatewayCacheForPlatform{sessionBindings: map[string]int64{}}
	svc := videoLookupTestService(t, cache, nil)

	accountID, err := svc.ResolveVideoLookupAccount(context.Background(), videoLookupTestKey(3), 1, "seedance:task-1")

	require.Error(t, err)
	require.Zero(t, accountID)
}

func TestResolveVideoLookupAccountReportsTaskStoreFailure(t *testing.T) {
	cache := &mockGatewayCacheForPlatform{sessionBindings: map[string]int64{}}
	svc := videoLookupTestService(t, cache, &brokenVideoTasks{})

	accountID, err := svc.ResolveVideoLookupAccount(context.Background(), videoLookupTestKey(3), 1, "seedance:task-1")

	require.ErrorIs(t, err, ErrVideoTaskStoreUnavailable)
	require.Zero(t, accountID)
}
