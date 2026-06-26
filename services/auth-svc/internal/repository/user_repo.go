package repository

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/Sakuya1998/ops-platform/services/auth-svc/internal/model"
	"gorm.io/gorm"
)

var ErrRefreshTokenAlreadyUsed = errors.New("refresh token already used")

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) DB() *gorm.DB {
	return r.db
}

func (r *UserRepository) Create(user *model.User) error {
	return r.db.Create(user).Error
}

func (r *UserRepository) CreateWithPasswordHistory(user *model.User, keepHistory int) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(user).Error; err != nil {
			return err
		}
		if user.PasswordHash == "" {
			return nil
		}
		return tx.Create(&model.PasswordHistory{
			UserID:       user.ID,
			PasswordHash: user.PasswordHash,
		}).Error
	})
}

func (r *UserRepository) GetByID(id uuid.UUID) (*model.User, error) {
	var user model.User
	err := r.db.Where("id = ? AND deleted_at IS NULL", id).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) GetByUsername(orgID uuid.UUID, username string) (*model.User, error) {
	var user model.User
	err := r.db.Where("org_id = ? AND username = ? AND deleted_at IS NULL", orgID, username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) List(orgID uuid.UUID, offset, limit int, keyword string) ([]model.User, int64, error) {
	var users []model.User
	var total int64
	q := r.db.Model(&model.User{}).Where("org_id = ? AND deleted_at IS NULL", orgID)
	if keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where("username ILIKE ? OR display_name ILIKE ? OR email ILIKE ?", like, like, like)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := q.Offset(offset).Limit(limit).Order("created_at DESC").Find(&users).Error
	return users, total, err
}

func (r *UserRepository) Update(user *model.User) error {
	return r.db.Save(user).Error
}

func (r *UserRepository) UpdatePasswordWithHistory(user *model.User, keepHistory int) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(user).Error; err != nil {
			return err
		}
		if user.PasswordHash == "" {
			return nil
		}
		history := &model.PasswordHistory{
			UserID:       user.ID,
			PasswordHash: user.PasswordHash,
		}
		if err := tx.Create(history).Error; err != nil {
			return err
		}
		if keepHistory <= 0 {
			return nil
		}
		var staleIDs []uuid.UUID
		if err := tx.Model(&model.PasswordHistory{}).
			Where("user_id = ?", user.ID).
			Order("created_at DESC").
			Offset(keepHistory).
			Pluck("id", &staleIDs).Error; err != nil {
			return err
		}
		if len(staleIDs) == 0 {
			return nil
		}
		return tx.Delete(&model.PasswordHistory{}, "id IN ?", staleIDs).Error
	})
}

func (r *UserRepository) DeleteIdentityData(userID uuid.UUID) error {
	now := time.Now()
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.Session{}).
			Where("user_id = ? AND status = ?", userID, "active").
			Updates(map[string]interface{}{
				"status":         "revoked",
				"revoked_at":     now,
				"revoked_reason": "user_deleted",
			}).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.RefreshToken{}).
			Where("user_id = ? AND revoked = false", userID).
			Updates(map[string]interface{}{
				"revoked":        true,
				"revoked_at":     now,
				"revoked_reason": "user_deleted",
			}).Error; err != nil {
			return err
		}
		tables := []interface{}{&model.UserCredential{}, &model.PasswordHistory{}, &model.MFARecoveryCode{}}
		for _, table := range tables {
			if err := tx.Delete(table, "user_id = ?", userID).Error; err != nil {
				return err
			}
		}
		return tx.Model(&model.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
			"status":                "deleted",
			"password_hash":         "",
			"mfa_enabled":           false,
			"mfa_secret":            "",
			"mfa_confirmed_at":      nil,
			"failed_login_attempts": 0,
			"locked_until":          nil,
			"deleted_at":            now,
		}).Error
	})
}

func (r *UserRepository) ListPasswordHistories(userID uuid.UUID, limit int) ([]model.PasswordHistory, error) {
	var histories []model.PasswordHistory
	q := r.db.Where("user_id = ?", userID).Order("created_at DESC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	err := q.Find(&histories).Error
	return histories, err
}

func (r *UserRepository) ReplaceMFARecoveryCodes(userID uuid.UUID, codeHashes []string) error {
	now := time.Now()
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.MFARecoveryCode{}).
			Where("user_id = ? AND used_at IS NULL", userID).
			Update("used_at", now).Error; err != nil {
			return err
		}
		codes := make([]model.MFARecoveryCode, 0, len(codeHashes))
		for _, hash := range codeHashes {
			codes = append(codes, model.MFARecoveryCode{UserID: userID, CodeHash: hash})
		}
		if len(codes) == 0 {
			return nil
		}
		return tx.Create(&codes).Error
	})
}

