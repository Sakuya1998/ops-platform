package service

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/Sakuya1998/ops-platform/services/auth-svc/internal/model"
	"golang.org/x/crypto/bcrypt"
)

const (
	minPasswordLength    = 8
	passwordHistoryLimit = 5
	passwordMaxAge       = 90 * 24 * time.Hour
)

func (s *AuthService) markPasswordExpiredIfNeeded(user *model.User) error {
	if user.Source != "local" || user.MustChangePassword || user.PasswordChangedAt == nil {
		return nil
	}
	if time.Since(*user.PasswordChangedAt) <= passwordMaxAge {
		return nil
	}
	user.MustChangePassword = true
	return s.userRepo.Update(user)
}

func (s *AuthService) validatePasswordReuse(user *model.User, newPassword string) error {
	if user.PasswordHash != "" && bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(newPassword)) == nil {
		return fmt.Errorf("password cannot reuse recent passwords")
	}
	histories, err := s.userRepo.ListPasswordHistories(user.ID, passwordHistoryLimit)
	if err != nil {
		return fmt.Errorf("check password history: %w", err)
	}
	for _, history := range histories {
		if history.PasswordHash == "" {
			continue
		}
		if bcrypt.CompareHashAndPassword([]byte(history.PasswordHash), []byte(newPassword)) == nil {
			return fmt.Errorf("password cannot reuse recent passwords")
		}
	}
	return nil
}

func (s *AuthService) ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user id")
	}
	user, err := s.userRepo.GetByID(uid)
	if err != nil {
		return fmt.Errorf("user not found")
	}
	if user.Source != "local" {
		return fmt.Errorf("password can only be changed for local users")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(oldPassword)); err != nil {
		return fmt.Errorf("old password is incorrect")
	}
	if err := validatePassword(newPassword); err != nil {
		return err
	}
	if err := s.validatePasswordReuse(user, newPassword); err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	user.PasswordHash = string(hash)
	now := time.Now()
	user.PasswordChangedAt = &now
	user.MustChangePassword = false
	if err := s.userRepo.UpdatePasswordWithHistory(user, passwordHistoryLimit); err != nil {
		return err
	}
	s.publishEvent("user.password_changed", user.ID.String(), user.OrgID.String(), user.Username, "change_password", "Password changed")
	return nil
}

func (s *AuthService) ResetUserPassword(userID, newPassword string, mustChange bool) (*model.User, error) {
	if err := validatePassword(newPassword); err != nil {
		return nil, err
	}
	user, err := s.GetUser(userID)
	if err != nil {
		return nil, err
	}
	if user.Source != "local" {
		return nil, fmt.Errorf("password can only be reset for local users")
	}
	if err := s.validatePasswordReuse(user, newPassword); err != nil {
		return nil, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	now := time.Now()
	user.PasswordHash = string(hash)
	user.PasswordChangedAt = &now
	user.MustChangePassword = mustChange
	user.FailedLoginAttempts = 0
	user.LockedUntil = nil
	if user.Status == "locked" {
		user.Status = "active"
	}
	if err := s.userRepo.UpdatePasswordWithHistory(user, passwordHistoryLimit); err != nil {
		return nil, err
	}
	_ = s.userRepo.RevokeAllSessions(user.ID, "password_reset")
	s.publishEvent("user.password_reset", user.ID.String(), user.OrgID.String(), user.Username, "reset_password", "Password reset")
	return user, nil
}

func validatePassword(password string) error {
	if len(password) < minPasswordLength {
		return fmt.Errorf("password must be at least %d characters", minPasswordLength)
	}
	var hasUpper, hasLower, hasDigit, hasSymbol bool
	for _, r := range password {
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsDigit(r):
			hasDigit = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			hasSymbol = true
		case unicode.IsSpace(r):
			return fmt.Errorf("password must not contain whitespace")
		}
	}
	missing := []string{}
	if !hasUpper {
		missing = append(missing, "uppercase")
	}
	if !hasLower {
		missing = append(missing, "lowercase")
	}
	if !hasDigit {
		missing = append(missing, "digit")
	}
	if !hasSymbol {
		missing = append(missing, "symbol")
	}
	if len(missing) > 0 {
		return fmt.Errorf("password must include %s", strings.Join(missing, ", "))
	}
	return nil
}
