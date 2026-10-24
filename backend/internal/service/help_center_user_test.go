//go:build unit

package service

import (
	"context"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestDismissHelpCenterKeyPromptUpdatesOnlyPreference(t *testing.T) {
	repo := &mockUserRepo{getByIDUser: &User{ID: 42, Balance: 25}}
	repo.updateFn = func(_ context.Context, user *User) error {
		repo.getByIDUser.HelpCenterKeyPromptDismissed = user.HelpCenterKeyPromptDismissed
		return nil
	}
	svc := NewUserService(repo, nil, nil, nil)
	require.NoError(t, svc.DismissHelpCenterKeyPrompt(context.Background(), 42))
	require.Equal(t, []UserUpdateFields{{HelpCenterKeyPromptDismissed: true}}, repo.updateFields)
	require.True(t, repo.getByIDUser.HelpCenterKeyPromptDismissed)
	require.NoError(t, svc.DismissHelpCenterKeyPrompt(context.Background(), 42))
	require.Equal(t, 1, repo.updateCalls)
}
