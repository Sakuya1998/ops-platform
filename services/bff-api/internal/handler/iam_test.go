package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ops-platform/bff-api/internal/client"
)

type recordingIAMService struct {
	userID           string
	userCtx          client.UserContext
	roleReq          client.RoleRequest
	assignRolesReq   client.AssignRolesRequest
	assignPermReq    client.AssignPermissionsRequest
	apiPermissionReq client.APIPermissionRequest
	rolesResp        []map[string]any
	roleResp         map[string]any
	permissionsResp  []map[string]any
	apiPermissions   []map[string]any
	userPermissions  []any
}

func (s *recordingIAMService) AssignUserRoles(ctx context.Context, userID string, req client.AssignRolesRequest, userCtx client.UserContext) error {
	s.userID = userID
	s.assignRolesReq = req
	s.userCtx = userCtx
	return nil
}

func (s *recordingIAMService) GetUserRoles(ctx context.Context, userID string, userCtx client.UserContext) ([]map[string]any, error) {
	s.userID = userID
	s.userCtx = userCtx
	return s.rolesResp, nil
}

func (s *recordingIAMService) GetUserPermissions(ctx context.Context, userID string, userCtx client.UserContext) ([]any, error) {
	s.userID = userID
	s.userCtx = userCtx
	return s.userPermissions, nil
}

func (s *recordingIAMService) ListRoles(ctx context.Context, userCtx client.UserContext) ([]map[string]any, error) {
	s.userCtx = userCtx
	return s.rolesResp, nil
}

func (s *recordingIAMService) CreateRole(ctx context.Context, req client.RoleRequest, userCtx client.UserContext) (map[string]any, error) {
	s.roleReq = req
	s.userCtx = userCtx
	return s.roleResp, nil
}

func (s *recordingIAMService) GetRole(ctx context.Context, id string, userCtx client.UserContext) (map[string]any, error) {
	s.userID = id
	s.userCtx = userCtx
	return s.roleResp, nil
}

func (s *recordingIAMService) UpdateRole(ctx context.Context, id string, req client.RoleRequest, userCtx client.UserContext) (map[string]any, error) {
	s.userID = id
	s.roleReq = req
	s.userCtx = userCtx
	return s.roleResp, nil
}

func (s *recordingIAMService) DeleteRole(ctx context.Context, id string, userCtx client.UserContext) error {
	s.userID = id
	s.userCtx = userCtx
	return nil
}

func (s *recordingIAMService) AssignRolePermissions(ctx context.Context, roleID string, req client.AssignPermissionsRequest, userCtx client.UserContext) error {
	s.userID = roleID
	s.assignPermReq = req
	s.userCtx = userCtx
	return nil
}

func (s *recordingIAMService) GetPermissionTree(ctx context.Context, userCtx client.UserContext) ([]map[string]any, error) {
	s.userCtx = userCtx
	return s.permissionsResp, nil
}

func (s *recordingIAMService) ListAPIPermissions(ctx context.Context, userCtx client.UserContext) ([]map[string]any, error) {
	s.userCtx = userCtx
	return s.apiPermissions, nil
}

func (s *recordingIAMService) CreateAPIPermission(ctx context.Context, req client.APIPermissionRequest, userCtx client.UserContext) (map[string]any, error) {
	s.apiPermissionReq = req
	s.userCtx = userCtx
	return map[string]any{"id": "api_2", "method": req.Method}, nil
}

func (s *recordingIAMService) UpdateAPIPermission(ctx context.Context, id string, req client.APIPermissionRequest, userCtx client.UserContext) (map[string]any, error) {
	s.userID = id
	s.apiPermissionReq = req
	s.userCtx = userCtx
	return map[string]any{"id": id, "method": req.Method}, nil
}

func (s *recordingIAMService) DeleteAPIPermission(ctx context.Context, id string, userCtx client.UserContext) error {
	s.userID = id
	s.userCtx = userCtx
	return nil
}

func TestIAMHandlerRoutesUseExplicitService(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &recordingIAMService{
		rolesResp:       []map[string]any{{"id": "r_1", "code": "admin"}},
		roleResp:        map[string]any{"id": "r_1", "code": "admin"},
		permissionsResp: []map[string]any{{"code": "user:read"}},
		apiPermissions:  []map[string]any{{"id": "api_1", "method": "GET"}},
		userPermissions: []any{"user:read"},
	}
	r := gin.New()
	iam := NewIAMHandler(svc)
	iam.RegisterUserRoutes(r.Group("/api/v1/users"))
	iam.RegisterRoleRoutes(r.Group("/api/v1/roles"))
	iam.RegisterPermissionRoutes(r.Group("/api/v1"))
	iam.RegisterAPIPermissionRoutes(r.Group("/api/v1/api-permissions"))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/u_1/roles", strings.NewReader(`{"role_ids":["r_1"]}`))
	req.Header.Set("X-User-Id", "actor_1")
	req.Header.Set("X-Org-Id", "org_1")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK || svc.userID != "u_1" || len(svc.assignRolesReq.RoleIDs) != 1 || svc.userCtx.OrgID != "org_1" {
		t.Fatalf("unexpected assign user roles status=%d userID=%s req=%+v ctx=%+v", w.Code, svc.userID, svc.assignRolesReq, svc.userCtx)
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/roles", strings.NewReader(`{"org_id":"org_1","name":"Ops","code":"ops"}`))
	req.Header.Set("X-User-Id", "actor_1")
	req.Header.Set("X-Org-Id", "org_1")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated || svc.roleReq.Code != "ops" {
		t.Fatalf("unexpected create role status=%d req=%+v", w.Code, svc.roleReq)
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/permissions", nil)
	req.Header.Set("X-User-Id", "actor_1")
	req.Header.Set("X-Org-Id", "org_1")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("unexpected permission tree status=%d body=%s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/api-permissions", strings.NewReader(`{"method":"GET","path_pattern":"/api/v1/users","permission_code":"user:read","enabled":true}`))
	req.Header.Set("X-User-Id", "actor_1")
	req.Header.Set("X-Org-Id", "org_1")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated || svc.apiPermissionReq.PermissionCode != "user:read" {
		t.Fatalf("unexpected api permission status=%d req=%+v", w.Code, svc.apiPermissionReq)
	}
	var body struct {
		Code int            `json:"code"`
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Code != 0 || body.Data["id"] != "api_2" {
		t.Fatalf("unexpected response: %+v", body)
	}
}
