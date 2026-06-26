package service

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base32"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ops-platform/auth-svc/internal/model"
)

type MFASetupResult struct {
	Secret     string `json:"secret"`
	OTPAuthURL string `json:"otpauth_url"`
	Enabled    bool   `json:"enabled"`
}

type MFAConfirmResult struct {
	RecoveryCodes []string `json:"recovery_codes"`
}

func (s *AuthService) SetupMFA(userID string) (*MFASetupResult, error) {
	user, err := s.GetUser(userID)
	if err != nil {
		return nil, err
	}
	if user.MFAEnabled {
		return nil, fmt.Errorf("mfa is already enabled")
	}
	secret, err := generateTOTPSecret()
	if err != nil {
		return nil, fmt.Errorf("generate mfa secret: %w", err)
	}
	encrypted, err := s.secretBox.EncryptString(secret)
	if err != nil {
		return nil, fmt.Errorf("encrypt mfa secret: %w", err)
	}
	user.MFASecret = encrypted
	user.MFAEnabled = false
	user.MFAConfirmedAt = nil
	if err := s.userRepo.Update(user); err != nil {
		return nil, err
	}
	return &MFASetupResult{
		Secret:     secret,
		OTPAuthURL: buildOTPAuthURL("Ops Platform", user.Username, secret),
		Enabled:    false,
	}, nil
}

func (s *AuthService) ConfirmMFA(userID, code string) (*MFAConfirmResult, error) {
	user, err := s.GetUser(userID)
	if err != nil {
		return nil, err
	}
	ok, err := s.verifyUserTOTP(user, code)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("invalid mfa code")
	}
	recoveryCodes, err := s.replaceMFARecoveryCodes(user.ID)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	user.MFAEnabled = true
	user.MFAConfirmedAt = &now
	if err := s.userRepo.Update(user); err != nil {
		return nil, err
	}
	s.publishEvent("user.mfa_enabled", user.ID.String(), user.OrgID.String(), user.Username, "mfa", "MFA enabled")
	return &MFAConfirmResult{RecoveryCodes: recoveryCodes}, nil
}

func (s *AuthService) DisableMFA(userID, code string) error {
	user, err := s.GetUser(userID)
	if err != nil {
		return err
	}
	if user.MFAEnabled {
		ok, err := s.verifyUserTOTP(user, code)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("invalid mfa code")
		}
	}
	user.MFAEnabled = false
	user.MFASecret = ""
	user.MFAConfirmedAt = nil
	if err := s.userRepo.ReplaceMFARecoveryCodes(user.ID, nil); err != nil {
		return err
	}
	if err := s.userRepo.Update(user); err != nil {
		return err
	}
	s.publishEvent("user.mfa_disabled", user.ID.String(), user.OrgID.String(), user.Username, "mfa", "MFA disabled")
	return nil
}

func (s *AuthService) RegenerateMFARecoveryCodes(userID, code string) (*MFAConfirmResult, error) {
	user, err := s.GetUser(userID)
	if err != nil {
		return nil, err
	}
	if !user.MFAEnabled {
		return nil, fmt.Errorf("mfa is not enabled")
	}
	ok, err := s.verifyUserTOTP(user, code)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("invalid mfa code")
	}
	recoveryCodes, err := s.replaceMFARecoveryCodes(user.ID)
	if err != nil {
		return nil, err
	}
	s.publishEvent("user.mfa_recovery_codes_rotated", user.ID.String(), user.OrgID.String(), user.Username, "mfa", "MFA recovery codes regenerated")
	return &MFAConfirmResult{RecoveryCodes: recoveryCodes}, nil
}

func (s *AuthService) verifyUserMFA(user *model.User, code string) (bool, error) {
	ok, err := s.verifyUserTOTP(user, code)
	if err != nil {
		return false, err
	}
	if ok {
		return true, nil
	}
	return s.consumeMFARecoveryCode(user.ID, code)
}

func (s *AuthService) verifyUserTOTP(user *model.User, code string) (bool, error) {
	if strings.TrimSpace(user.MFASecret) == "" {
		return false, fmt.Errorf("mfa is not configured")
	}
	secret, err := s.secretBox.DecryptString(user.MFASecret)
	if err != nil {
		return false, fmt.Errorf("decrypt mfa secret: %w", err)
	}
	return verifyTOTP(secret, code, time.Now()), nil
}

