package service

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ops-platform/auth-svc/internal/model"
	"golang.org/x/crypto/bcrypt"
)

func (s *AuthService) ListUsers(orgID string, page, pageSize int, keyword string) ([]model.User, int64, error) {
	oid, err := uuid.Parse(orgID)
	if err != nil {
		return nil, 0, fmt.Errorf("invalid org id")
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.userRepo.List(oid, (page-1)*pageSize, pageSize, keyword)
}

func (s *AuthService) CreateUser(orgID, username, email, phone, displayName, password string) (*model.User, error) {
	oid, err := uuid.Parse(orgID)
	if err != nil {
		return nil, fmt.Errorf("invalid org id")
	}
	if err := validatePassword(password); err != nil {
		return nil, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	now := time.Now()
	user := &model.User{
		OrgID: oid, Username: username, Email: email, Phone: phone,
		DisplayName: displayName, PasswordHash: string(hash),
		Status: "active", Source: "local", PasswordChangedAt: &now,
	}
	if err := s.userRepo.CreateWithPasswordHistory(user, passwordHistoryLimit); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	s.publishEvent("user.created", user.ID.String(), user.OrgID.String(), user.Username, "create", "User created")
	return user, nil
}

func (s *AuthService) GetUser(userID string) (*model.User, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user id")
	}
	return s.userRepo.GetByID(uid)
}

func (s *AuthService) UpdateUser(userID, email, phone, displayName string) (*model.User, error) {
	user, err := s.GetUser(userID)
	if err != nil {
		return nil, err
	}
	user.Email = email
	user.Phone = phone
	user.DisplayName = displayName
	if err := s.userRepo.Update(user); err != nil {
		return nil, err
	}
	s.publishEvent("user.updated", user.ID.String(), user.OrgID.String(), user.Username, "update", "User updated")
	return user, nil
}

func (s *AuthService) UpdateUserStatus(userID, status string) (*model.User, error) {
	user, err := s.GetUser(userID)
	if err != nil {
		return nil, err
	}
	user.Status = status
	if err := s.userRepo.Update(user); err != nil {
		return nil, err
	}
	if status == "disabled" || status == "locked" {
		_ = s.userRepo.RevokeAllSessions(user.ID, "user_"+status)
	}
	s.publishEvent("user.status_changed", user.ID.String(), user.OrgID.String(), user.Username, "status", "User status changed")
	return user, nil
}

func (s *AuthService) DeleteUser(userID string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user id")
	}
	user, _ := s.userRepo.GetByID(uid)
	if err := s.userRepo.DeleteIdentityData(uid); err != nil {
		return err
	}
	if user != nil {
		s.publishEvent("user.deleted", user.ID.String(), user.OrgID.String(), user.Username, "delete", "User deleted")
	}
	return nil
}
