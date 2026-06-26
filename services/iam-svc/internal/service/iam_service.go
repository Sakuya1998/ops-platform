package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/Sakuya1998/ops-platform/services/iam-svc/internal/model"
	"github.com/Sakuya1998/ops-platform/services/iam-svc/internal/repository"
	"github.com/Sakuya1998/ops-platform/pkg/cache"
	"github.com/Sakuya1998/ops-platform/pkg/config"
	pkgjwt "github.com/Sakuya1998/ops-platform/pkg/jwt"
	"github.com/Sakuya1998/ops-platform/pkg/kafka"
	"gorm.io/gorm"
)

type IAMService struct {
	roleRepo    *repository.RoleRepository
	permRepo    *repository.PermissionRepository
	apiPermRepo *repository.APIPermissionRepository
	kafkaProd   *kafka.Producer
	jwtMgr      *pkgjwt.Manager
	permCache   *cache.Cache
}

func NewIAMService(roleRepo *repository.RoleRepository,
	permRepo *repository.PermissionRepository,
	kafkaProd *kafka.Producer, jwtCfg config.JWTConfig) *IAMService {
	return &IAMService{
		roleRepo: roleRepo, permRepo: permRepo,
		kafkaProd: kafkaProd, jwtMgr: pkgjwt.NewManager(jwtCfg.Secret, jwtCfg.ExpireHour, jwtCfg.Issuer),
	}
}

func (s *IAMService) WithAPIPermissionRepository(apiPermRepo *repository.APIPermissionRepository) *IAMService {
	s.apiPermRepo = apiPermRepo
	return s
}

func (s *IAMService) WithPermissionCache(permissionCache *cache.Cache) *IAMService {
	s.permCache = permissionCache
	return s
}

func (s *IAMService) InvalidateUserPermissionCache(ctx context.Context, userID string) error {
	if s.permCache == nil {
		return nil
	}
	uid, err := parseUUID(userID, "user id")
	if err != nil {
		return err
	}
	return s.permCache.Delete(ctx, permissionCacheKey(uid))
}

type RequestUserContext struct {
	UserID    string
	OrgID     string
	SessionID string
	Roles     []string
}

type PermissionCheck struct {
	Method string
	Path   string
}

type PermissionCheckResult struct {
	Method             string `json:"method"`
	Path               string `json:"path"`
	Allowed            bool   `json:"allowed"`
	Reason             string `json:"reason,omitempty"`
	RequiredPermission string `json:"required_permission,omitempty"`
}

type CreateAPIPermissionInput struct {
	Method         string
	PathPattern    string
	PermissionCode string
	Description    string
	Enabled        bool
}

type UpdateAPIPermissionInput struct {
	Method         string
	PathPattern    string
	PermissionCode string
	Description    string
	Enabled        bool
}

type permissionSnapshot struct {
	RoleCodes       []string `json:"role_codes"`
	PermissionCodes []string `json:"permission_codes"`
	Admin           bool     `json:"admin"`
}

func (s *IAMService) AssignRoles(userID string, roleIDs []string) error {
	uid, err := parseUUID(userID, "user id")
	if err != nil {
		return err
	}
	rids := make([]uuid.UUID, 0, len(roleIDs))
	for _, roleID := range roleIDs {
		rid, err := parseUUID(roleID, "role id")
		if err != nil {
			return err
		}
		rids = append(rids, rid)
	}
	err = s.roleRepo.AssignUserRoles(uid, rids)
	s.publishEvent("user.role_changed", uid.String())
	return err
}

func (s *IAMService) CreateRole(orgID, name, code, description string) (*model.Role, error) {
	oid, err := parseUUID(orgID, "org id")
	if err != nil {
		return nil, err
	}
	role := &model.Role{OrgID: oid, Name: name, Code: code, Description: description, IsSystem: false}
	if err := s.roleRepo.Create(role); err != nil {
		return nil, err
	}
	s.publishEvent("role.created", "")
	return role, nil
}