func (s *AuthService) replaceMFARecoveryCodes(userID uuid.UUID) ([]string, error) {
	codes, err := generateMFARecoveryCodes(10)
	if err != nil {
		return nil, fmt.Errorf("generate recovery codes: %w", err)
	}
	hashes := make([]string, 0, len(codes))
	for _, code := range codes {
		hashes = append(hashes, s.hashRecoveryCode(code))
	}
	if err := s.userRepo.ReplaceMFARecoveryCodes(userID, hashes); err != nil {
		return nil, fmt.Errorf("save recovery codes: %w", err)
	}
	return codes, nil
}

func (s *AuthService) consumeMFARecoveryCode(userID uuid.UUID, code string) (bool, error) {
	normalized := normalizeRecoveryCode(code)
	if normalized == "" {
		return false, nil
	}
	return s.userRepo.ConsumeMFARecoveryCode(userID, s.hashRecoveryCode(normalized))
}

func generateTOTPSecret() (string, error) {
	raw := make([]byte, 20)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return strings.TrimRight(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(raw), "="), nil
}

func generateMFARecoveryCodes(count int) ([]string, error) {
	codes := make([]string, 0, count)
	for i := 0; i < count; i++ {
		raw := make([]byte, 5)
		if _, err := rand.Read(raw); err != nil {
			return nil, err
		}
		encoded := strings.ToUpper(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(raw))
		if len(encoded) > 8 {
			encoded = encoded[:8]
		}
		code := encoded[:4] + "-" + encoded[4:]
		codes = append(codes, code)
	}
	return codes, nil
}

func normalizeRecoveryCode(code string) string {
	code = strings.ToUpper(strings.TrimSpace(code))
	code = strings.ReplaceAll(code, "-", "")
	code = strings.ReplaceAll(code, " ", "")
	if len(code) != 8 {
		return ""
	}
	for _, r := range code {
		if (r < 'A' || r > 'Z') && (r < '2' || r > '7') {
			return ""
		}
	}
	return code
}

func hashRecoveryCode(code string) string {
	normalized := normalizeRecoveryCode(code)
	sum := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(sum[:])
}

func (s *AuthService) hashRecoveryCode(code string) string {
	normalized := normalizeRecoveryCode(code)
	if normalized == "" {
		return ""
	}
	key := s.secretPepper
	if len(key) == 0 {
		key = []byte(s.cfg.Secret)
	}
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte("auth-svc:mfa-recovery-code:v1:"))
	_, _ = mac.Write([]byte(normalized))
	return hex.EncodeToString(mac.Sum(nil))
}

func buildOTPAuthURL(issuer, account, secret string) string {
	label := url.PathEscape(issuer + ":" + account)
	values := url.Values{}
	values.Set("secret", secret)
	values.Set("issuer", issuer)
	values.Set("algorithm", "SHA1")
	values.Set("digits", "6")
	values.Set("period", "30")
	return "otpauth://totp/" + label + "?" + values.Encode()
}

func verifyTOTP(secret, code string, now time.Time) bool {
	code = strings.TrimSpace(code)
	if len(code) != 6 {
		return false
	}
	for _, r := range code {
		if r < '0' || r > '9' {
			return false
		}
	}
	normalized := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(secret), " ", ""))
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(normalized)
	if err != nil {
		if padded := normalized + strings.Repeat("=", (8-len(normalized)%8)%8); padded != normalized {
			key, err = base32.StdEncoding.DecodeString(padded)
		}
	}
	if err != nil {
		return false
	}
	counter := now.Unix() / 30
	for offset := int64(-1); offset <= 1; offset++ {
		if generateTOTPCode(key, uint64(counter+offset)) == code {
			return true
		}
	}
	return false
}

func generateTOTPCode(key []byte, counter uint64) string {
	var msg [8]byte
	binary.BigEndian.PutUint64(msg[:], counter)
	mac := hmac.New(sha1.New, key)
	_, _ = mac.Write(msg[:])
	sum := mac.Sum(nil)
	offset := sum[len(sum)-1] & 0x0f
	value := (uint32(sum[offset])&0x7f)<<24 |
		(uint32(sum[offset+1])&0xff)<<16 |
		(uint32(sum[offset+2])&0xff)<<8 |
		(uint32(sum[offset+3]) & 0xff)
	return fmt.Sprintf("%06d", value%1000000)
}
