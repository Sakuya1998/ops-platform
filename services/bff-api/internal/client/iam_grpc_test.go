package client

import (
	"context"
	"net"
	"testing"

	iamv1 "github.com/ops-platform/pkg/proto/iam/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

func TestIAMGRPCClientAuthorizationCenterMethods(t *testing.T) {
	conn := newIAMBufConn(t, &stubIAMServer{})
	defer conn.Close()

	c := NewIAMGRPCClient(conn)
	userCtx := UserContext{UserID: "actor_1", OrgID: "org_1"}
	decision, err := c.CheckPermission(testContext(t), CheckPermissionRequest{
		UserID: "u_1", OrgID: "org_1", Method: "GET", Path: "/api/v1/users",
	})
	if err != nil || !decision.Allowed || len(decision.Roles) != 1 || decision.Roles[0] != "admin" {
		t.Fatalf("decision=%+v err=%v", decision, err)
	}
	roles, err := c.GetUserRoles(testContext(t), "u_1", userCtx)
	if err != nil || len(roles) != 1 || roles[0]["code"] != "admin" {
		t.Fatalf("roles=%+v err=%v", roles, err)
	}
	if err := c.AssignUserRoles(testContext(t), "u_1", AssignRolesRequest{RoleIDs: []string{"r_1"}}, userCtx); err != nil {
		t.Fatalf("assign user roles: %v", err)
	}
	permissions, err := c.GetUserPermissions(testContext(t), "u_1", userCtx)
	if err != nil || len(permissions) != 1 || permissions[0] != "user:read" {
		t.Fatalf("permissions=%+v err=%v", permissions, err)
	}
	listRoles, err := c.ListRoles(testContext(t), userCtx)
	if err != nil || len(listRoles) != 1 || listRoles[0]["id"] != "r_1" {
		t.Fatalf("list roles=%+v err=%v", listRoles, err)
	}
	createdRole, err := c.CreateRole(testContext(t), RoleRequest{OrgID: "org_1", Name: "Ops", Code: "ops"}, userCtx)
	if err != nil || createdRole["code"] != "ops" {
		t.Fatalf("created role=%+v err=%v", createdRole, err)
	}
	if _, err := c.GetRole(testContext(t), "r_1", userCtx); err != nil {
		t.Fatalf("get role: %v", err)
	}
	if _, err := c.UpdateRole(testContext(t), "r_1", RoleRequest{Name: "Admin Updated", Description: "desc"}, userCtx); err != nil {
		t.Fatalf("update role: %v", err)
	}
	if err := c.AssignRolePermissions(testContext(t), "r_1", AssignPermissionsRequest{PermissionIDs: []string{"p_1"}}, userCtx); err != nil {
		t.Fatalf("assign permissions: %v", err)
	}
	if err := c.DeleteRole(testContext(t), "r_1", userCtx); err != nil {
		t.Fatalf("delete role: %v", err)
	}
	tree, err := c.GetPermissionTree(testContext(t), userCtx)
	if err != nil || len(tree) != 1 || tree[0]["code"] != "user:read" {
		t.Fatalf("tree=%+v err=%v", tree, err)
	}
	apiPermissions, err := c.ListAPIPermissions(testContext(t), userCtx)
	if err != nil || len(apiPermissions) != 1 || apiPermissions[0]["method"] != "GET" {
		t.Fatalf("api permissions=%+v err=%v", apiPermissions, err)
	}
	createdAPI, err := c.CreateAPIPermission(testContext(t), APIPermissionRequest{Method: "POST", PathPattern: "/api/v1/x", PermissionCode: "x:create", Enabled: true}, userCtx)
	if err != nil || createdAPI["id"] != "api_2" {
		t.Fatalf("created api=%+v err=%v", createdAPI, err)
	}
	updatedAPI, err := c.UpdateAPIPermission(testContext(t), "api_1", APIPermissionRequest{Method: "PUT", PathPattern: "/api/v1/x", PermissionCode: "x:update", Enabled: true}, userCtx)
	if err != nil || updatedAPI["method"] != "PUT" {
		t.Fatalf("updated api=%+v err=%v", updatedAPI, err)
	}
	if err := c.DeleteAPIPermission(testContext(t), "api_1", userCtx); err != nil {
		t.Fatalf("delete api: %v", err)
	}
}

type stubIAMServer struct {
	iamv1.UnimplementedIAMServiceServer
}

func (s *stubIAMServer) CheckPermission(ctx context.Context, req *iamv1.CheckPermissionRequest) (*iamv1.CheckPermissionResponse, error) {
	if req.UserId != "u_1" || req.OrgId != "org_1" || req.Method != "GET" {
		return nil, errUnexpectedRequest
	}
	return &iamv1.CheckPermissionResponse{Allowed: true, Roles: []string{"admin"}}, nil
}

func (s *stubIAMServer) AssignUserRoles(ctx context.Context, req *iamv1.AssignUserRolesRequest) (*iamv1.Empty, error) {
	if req.UserId != "u_1" || len(req.RoleIds) != 1 || req.RoleIds[0] != "r_1" {
		return nil, errUnexpectedRequest
	}
	return &iamv1.Empty{}, nil
}

func (s *stubIAMServer) GetUserRoles(ctx context.Context, req *iamv1.GetUserRolesRequest) (*iamv1.GetUserRolesResponse, error) {
	return &iamv1.GetUserRolesResponse{Roles: []*iamv1.Role{role("r_1", "admin")}}, nil
}

func (s *stubIAMServer) GetUserPermissions(ctx context.Context, req *iamv1.GetUserPermissionsRequest) (*iamv1.GetUserPermissionsResponse, error) {
	return &iamv1.GetUserPermissionsResponse{Codes: []string{"user:read"}}, nil
}

func (s *stubIAMServer) ListRoles(ctx context.Context, req *iamv1.ListRolesRequest) (*iamv1.ListRolesResponse, error) {
	return &iamv1.ListRolesResponse{Roles: []*iamv1.Role{role("r_1", "admin")}}, nil
}

func (s *stubIAMServer) CreateRole(ctx context.Context, req *iamv1.CreateRoleRequest) (*iamv1.Role, error) {
	return role("r_2", req.Code), nil
}

func (s *stubIAMServer) GetRole(ctx context.Context, req *iamv1.GetRoleRequest) (*iamv1.Role, error) {
	return role(req.Id, "admin"), nil
}

func (s *stubIAMServer) UpdateRole(ctx context.Context, req *iamv1.UpdateRoleRequest) (*iamv1.Role, error) {
	return &iamv1.Role{Id: req.Id, Name: req.Name, Description: req.Description, Code: "admin"}, nil
}

func (s *stubIAMServer) DeleteRole(ctx context.Context, req *iamv1.DeleteRoleRequest) (*iamv1.Empty, error) {
	return &iamv1.Empty{}, nil
}

func (s *stubIAMServer) AssignPermissions(ctx context.Context, req *iamv1.AssignPermissionsRequest) (*iamv1.Empty, error) {
	if req.RoleId != "r_1" || len(req.PermissionIds) != 1 {
		return nil, errUnexpectedRequest
	}
	return &iamv1.Empty{}, nil
}

func (s *stubIAMServer) GetPermissionTree(ctx context.Context, req *iamv1.Empty) (*iamv1.PermissionTreeResponse, error) {
	return &iamv1.PermissionTreeResponse{Permissions: []*iamv1.Permission{{Id: "p_1", Code: "user:read", Name: "Read User", Resource: "user", Action: "read"}}}, nil
}

func (s *stubIAMServer) ListAPIPermissions(ctx context.Context, req *iamv1.ListAPIPermissionsRequest) (*iamv1.ListAPIPermissionsResponse, error) {
	return &iamv1.ListAPIPermissionsResponse{ApiPermissions: []*iamv1.APIPermission{{Id: "api_1", Method: "GET", PathPattern: "/api/v1/users", PermissionCode: "user:read", Enabled: true}}}, nil
}

func (s *stubIAMServer) CreateAPIPermission(ctx context.Context, req *iamv1.CreateAPIPermissionRequest) (*iamv1.APIPermission, error) {
	return &iamv1.APIPermission{Id: "api_2", Method: req.Method, PathPattern: req.PathPattern, PermissionCode: req.PermissionCode, Enabled: req.Enabled}, nil
}

func (s *stubIAMServer) UpdateAPIPermission(ctx context.Context, req *iamv1.UpdateAPIPermissionRequest) (*iamv1.APIPermission, error) {
	return &iamv1.APIPermission{Id: req.Id, Method: req.Method, PathPattern: req.PathPattern, PermissionCode: req.PermissionCode, Enabled: req.Enabled}, nil
}

func (s *stubIAMServer) DeleteAPIPermission(ctx context.Context, req *iamv1.DeleteAPIPermissionRequest) (*iamv1.Empty, error) {
	return &iamv1.Empty{}, nil
}

func role(id, code string) *iamv1.Role {
	return &iamv1.Role{Id: id, OrgId: "org_1", Name: code, Code: code, Description: "desc"}
}

func newIAMBufConn(t *testing.T, server iamv1.IAMServiceServer) *grpc.ClientConn {
	t.Helper()
	lis := bufconn.Listen(1024 * 1024)
	grpcServer := grpc.NewServer()
	iamv1.RegisterIAMServiceServer(grpcServer, server)
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			t.Logf("iam bufconn server stopped: %v", err)
		}
	}()
	t.Cleanup(grpcServer.Stop)
	conn, err := grpc.DialContext(context.Background(), "bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("dial bufconn: %v", err)
	}
	return conn
}