func (s *IAMService) GetRole(id string) (*model.Role, error) {
	rid, err := parseUUID(id, "role id")
	if err != nil {
		return nil, err
	}
	role, err := s.roleRepo.GetByID(rid)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, notFound("role", err)
	}
	return role, err
}

func (s *IAMService) ListRoles(orgID string) ([]model.Role, error) {
	oid, err := parseUUID(orgID, "org id")
	if err != nil {
		return nil, err
	}
	return s.roleRepo.ListByOrg(oid)
}

func (s *IAMService) UpdateRole(id, name, description string) (*model.Role, error) {
	rid, err := parseUUID(id, "role id")
	if err != nil {
		return nil, err
	}
	role, err := s.roleRepo.GetByID(rid)
	if err != nil {
		return nil, err
	}
	role.Name = name
	role.Description = description
	if err := s.roleRepo.Update(role); err != nil {
		return nil, err
	}
	s.publishEvent("role.updated", "")
	return role, nil
}

func (s *IAMService) DeleteRole(id string) error {
	rid, err := parseUUID(id, "role id")
	if err != nil {
		return err
	}
	if err := s.roleRepo.Delete(rid); err != nil {
		return err
	}
	s.publishEvent("role.deleted", "")
	return nil
}

func (s *IAMService) AssignPermissions(roleID string, permIDs []string) error {
	rid, err := parseUUID(roleID, "role id")
	if err != nil {
		return err
	}
	var pids []uuid.UUID
	for _, pid := range permIDs {
		parsed, err := parseUUID(pid, "permission id")
		if err != nil {
			return err
		}
		pids = append(pids, parsed)
	}
	if err := s.roleRepo.AssignPermissions(rid, pids); err != nil {
		return err
	}
	s.publishEvent("role.permission_changed", "")
	return nil
}

func (s *IAMService) GetPermissionTree() ([]*model.Permission, error) {
	perms, err := s.permRepo.ListAll()
	if err != nil {
		return nil, err
	}
	return buildTree(perms), nil
}

func (s *IAMService) ListAPIPermissions() ([]model.APIPermission, error) {
	if s.apiPermRepo == nil {
		return nil, unavailable("api permission repository is not configured")
	}
	return s.apiPermRepo.List()
}

func (s *IAMService) CreateAPIPermission(input CreateAPIPermissionInput) (*model.APIPermission, error) {
	if s.apiPermRepo == nil {
		return nil, unavailable("api permission repository is not configured")
	}
	permission := &model.APIPermission{
		Method:         strings.ToUpper(strings.TrimSpace(input.Method)),
		PathPattern:    strings.TrimSpace(input.PathPattern),
		PermissionCode: strings.TrimSpace(input.PermissionCode),
		Description:    input.Description,
		Enabled:        input.Enabled,
	}
	if permission.Method == "" || permission.PathPattern == "" || permission.PermissionCode == "" {
		return nil, invalidArgument("api permission")
	}
	if err := s.apiPermRepo.Create(permission); err != nil {
		return nil, err
	}
	s.publishEvent("api_permission.created", "")
	return permission, nil
}

func (s *IAMService) UpdateAPIPermission(id string, input UpdateAPIPermissionInput) (*model.APIPermission, error) {
	if s.apiPermRepo == nil {
		return nil, unavailable("api permission repository is not configured")
	}
	uid, err := parseUUID(id, "api permission id")
	if err != nil {
		return nil, err
	}
	permission := &model.APIPermission{
		ID:             uid,
		Method:         strings.ToUpper(strings.TrimSpace(input.Method)),
		PathPattern:    strings.TrimSpace(input.PathPattern),
		PermissionCode: strings.TrimSpace(input.PermissionCode),
		Description:    input.Description,
		Enabled:        input.Enabled,
	}
	if permission.Method == "" || permission.PathPattern == "" || permission.PermissionCode == "" {
		return nil, invalidArgument("api permission")
	}
	if err := s.apiPermRepo.Update(permission); err != nil {
		return nil, err
	}
	s.publishEvent("api_permission.updated", "")
	return permission, nil
}

