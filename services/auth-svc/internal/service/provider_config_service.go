package service

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	authconfig "github.com/Sakuya1998/ops-platform/services/auth-svc/internal/config"
)

type SystemConfig struct {
	LDAP authconfig.LDAPConfig `json:"ldap"`
	OIDC authconfig.OIDCConfig `json:"oidc"`
}

func (s *AuthService) ldapConfigForOrg(orgID uuid.UUID) *authconfig.LDAPConfig {
	cfg := *s.ldapSvc.cfg
	cfg.DefaultOrgCode = orgID.String()
	if s.providerRepo == nil {
		return &cfg
	}
	provider, err := s.providerRepo.GetByOrgAndProvider(orgID, "ldap")
	if err != nil || provider.Config == "" {
		return &cfg
	}
	_ = json.Unmarshal([]byte(provider.Config), &cfg)
	cfg.Enabled = provider.IsEnabled
	cfg.DefaultOrgCode = orgID.String()
	_ = s.decryptLDAPConfig(&cfg)
	return &cfg
}

func (s *AuthService) GetSystemConfig(orgID string) (*SystemConfig, error) {
	oid, err := uuid.Parse(orgID)
	if err != nil {
		return nil, fmt.Errorf("invalid org id")
	}
	ldapCfg := *s.ldapSvc.cfg
	ldapCfg.DefaultOrgCode = oid.String()
	if provider, err := s.providerRepo.GetByOrgAndProvider(oid, "ldap"); err == nil && provider.Config != "" {
		_ = json.Unmarshal([]byte(provider.Config), &ldapCfg)
		ldapCfg.Enabled = provider.IsEnabled
		ldapCfg.DefaultOrgCode = oid.String()
		_ = s.decryptLDAPConfig(&ldapCfg)
	}
	oidcCfg := authconfig.OIDCConfig{}
	if provider, err := s.providerRepo.GetByOrgAndProvider(oid, "oidc"); err == nil && provider.Config != "" {
		_ = json.Unmarshal([]byte(provider.Config), &oidcCfg)
		oidcCfg.Enabled = provider.IsEnabled
		_ = s.decryptOIDCConfig(&oidcCfg)
	} else {
		oidcCfg = authconfig.OIDCConfig{}
	}
	return &SystemConfig{LDAP: ldapCfg, OIDC: oidcCfg}, nil
}

func (s *AuthService) UpdateSystemConfig(orgID string, cfg SystemConfig) error {
	oid, err := uuid.Parse(orgID)
	if err != nil {
		return fmt.Errorf("invalid org id")
	}
	cfg.LDAP.DefaultOrgCode = oid.String()
	ldapCfg := cfg.LDAP
	if err := s.encryptLDAPConfig(&ldapCfg); err != nil {
		return fmt.Errorf("encrypt ldap config: %w", err)
	}
	ldapData, err := json.Marshal(ldapCfg)
	if err != nil {
		return fmt.Errorf("marshal ldap config: %w", err)
	}
	if err := s.providerRepo.Upsert(oid, "ldap", "LDAP Authentication", string(ldapData), cfg.LDAP.Enabled); err != nil {
		return err
	}
	cfg.OIDC.DefaultOrgCode = oid.String()
	oidcCfg := cfg.OIDC
	if err := s.encryptOIDCConfig(&oidcCfg); err != nil {
		return fmt.Errorf("encrypt oidc config: %w", err)
	}
	oidcData, err := json.Marshal(oidcCfg)
	if err != nil {
		return fmt.Errorf("marshal oidc config: %w", err)
	}
	if err := s.providerRepo.Upsert(oid, "oidc", "OIDC Authentication", string(oidcData), cfg.OIDC.Enabled); err != nil {
		return err
	}
	s.publishEvent("system.config_updated", "", oid.String(), "", "update", "Authentication provider config updated")
	return nil
}

func (s *AuthService) encryptLDAPConfig(cfg *authconfig.LDAPConfig) error {
	encrypted, err := s.secretBox.EncryptString(cfg.BindPassword)
	if err != nil {
		return err
	}
	cfg.BindPassword = encrypted
	return nil
}

func (s *AuthService) decryptLDAPConfig(cfg *authconfig.LDAPConfig) error {
	decrypted, err := s.secretBox.DecryptString(cfg.BindPassword)
	if err != nil {
		return err
	}
	cfg.BindPassword = decrypted
	return nil
}

func (s *AuthService) encryptOIDCConfig(cfg *authconfig.OIDCConfig) error {
	encrypted, err := s.secretBox.EncryptString(cfg.ClientSecret)
	if err != nil {
		return err
	}
	cfg.ClientSecret = encrypted
	return nil
}

func (s *AuthService) decryptOIDCConfig(cfg *authconfig.OIDCConfig) error {
	decrypted, err := s.secretBox.DecryptString(cfg.ClientSecret)
	if err != nil {
		return err
	}
	cfg.ClientSecret = decrypted
	return nil
}
