package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func testContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func TestIAMHTTPClientCheckPermission(t *testing.T) {
	var gotBody map[string]any
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/v1/check-permission" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"allowed":true,"reason":"","roles":["admin"]}}`))
	}))
	defer backend.Close()

	c, err := NewIAMHTTPClient(backend.URL, 2*time.Second)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	result, err := c.CheckPermission(testContext(t), CheckPermissionRequest{
		UserID: "u_1",
		OrgID:  "org_1",
		Method: http.MethodGet,
		Path:   "/api/v1/users",
	})
	if err != nil {
		t.Fatalf("check permission: %v", err)
	}

	if !result.Allowed || len(result.Roles) != 1 || result.Roles[0] != "admin" {
		t.Fatalf("unexpected result: %+v", result)
	}
	for key, want := range map[string]string{
		"user_id": "u_1",
		"org_id":  "org_1",
		"method":  http.MethodGet,
		"path":    "/api/v1/users",
	} {
		if gotBody[key] != want {
			t.Fatalf("body %s: expected %q, got %#v", key, want, gotBody[key])
		}
	}
}

func TestIAMHTTPClientDeniesApplicationError(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"allowed":false,"reason":"missing permission","roles":[]}}`))
	}))
	defer backend.Close()

	c, err := NewIAMHTTPClient(backend.URL, 2*time.Second)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	result, err := c.CheckPermission(testContext(t), CheckPermissionRequest{
		UserID: "u_1",
		OrgID:  "org_1",
		Method: http.MethodDelete,
		Path:   "/api/v1/users/u_2",
	})
	if err != nil {
		t.Fatalf("check permission: %v", err)
	}
	if result.Allowed || result.Reason != "missing permission" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestIAMHTTPClientAuthorizationCenterMethods(t *testing.T) {
	var gotOrg, gotBody string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		gotOrg = r.Header.Get("X-Org-Id")
		switch r.URL.Path {
		case "/api/v1/users/u_1/roles":
			if r.Method == http.MethodGet {
				_, _ = w.Write([]byte(`{"code":0,"message":"success","data":[{"id":"r_1","code":"admin"}]}`))
				return
			}
			gotBody = readJSONBody(t, r)
			_, _ = w.Write([]byte(`{"code":0,"message":"success"}`))
		case "/api/v1/users/u_1/permissions":
			_, _ = w.Write([]byte(`{"code":0,"message":"success","data":["user:read"]}`))
		case "/api/v1/roles":
			if r.Method == http.MethodGet {
				_, _ = w.Write([]byte(`{"code":0,"message":"success","data":[{"id":"r_1","code":"admin"}]}`))
				return
			}
			gotBody = readJSONBody(t, r)
			_, _ = w.Write([]byte(`{"code":0,"message":"created","data":{"id":"r_2","code":"ops"}}`))
		case "/api/v1/roles/r_1":
			if r.Method == http.MethodDelete {
				_, _ = w.Write([]byte(`{"code":0,"message":"success"}`))
				return
			}
			_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"id":"r_1","code":"admin"}}`))
		case "/api/v1/roles/r_1/permissions":
			gotBody = readJSONBody(t, r)
			_, _ = w.Write([]byte(`{"code":0,"message":"success"}`))
		case "/api/v1/permissions":
			_, _ = w.Write([]byte(`{"code":0,"message":"success","data":[{"code":"user:read"}]}`))
		case "/api/v1/api-permissions":
			if r.Method == http.MethodGet {
				_, _ = w.Write([]byte(`{"code":0,"message":"success","data":[{"id":"api_1","method":"GET"}]}`))
				return
			}
			gotBody = readJSONBody(t, r)
			_, _ = w.Write([]byte(`{"code":0,"message":"created","data":{"id":"api_2","method":"POST"}}`))
		case "/api/v1/api-permissions/api_1":
			if r.Method == http.MethodDelete {
				_, _ = w.Write([]byte(`{"code":0,"message":"success"}`))
				return
			}
			gotBody = readJSONBody(t, r)
			_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"id":"api_1","method":"PUT"}}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer backend.Close()

	c, err := NewIAMHTTPClient(backend.URL, 2*time.Second)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	userCtx := UserContext{UserID: "actor_1", OrgID: "org_1", SessionID: "s_1"}
	roles, err := c.GetUserRoles(testContext(t), "u_1", userCtx)
	if err != nil || len(roles) != 1 || gotOrg != "org_1" {
		t.Fatalf("roles=%+v org=%s err=%v", roles, gotOrg, err)
	}
	if err := c.AssignUserRoles(testContext(t), "u_1", AssignRolesRequest{RoleIDs: []string{"r_1"}}, userCtx); err != nil {
		t.Fatalf("assign roles: %v", err)
	}
	if !strings.Contains(gotBody, `"role_ids":["r_1"]`) {
		t.Fatalf("unexpected assign roles body: %s", gotBody)
	}
	if permissions, err := c.GetUserPermissions(testContext(t), "u_1", userCtx); err != nil || len(permissions) != 1 {
		t.Fatalf("permissions=%+v err=%v", permissions, err)
	}
	if roles, err := c.ListRoles(testContext(t), userCtx); err != nil || len(roles) != 1 {
		t.Fatalf("list roles=%+v err=%v", roles, err)
	}
	if role, err := c.CreateRole(testContext(t), RoleRequest{OrgID: "org_1", Name: "Ops", Code: "ops"}, userCtx); err != nil || role["code"] != "ops" {
		t.Fatalf("create role=%+v err=%v", role, err)
	}
	if role, err := c.GetRole(testContext(t), "r_1", userCtx); err != nil || role["id"] != "r_1" {
		t.Fatalf("get role=%+v err=%v", role, err)
	}
	if _, err := c.UpdateRole(testContext(t), "r_1", RoleRequest{Name: "Admin"}, userCtx); err != nil {
		t.Fatalf("update role: %v", err)
	}
	if err := c.AssignRolePermissions(testContext(t), "r_1", AssignPermissionsRequest{PermissionIDs: []string{"p_1"}}, userCtx); err != nil {
		t.Fatalf("assign permissions: %v", err)
	}
	if err := c.DeleteRole(testContext(t), "r_1", userCtx); err != nil {
		t.Fatalf("delete role: %v", err)
	}
	if tree, err := c.GetPermissionTree(testContext(t), userCtx); err != nil || len(tree) != 1 {
		t.Fatalf("permission tree=%+v err=%v", tree, err)
	}
	if apiPermissions, err := c.ListAPIPermissions(testContext(t), userCtx); err != nil || len(apiPermissions) != 1 {
		t.Fatalf("api permissions=%+v err=%v", apiPermissions, err)
	}
	if created, err := c.CreateAPIPermission(testContext(t), APIPermissionRequest{Method: "POST", PathPattern: "/api/v1/x", PermissionCode: "x:create", Enabled: true}, userCtx); err != nil || created["id"] != "api_2" {
		t.Fatalf("create api permission=%+v err=%v", created, err)
	}
	if updated, err := c.UpdateAPIPermission(testContext(t), "api_1", APIPermissionRequest{Method: "PUT", PathPattern: "/api/v1/x", PermissionCode: "x:update", Enabled: true}, userCtx); err != nil || updated["method"] != "PUT" {
		t.Fatalf("update api permission=%+v err=%v", updated, err)
	}
	if err := c.DeleteAPIPermission(testContext(t), "api_1", userCtx); err != nil {
		t.Fatalf("delete api permission: %v", err)
	}
}

func readJSONBody(t *testing.T, r *http.Request) string {
	t.Helper()
	var payload map[string]any
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	return string(raw)
}
