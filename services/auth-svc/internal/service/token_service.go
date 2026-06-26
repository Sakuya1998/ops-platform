package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/Sakuya1998/ops-platform/services/auth-svc/internal/model"
	"github.com/Sakuya1998/ops-platform/services/auth-svc/internal/repository"
	"github.com/Sakuya1998/ops-platform/pkg/jwt"
)

type TokenVerifyResult struct {
	Active    bool   `json:"active"`
	UserID    string `json:"user_id,omitempty"`
	OrgID     string `json:"org_id,omitempty"`
	SessionID string `json:"session_id,omitempty"`
	JTI       string `json:"jti,omitempty"`
	ExpiresAt int64  `json:"expires_at,omitempty"`
	IssuedAt  int64  `json:"issued_at,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

func (s *AuthService) ValidateToken(tokenStr string) (*jwt.Claims, error) {
	return s.jwtManager.Validate(tokenStr)
}

func (s *AuthService) RefreshToken(ctx context.Context, refreshTokenStr string) (*LoginResult, error) {
	hash := sha256.Sum256([]byte(refreshTokenStr))
	hashStr := hex.EncodeToString(hash[:])

	stored, err := s.userRepo.GetRefreshTokenByHash(hashStr)
	if err != nil {
		if replayErr := s.handleRefreshTokenReplay(hashStr); replayErr != nil {
			return nil, replayErr
		}
		return nil, fmt.Errorf("invalid or expired refresh token")
	}

	user, err := s.userRepo.GetByID(stored.UserID)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	session, err := s.userRepo.GetSessionByID(stored.SessionID)
	if err != nil || session.Status != "active" || time.Now().After(session.ExpiresAt) {
		return nil, fmt.Errorf("session expired or revoked")
	}

	if user.Status != "active" {
		return nil, fmt.Errorf("user is %s", user.Status)
	}
	roles := s.loadRoleCodes(ctx, user.ID.String())

	jti := uuid.New().String()
	accessToken, err := s.jwtManager.GenerateWithSession(user.ID.String(), user.OrgID.String(), stored.SessionID.String(), jti)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}
	newRefreshToken, nextToken, err := s.newRefreshToken(stored.UserID, stored.SessionID, jti)
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}
	if err := s.userRepo.RotateRefreshToken(stored.ID, nextToken); err != nil {
		if errors.Is(err, repository.ErrRefreshTokenAlreadyUsed) {
			_ = s.userRepo.RevokeSession(stored.SessionID, "refresh_token_reuse")
			s.publishEvent("auth.refresh_reuse", stored.UserID.String(), user.OrgID.String(), user.Username, "refresh", "Refresh token reuse detected")
			return nil, fmt.Errorf("refresh token reuse detected")
		}
		return nil, fmt.Errorf("rotate refresh token: %w", err)
	}
	_ = s.userRepo.TouchSession(stored.SessionID)

	return &LoginResult{
		AccessToken:        accessToken,
		RefreshToken:       newRefreshToken,
		ExpiresIn:          int64(s.cfg.ExpireHour) * 3600,
		SessionID:          stored.SessionID.String(),
		JTI:                jti,
		UserID:             user.ID.String(),
		OrgID:              user.OrgID.String(),
		Username:           user.Username,
		DisplayName:        user.DisplayName,
		Email:              user.Email,
		Roles:              roles,
		MustChangePassword: user.MustChangePassword,
		MFAEnabled:         user.MFAEnabled,
	}, nil
}

func (s *AuthService) createSession(user *model.User, loginCtx LoginContext) (*model.Session, error) {
	session := &model.Session{
		UserID:     user.ID,
		OrgID:      user.OrgID,
		Status:     "active",
		IP:         loginCtx.IP,
		UserAgent:  loginCtx.UserAgent,
		DeviceName: loginCtx.DeviceName,
		ExpiresAt:  time.Now().Add(7 * 24 * time.Hour),
	}
	if err := s.userRepo.CreateSession(session); err != nil {
		return nil, err
	}
	return session, nil
}

func (s *AuthService) generateRefreshToken(userID, sessionID uuid.UUID, jti string) (string, error) {
	token, rt, err := s.newRefreshToken(userID, sessionID, jti)
	if err != nil {
		return "", err
	}
	return token, s.userRepo.CreateRefreshToken(rt)
}

func (s *AuthService) newRefreshToken(userID, sessionID uuid.UUID, jti string) (string, *model.RefreshToken, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", nil, fmt.Errorf("generate refresh token entropy: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	hash := sha256.Sum256([]byte(token))

	rt := &model.RefreshToken{
		UserID:    userID,
		SessionID: sessionID,
		JTI:       jti,
		TokenHash: hex.EncodeToString(hash[:]),
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}
	return token, rt, nil
}

func (s *AuthService) handleRefreshTokenReplay(hash string) error {
	stored, err := s.userRepo.GetRefreshTokenRecordByHash(hash)
	if err != nil {
		return nil
	}
	if !stored.Revoked {
		return nil
	}
	if stored.SessionID != uuid.Nil {
		_ = s.userRepo.RevokeSession(stored.SessionID, "refresh_token_reuse")
	}
	s.publishEvent("auth.refresh_reuse", stored.UserID.String(), "", "", "refresh", "Refresh token reuse detected")
	return fmt.Errorf("refresh token reuse detected")
}

func (s *AuthService) RevokeAllUserTokens(ctx context.Context, userID string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user id")
	}
	return s.userRepo.RevokeAllSessions(uid, "logout")
}

func (s *AuthService) VerifyToken(tokenStr string) TokenVerifyResult {
	claims, err := s.ValidateToken(tokenStr)
	if err != nil {
		return TokenVerifyResult{Active: false, Reason: err.Error()}
	}
	if claims.SessionID != "" {
		sessionID, err := uuid.Parse(claims.SessionID)
		if err != nil {
			return TokenVerifyResult{Active: false, Reason: "invalid session id"}
		}
		session, err := s.userRepo.GetSessionByID(sessionID)
		if err != nil {
			return TokenVerifyResult{Active: false, Reason: "session not found"}
		}
		if session.Status != "active" {
			return TokenVerifyResult{Active: false, Reason: "session revoked"}
		}
		if time.Now().After(session.ExpiresAt) {
			return TokenVerifyResult{Active: false, Reason: "session expired"}
		}
	}
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return TokenVerifyResult{Active: false, Reason: "invalid user id"}
	}
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return TokenVerifyResult{Active: false, Reason: "user not found"}
	}
	if user.Status != "active" {
		return TokenVerifyResult{Active: false, Reason: "user is " + user.Status}
	}
	result := TokenVerifyResult{
		Active:    true,
		UserID:    claims.UserID,
		OrgID:     claims.OrgID,
		SessionID: claims.SessionID,
		JTI:       claims.ID,
	}
	if claims.ExpiresAt != nil {
		result.ExpiresAt = claims.ExpiresAt.Unix()
	}
	if claims.IssuedAt != nil {
		result.IssuedAt = claims.IssuedAt.Unix()
	}
	return result
}

func (s *AuthService) ListSessions(userID string) ([]model.Session, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user id")
	}
	return s.userRepo.ListSessions(uid)
}

func (s *AuthService) RevokeSession(userID, sessionID, reason string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user id")
	}
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return fmt.Errorf("invalid session id")
	}
	session, err := s.userRepo.GetSessionByID(sid)
	if err != nil {
		return fmt.Errorf("session not found")
	}
	if session.UserID != uid {
		return fmt.Errorf("session does not belong to user")
	}
	if reason == "" {
		reason = "revoked"
	}
	return s.userRepo.RevokeSession(sid, reason)
}

func (s *AuthService) RevokeOtherSessions(userID, keepSessionID string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user id")
	}
	sid, err := uuid.Parse(keepSessionID)
	if err != nil {
		return fmt.Errorf("invalid session id")
	}
	return s.userRepo.RevokeOtherSessions(uid, sid, "revoked_by_user")
}