func (s *IAMService) DeleteAPIPermission(id string) error {
	if s.apiPermRepo == nil {
		return unavailable("api permission repository is not configured")
	}
	uid, err := parseUUID(id, "api permission id")
	if err != nil {
		return err
	}
	if err := s.apiPermRepo.Delete(uid); err != nil {
		return err
	}
	s.publishEvent("api_permission.deleted", "")
	return nil
}

var routePermissions = map[string]string{
	"GET:/api/v1/auth/me":                     "user:read",
	"POST:/api/v1/auth/logout":                "user:read",
	"PUT:/api/v1/auth/me/password":            "user:update",
	"POST:/api/v1/auth/me/mfa/setup":          "user:update",
	"POST:/api/v1/auth/me/mfa/confirm":        "user:update",
	"POST:/api/v1/auth/me/mfa/recovery-codes": "user:update",
	"DELETE:/api/v1/auth/me/mfa":              "user:update",
	"GET:/api/v1/auth/sessions":               "user:read",
	"DELETE:/api/v1/auth/sessions":            "user:update",
	"DELETE:/api/v1/auth/sessions/:id":        "user:update",
	"POST:/api/v1/users":                      "user:create",
	"GET:/api/v1/users":                       "user:read",
	"GET:/api/v1/users/:id":                   "user:read",
	"PUT:/api/v1/users/:id":                   "user:update",
	"PUT:/api/v1/users/:id/status":            "user:update",
	"PUT:/api/v1/users/:id/password/reset":    "user:update",
	"PUT:/api/v1/users/:id/roles":             "role:assign",
	"GET:/api/v1/users/:id/roles":             "role:read",
	"DELETE:/api/v1/users/:id":                "user:delete",
	"POST:/api/v1/roles":                      "role:create",
	"GET:/api/v1/roles":                       "role:read",
	"GET:/api/v1/roles/:id":                   "role:read",
	"PUT:/api/v1/roles/:id":                   "role:update",
	"DELETE:/api/v1/roles/:id":                "role:delete",
	"PUT:/api/v1/roles/:id/permissions":       "role:assign",
	"GET:/api/v1/permissions":                 "permission:read",
	"POST:/api/v1/organizations":              "org:create",
	"GET:/api/v1/organizations":               "org:read",
	"PUT:/api/v1/organizations/:id":           "org:update",
	"GET:/api/v1/audit-logs":                  "audit:read",
	"GET:/api/v1/audit-logs/event-types":      "audit:read",
	"GET:/api/v1/notifications":               "notify:read",
	"POST:/api/v1/notifications":              "notify:create",
	"PUT:/api/v1/notifications/:id":           "notify:update",
	"DELETE:/api/v1/notifications/:id":        "notify:delete",
	"GET:/api/v1/notify/templates":            "notify:read",
	"POST:/api/v1/notify/templates":           "notify:create",
	"PUT:/api/v1/notify/templates/:id":        "notify:update",
	"DELETE:/api/v1/notify/templates/:id":     "notify:delete",
	"GET:/api/v1/notify/logs":                 "notify:read",
	"GET:/api/v1/system/config":               "org:read",
}

func buildTree(perms []model.Permission) []*model.Permission {
	pmap := make(map[string]*model.Permission)
	var roots []*model.Permission
	for i := range perms {
		p := &perms[i]
		pmap[p.ID.String()] = p
	}
	for i := range perms {
		p := &perms[i]
		if p.ParentID == nil {
			roots = append(roots, pmap[p.ID.String()])
		} else {
			parent := pmap[p.ParentID.String()]
			if parent != nil {
				parent.Children = append(parent.Children, pmap[p.ID.String()])
			}
		}
	}
	return roots
}

