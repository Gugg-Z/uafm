package service

import (
	"context"
	"fmt"
)

func (s *UserService) DismissHelpCenterKeyPrompt(ctx context.Context, userID int64) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}
	if user.HelpCenterKeyPromptDismissed {
		return nil
	}
	user.HelpCenterKeyPromptDismissed = true
	if err := s.userRepo.Update(ctx, user, UserUpdateFields{HelpCenterKeyPromptDismissed: true}); err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	return nil
}
