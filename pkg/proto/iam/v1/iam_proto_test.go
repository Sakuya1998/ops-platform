package iamv1

import (
	"os"
	"strings"
	"testing"
)

func TestIAMProtoMatchesAuthorizationBoundary(t *testing.T) {
	contentBytes, err := os.ReadFile("iam.proto")
	if err != nil {
		t.Fatalf("read iam.proto: %v", err)
	}
	content := string(contentBytes)

	forbidden := []string{
		"rpc CreateUser(",
		"rpc GetUser(",
		"rpc ListUsers(",
		"rpc UpdateUser(",
		"rpc DeleteUser(",
		"rpc CreateOrganization(",
		"rpc ListOrganizations(",
		"rpc UpdateOrganization(",
	}
	for _, snippet := range forbidden {
		if strings.Contains(content, snippet) {
			t.Fatalf("iam proto should not own identity-domain rpc: %s", snippet)
		}
	}

	required := []string{
		"rpc AssignUserRoles(",
		"rpc GetUserRoles(",
		"rpc GetUserPermissions(",
		"rpc CheckPermission(",
		"rpc BatchCheckPermission(",
		"rpc ListAPIPermissions(",
		"rpc CreateAPIPermission(",
		"rpc UpdateAPIPermission(",
		"rpc DeleteAPIPermission(",
		"rpc CreateResource(",
		"rpc CreatePolicy(",
		"rpc BindPolicy(",
	}
	for _, snippet := range required {
		if !strings.Contains(content, snippet) {
			t.Fatalf("iam proto missing authorization-domain rpc: %s", snippet)
		}
	}
}
