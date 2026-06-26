package service

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/go-ldap/ldap/v3"
	"github.com/google/uuid"
	"github.com/Sakuya1998/ops-platform/services/auth-svc/internal/config"
	"github.com/Sakuya1998/ops-platform/services/auth-svc/internal/model"
	"github.com/Sakuya1998/ops-platform/services/auth-svc/internal/repository"
)

type LdapService struct {
	cfg      *config.LDAPConfig
	userRepo *repository.UserRepository
}

func NewLdapService(cfg *config.LDAPConfig, userRepo *repository.UserRepository) *LdapService {
	return &LdapService{cfg: cfg, userRepo: userRepo}
}

func (s *LdapService) Authenticate(ctx context.Context, username, password string) (*model.User, error) {
	return s.AuthenticateWithConfig(ctx, username, password, s.cfg)
}

func (s *LdapService) AuthenticateWithConfig(ctx context.Context, username, password string, cfg *config.LDAPConfig) (*model.User, error) {
	if cfg == nil || !cfg.Enabled || cfg.Host == "" {
		return nil, fmt.Errorf("LDAP authentication is disabled")
	}
	if username == "" || password == "" {
		return nil, fmt.Errorf("username and password required")
	}

	conn, err := s.connectWithConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("connect LDAP: %w", err)
	}
	defer conn.Close()

	if err := conn.Bind(cfg.BindDN, cfg.BindPassword); err != nil {
		return nil, fmt.Errorf("bind LDAP service account: %w", err)
	}

	filter := strings.Replace(cfg.UserFilter, "%s", ldap.EscapeFilter(username), -1)
	searchReq := ldap.NewSearchRequest(
		cfg.BaseDN, ldap.ScopeWholeSubtree, ldap.NeverDerefAliases,
		0, 0, false, filter,
		[]string{cfg.UIDAttr, cfg.DisplayNameAttr, cfg.EmailAttr, "dn"},
		nil,
	)

	result, err := conn.Search(searchReq)
	if err != nil {
		return nil, fmt.Errorf("search LDAP: %w", err)
	}
	if len(result.Entries) == 0 {
		return nil, fmt.Errorf("user '%s' not found in LDAP", username)
	}

	entry := result.Entries[0]
	if err := conn.Bind(entry.DN, password); err != nil {
		return nil, fmt.Errorf("invalid credentials for %s", username)
	}

	displayName := entry.GetAttributeValue(cfg.DisplayNameAttr)
	email := entry.GetAttributeValue(cfg.EmailAttr)
	if displayName == "" {
		displayName = username
	}

	orgID := resolveOrgIDFromCode(cfg.DefaultOrgCode)
	user, err := s.userRepo.GetByUsername(orgID, username)
	if err != nil {
		if !cfg.AutoProvision {
			return nil, fmt.Errorf("user %s not found and auto-provision disabled", username)
		}
		user = &model.User{
			OrgID: orgID, Username: username,
			Email: email, DisplayName: displayName,
			Status: "active", Source: "ldap",
		}
		if err := s.userRepo.Create(user); err != nil {
			return nil, fmt.Errorf("create user: %w", err)
		}
		log.Printf("[LDAP] Auto-created user: %s", username)
	} else {
		user.Email = email
		user.DisplayName = displayName
		user.Source = "ldap"
		s.userRepo.Update(user)
	}
	return user, nil
}

func (s *LdapService) connect() (*ldap.Conn, error) {
	return s.connectWithConfig(s.cfg)
}

func (s *LdapService) connectWithConfig(cfg *config.LDAPConfig) (*ldap.Conn, error) {
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	var conn *ldap.Conn
	var err error

	switch cfg.Security {
	case "tls":
		conn, err = ldap.DialTLS("tcp", addr, &tls.Config{
			InsecureSkipVerify: cfg.SkipVerify,
			ServerName:         cfg.Host,
		})
	case "starttls":
		conn, err = ldap.Dial("tcp", addr)
		if err == nil {
			if err = conn.StartTLS(&tls.Config{InsecureSkipVerify: cfg.SkipVerify, ServerName: cfg.Host}); err != nil {
				conn.Close()
				return nil, fmt.Errorf("starttls: %w", err)
			}
		}
	default:
		conn, err = ldap.Dial("tcp", addr)
	}

	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", addr, err)
	}
	conn.SetTimeout(10 * time.Second)
	return conn, nil
}

func (s *LdapService) IsEnabled() bool {
	return s.cfg != nil && s.cfg.Enabled && s.cfg.Host != ""
}

func resolveOrgIDFromCode(orgCode string) uuid.UUID {
	if id, err := uuid.Parse(orgCode); err == nil {
		return id
	}
	return uuid.MustParse("00000000-0000-0000-0000-000000000001")
}
