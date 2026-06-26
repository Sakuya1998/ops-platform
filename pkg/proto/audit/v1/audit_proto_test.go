package auditv1_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAuditProtoCoversAuditLogQueryContract(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("audit.proto"))
	if err != nil {
		t.Fatalf("read audit.proto: %v", err)
	}
	proto := string(data)

	requiredSnippets := []string{
		"package audit.v1;",
		"message AuditLog",
		"message ListAuditLogsRequest",
		"message ListAuditLogsResponse",
		"message ListEventTypesRequest",
		"message ListEventTypesResponse",
		"message RecordAuditEventRequest",
		"message RecordAuditEventResponse",
		"service AuditService",
		"rpc ListAuditLogs(",
		"rpc ListEventTypes(",
		"rpc RecordAuditEvent(",
		"string reason_code",
		"string request_id",
	}

	for _, snippet := range requiredSnippets {
		if !strings.Contains(proto, snippet) {
			t.Fatalf("audit.proto missing %q", snippet)
		}
	}
}