func (s *IAMService) ResolveToken(token string) (*RequestUserContext, error) {
	if strings.HasPrefix(strings.ToLower(token), "bearer ") {
		token = strings.TrimSpace(token[7:])
	}
	claims, err := s.jwtMgr.Validate(token)
	if err != nil {
		return nil, err
	}
	return &RequestUserContext{
		UserID: claims.UserID, OrgID: claims.OrgID, SessionID: claims.SessionID,
	}, nil
}

func (s *IAMService) CheckPermission(userID, orgID, method, path string) (bool, string, *RequestUserContext) {
	results, userCtx, err := s.batchCheckPermission(userID, orgID, []PermissionCheck{{Method: method, Path: path}})
	if err != nil {
		return false, err.Error(), nil
	}
	if len(results) == 0 {
		return false, "empty permission check", nil
	}
	return results[0].Allowed, results[0].Reason, userCtx
}

func (s *IAMService) BatchCheckPermission(userID, orgID string, checks []PermissionCheck) ([]PermissionCheckResult, error) {
	results, _, err := s.batchCheckPermission(userID, orgID, checks)
	return results, err
}

func (s *IAMService) batchCheckPermission(userID, orgID string, checks []PermissionCheck) ([]PermissionCheckResult, *RequestUserContext, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, nil, invalidArgument("user id")
	}
	snapshot, err := s.getPermissionSnapshot(context.Background(), uid)
	if err != nil {
		return nil, nil, err
	}
	userCtx := &RequestUserContext{UserID: userID, OrgID: orgID, Roles: snapshot.RoleCodes}
	if snapshot.Admin {
		return allowAllChecks(checks), userCtx, nil
	}
	userPerms := make(map[string]bool, len(snapshot.PermissionCodes))
	for _, code := range snapshot.PermissionCodes {
		userPerms[code] = true
	}
	results := make([]PermissionCheckResult, 0, len(checks))
	for _, check := range checks {
		result := PermissionCheckResult{Method: check.Method, Path: check.Path}
		normalizedPath := normalizeRoutePath(check.Path)
		required, exists, err := s.requiredPermission(check.Method, normalizedPath)
		if err != nil {
			return nil, nil, err
		}
		if !exists {
			result.Reason = "no permission mapping for " + routeKey(check.Method, normalizedPath)
			results = append(results, result)
			continue
		}
		result.RequiredPermission = required
		if userPerms[required] {
			result.Allowed = true
		} else {
			result.Reason = "permission denied: " + required
		}
		results = append(results, result)
	}
	return results, userCtx, nil
}

func (s *IAMService) requiredPermission(method, normalizedPath string) (string, bool, error) {
	method = strings.ToUpper(method)
	if s.apiPermRepo != nil {
		apiPermission, err := s.apiPermRepo.GetByRoute(method, normalizedPath)
		if err == nil {
			return apiPermission.PermissionCode, true, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return "", false, err
		}
		return "", false, nil
	}
	required, exists := routePermissions[routeKey(method, normalizedPath)]
	return required, exists, nil
}

func routeKey(method, normalizedPath string) string {
	return strings.ToUpper(method) + ":" + normalizedPath
}

func (s *IAMService) getPermissionSnapshot(ctx context.Context, uid uuid.UUID) (*permissionSnapshot, error) {
	if s.permCache != nil {
		value, err := s.permCache.GetOrLoad(ctx, permissionCacheKey(uid), func(ctx context.Context) ([]byte, time.Duration, error) {
			snapshot, err := s.loadPermissionSnapshot(uid)
			if err != nil {
				return nil, 0, err
			}
			payload, err := json.Marshal(snapshot)
			if err != nil {
				return nil, 0, err
			}
			return payload, 5 * time.Minute, nil
		})
		if err != nil {
			return nil, err
		}
		var snapshot permissionSnapshot
		if err := json.Unmarshal(value, &snapshot); err != nil {
			_ = s.permCache.Delete(ctx, permissionCacheKey(uid))
			return nil, err
		}
		return &snapshot, nil
	}
	return s.loadPermissionSnapshot(uid)
}