func (r *UserRepository) ConsumeMFARecoveryCode(userID uuid.UUID, codeHash string) (bool, error) {
	now := time.Now()
	result := r.db.Model(&model.MFARecoveryCode{}).
		Where("user_id = ? AND code_hash = ? AND used_at IS NULL", userID, codeHash).
		Update("used_at", now)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func (r *UserRepository) GetCredential(orgID uuid.UUID, provider, providerUserID string) (*model.UserCredential, error) {
	var cred model.UserCredential
	err := r.db.Where("org_id = ? AND provider = ? AND provider_user_id = ?", orgID, provider, providerUserID).First(&cred).Error
	if err != nil {
		return nil, err
	}
	return &cred, nil
}

func (r *UserRepository) UpsertCredential(cred *model.UserCredential) error {
	var existing model.UserCredential
	err := r.db.Where("org_id = ? AND provider = ? AND provider_user_id = ?", cred.OrgID, cred.Provider, cred.ProviderUserID).First(&existing).Error
	if err != nil {
		return r.db.Create(cred).Error
	}
	existing.UserID = cred.UserID
	existing.Username = cred.Username
	existing.Email = cred.Email
	existing.RawProfile = cred.RawProfile
	return r.db.Save(&existing).Error
}

func (r *UserRepository) UpdateLastLogin(id uuid.UUID) error {
	return r.db.Model(&model.User{}).Where("id = ?", id).
		Update("last_login_at", gorm.Expr("NOW()")).Error
}

func (r *UserRepository) RecordFailedLogin(id uuid.UUID, maxAttempts int, lockDuration time.Duration) error {
	var user model.User
	if err := r.db.First(&user, "id = ?", id).Error; err != nil {
		return err
	}
	user.FailedLoginAttempts++
	if maxAttempts > 0 && user.FailedLoginAttempts >= maxAttempts {
		lockedUntil := time.Now().Add(lockDuration)
		user.LockedUntil = &lockedUntil
		user.Status = "locked"
	}
	return r.db.Save(&user).Error
}

func (r *UserRepository) ResetLoginFailures(id uuid.UUID) error {
	return r.db.Model(&model.User{}).Where("id = ?", id).Updates(map[string]interface{}{
		"failed_login_attempts": 0,
		"locked_until":          nil,
		"status":                "active",
	}).Error
}

func (r *UserRepository) CreateRefreshToken(rt *model.RefreshToken) error {
	return r.db.Create(rt).Error
}

func (r *UserRepository) GetRefreshTokenByHash(hash string) (*model.RefreshToken, error) {
	var rt model.RefreshToken
	err := r.db.Where("token_hash = ? AND revoked = false AND expires_at > NOW()", hash).First(&rt).Error
	if err != nil {
		return nil, err
	}
	return &rt, nil
}

func (r *UserRepository) GetRefreshTokenRecordByHash(hash string) (*model.RefreshToken, error) {
	var rt model.RefreshToken
	err := r.db.Where("token_hash = ?", hash).First(&rt).Error
	if err != nil {
		return nil, err
	}
	return &rt, nil
}

func (r *UserRepository) RevokeRefreshToken(id uuid.UUID) error {
	now := time.Now()
	return r.db.Model(&model.RefreshToken{}).Where("id = ?", id).Updates(map[string]interface{}{
		"revoked":        true,
		"revoked_at":     now,
		"revoked_reason": "rotated",
	}).Error
}

func (r *UserRepository) RotateRefreshToken(oldID uuid.UUID, next *model.RefreshToken) error {
	now := time.Now()
	return r.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&model.RefreshToken{}).
			Where("id = ? AND revoked = false", oldID).
			Updates(map[string]interface{}{
				"revoked":        true,
				"revoked_at":     now,
				"revoked_reason": "rotated",
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrRefreshTokenAlreadyUsed
		}
		return tx.Create(next).Error
	})
}

type ProviderRepository struct {
	db *gorm.DB
}

func NewProviderRepository(db *gorm.DB) *ProviderRepository {
	return &ProviderRepository{db: db}
}

func (r *UserRepository) RevokeAllRefreshTokens(userID uuid.UUID) error {
	now := time.Now()
	return r.db.Model(&model.RefreshToken{}).Where("user_id = ? AND revoked = false", userID).Updates(map[string]interface{}{
		"revoked":        true,
		"revoked_at":     now,
		"revoked_reason": "logout",
	}).Error
}

