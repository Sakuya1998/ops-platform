package authv1_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAuthProtoCoversIdentityCenterPhaseOneContract(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("auth.proto"))
	if err != nil {
		t.Fatalf("read auth.proto: %v", err)
	}
	proto := string(data)

	requiredSnippets := []string{
		"message User",
		"message Organization",
		"message Session",
		"message AuthProvider",
		"message SystemConfig",
		"message MFASetupResponse",
		"message OIDCStatusResponse",
		"message ChangePasswordRequest",
		"message ResetUserPasswordRequest",
		"message ListUsersRequest",
		"message ListUsersResponse",
		"message CreateUserRequest",
		"message UpdateUserRequest",
		"message UpdateUserStatusRequest",
		"message ListOrganizationsResponse",
		"message UpdateSystemConfigRequest",
		"rpc VerifyToken(",
		"rpc GetCurrentUser(",
		"rpc ChangePassword(",
		"rpc SetupMFA(",
		"rpc ConfirmMFA(",
		"rpc DisableMFA(",
		"rpc RegenerateMFARecoveryCodes(",
		"rpc ListSessions(",
		"rpc RevokeSession(",
		"rpc RevokeOtherSessions(",
		"rpc OIDCStatus(",
		"rpc OIDCExchange(",
		"rpc ListUsers(",
		"rpc CreateUser(",
		"rpc GetUser(",
		"rpc UpdateUser(",
		"rpc UpdateUserStatus(",
		"rpc DeleteUser(",
		"rpc ResetUserPassword(",
		"rpc ListOrganizations(",
		"rpc CreateOrganization(",
		"rpc UpdateOrganization(",
		"rpc GetSystemConfig(",
		"rpc UpdateSystemConfig(",
	}

	for _, snippet := range requiredSnippets {
		if !strings.Contains(proto, snippet) {
			t.Fatalf("auth.proto missing %q", snippet)
		}
	}
}