func (s *IAMService) loadPermissionSnapshot(uid uuid.UUID) (*permissionSnapshot, error) {
	roles, err := s.roleRepo.GetUserRoles(uid)
	if err != nil {
		return nil, err
	}
	roleCodes := make([]string, 0, len(roles))
	snapshot := &permissionSnapshot{RoleCodes: roleCodes}
	for _, r := range roles {
		roleCodes = append(roleCodes, r.Code)
		if r.Code == "admin" {
			snapshot.RoleCodes = roleCodes
			snapshot.Admin = true
			return snapshot, nil
		}
	}
	userPerms := make(map[string]bool)
	for _, r := range roles {
		perms, err := s.permRepo.GetRolePermissions(r.ID)
		if err != nil {
			continue
		}
		for _, p := range perms {
			userPerms[p.Code] = true
		}
	}
	snapshot.RoleCodes = roleCodes
	for code := range userPerms {
		snapshot.PermissionCodes = append(snapshot.PermissionCodes, code)
	}
	return snapshot, nil
}

func permissionCacheKey(uid uuid.UUID) string {
	return fmt.Sprintf("iam:user-permissions:%s", uid.String())
}

func allowAllChecks(checks []PermissionCheck) []PermissionCheckResult {
	results := make([]PermissionCheckResult, 0, len(checks))
	for _, check := range checks {
		result := PermissionCheckResult{Method: check.Method, Path: check.Path, Allowed: true}
		if required, exists := routePermissions[routeKey(check.Method, normalizeRoutePath(check.Path))]; exists {
			result.RequiredPermission = required
		}
		results = append(results, result)
	}
	return results
}

func normalizeRoutePath(path string) string {
	if parsed, err := url.Parse(path); err == nil && parsed.Path != "" {
		path = parsed.Path
	}
	path = strings.TrimRight(path, "/")
	if path == "" {
		path = "/"
	}
	segments := strings.Split(path, "/")
	for i, segment := range segments {
		if looksLikeID(segment) {
			segments[i] = ":id"
		}
	}
	return strings.Join(segments, "/")
}

func looksLikeID(segment string) bool {
	if segment == "" {
		return false
	}
	if _, err := uuid.Parse(segment); err == nil {
		return true
	}
	if len(segment) < 8 {
		return false
	}
	hasDigit := false
	for _, r := range segment {
		switch {
		case r >= '0' && r <= '9':
			hasDigit = true
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r == '-' || r == '_':
		default:
			return false
		}
	}
	return hasDigit
}

func (s *IAMService) GetUserPermissions(userID string) ([]string, error) {
	var codes []string
	uid, err := parseUUID(userID, "user id")
	if err != nil {
		return nil, err
	}
	roles, err := s.roleRepo.GetUserRoles(uid)
	if err != nil {
		return nil, err
	}
	for _, r := range roles {
		perms, err := s.permRepo.GetRolePermissions(r.ID)
		if err != nil {
			continue
		}
		for _, p := range perms {
			codes = append(codes, p.Code)
		}
	}
	return codes, nil
}

func (s *IAMService) GetUserRoles(userID string) ([]model.Role, error) {
	uid, err := parseUUID(userID, "user id")
	if err != nil {
		return nil, err
	}
	return s.roleRepo.GetUserRoles(uid)
}

func parseUUID(value, field string) (uuid.UUID, error) {
	id, err := uuid.Parse(strings.TrimSpace(value))
	if err != nil {
		return uuid.Nil, invalidArgument(field)
	}
	return id, nil
}

func (s *IAMService) publishEvent(eventType, userID string) {
	if s.kafkaProd == nil {
		return
	}
	_ = s.kafkaProd.PublishEvent(context.Background(), kafka.Event{
		EventType: eventType, UserID: userID,
		Timestamp: time.Now().Format(time.RFC3339),
	})
}