func (r *UserRepository) RevokeAllSessions(userID uuid.UUID, reason string) error {
	now := time.Now()
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.Session{}).
			Where("user_id = ? AND status = ?", userID, "active").
			Updates(map[string]interface{}{
				"status":         "revoked",
				"revoked_at":     now,
				"revoked_reason": reason,
			}).Error; err != nil {
			return err
		}
		return tx.Model(&model.RefreshToken{}).
			Where("user_id = ? AND revoked = false", userID).
			Updates(map[string]interface{}{
				"revoked":        true,
				"revoked_at":     now,
				"revoked_reason": reason,
			}).Error
	})
}

func (r *UserRepository) CreateSession(session *model.Session) error {
	return r.db.Create(session).Error
}

func (r *UserRepository) GetSessionByID(id uuid.UUID) (*model.Session, error) {
	var session model.Session
	err := r.db.First(&session, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *UserRepository) ListSessions(userID uuid.UUID) ([]model.Session, error) {
	var sessions []model.Session
	err := r.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&sessions).Error
	return sessions, err
}

func (r *UserRepository) TouchSession(id uuid.UUID) error {
	return r.db.Model(&model.Session{}).Where("id = ? AND status = ?", id, "active").
		Update("last_seen_at", gorm.Expr("NOW()")).Error
}

func (r *UserRepository) RevokeSession(id uuid.UUID, reason string) error {
	now := time.Now()
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.Session{}).Where("id = ?", id).Updates(map[string]interface{}{
			"status":         "revoked",
			"revoked_at":     now,
			"revoked_reason": reason,
		}).Error; err != nil {
			return err
		}
		return tx.Model(&model.RefreshToken{}).Where("session_id = ? AND revoked = false", id).Updates(map[string]interface{}{
			"revoked":        true,
			"revoked_at":     now,
			"revoked_reason": reason,
		}).Error
	})
}

func (r *UserRepository) RevokeOtherSessions(userID, keepSessionID uuid.UUID, reason string) error {
	now := time.Now()
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.Session{}).
			Where("user_id = ? AND id <> ? AND status = ?", userID, keepSessionID, "active").
			Updates(map[string]interface{}{
				"status":         "revoked",
				"revoked_at":     now,
				"revoked_reason": reason,
			}).Error; err != nil {
			return err
		}
		return tx.Model(&model.RefreshToken{}).
			Where("user_id = ? AND session_id <> ? AND revoked = false", userID, keepSessionID).
			Updates(map[string]interface{}{
				"revoked":        true,
				"revoked_at":     now,
				"revoked_reason": reason,
			}).Error
	})
}

func (r *ProviderRepository) GetByOrgAndProvider(orgID uuid.UUID, provider string) (*model.AuthProvider, error) {
	var p model.AuthProvider
	err := r.db.Where("org_id = ? AND provider = ?", orgID, provider).First(&p).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *ProviderRepository) Upsert(orgID uuid.UUID, provider, name, config string, enabled bool) error {
	var p model.AuthProvider
	err := r.db.Where("org_id = ? AND provider = ?", orgID, provider).First(&p).Error
	if err != nil {
		now := time.Now()
		return r.db.Table("auth_providers").Create(map[string]interface{}{
			"id":         uuid.New(),
			"org_id":     orgID,
			"provider":   provider,
			"name":       name,
			"config":     config,
			"is_enabled": enabled,
			"created_at": now,
			"updated_at": now,
		}).Error
	}
	p.Name = name
	p.Config = config
	p.IsEnabled = enabled
	return r.db.Save(&p).Error
}

type OrganizationRepository struct {
	db *gorm.DB
}

func NewOrganizationRepository(db *gorm.DB) *OrganizationRepository {
	return &OrganizationRepository{db: db}
}

func (r *OrganizationRepository) Create(org *model.Organization) error {
	return r.db.Create(org).Error
}

func (r *OrganizationRepository) List() ([]model.Organization, error) {
	var orgs []model.Organization
	err := r.db.Order("created_at DESC").Find(&orgs).Error
	return orgs, err
}

func (r *OrganizationRepository) GetByID(id uuid.UUID) (*model.Organization, error) {
	var org model.Organization
	err := r.db.First(&org, "id = ?", id).Error
	return &org, err
}

func (r *OrganizationRepository) GetByCode(code string) (*model.Organization, error) {
	var org model.Organization
	err := r.db.First(&org, "code = ?", code).Error
	return &org, err
}

func (r *OrganizationRepository) Update(org *model.Organization) error {
	return r.db.Save(org).Error
}
