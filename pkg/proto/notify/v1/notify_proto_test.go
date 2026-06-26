package notifyv1_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNotifyProtoCoversNotificationCenterPhaseOneContract(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("notify.proto"))
	if err != nil {
		t.Fatalf("read notify.proto: %v", err)
	}
	proto := string(data)

	requiredSnippets := []string{
		"package notify.v1;",
		"message NotificationChannel",
		"message NotificationTemplate",
		"message NotificationLog",
		"message ListChannelsRequest",
		"message CreateChannelRequest",
		"message UpdateChannelRequest",
		"message ListTemplatesRequest",
		"message CreateTemplateRequest",
		"message UpdateTemplateRequest",
		"message ListNotificationLogsRequest",
		"message SendNotificationRequest",
		"message SendNotificationResponse",
		"service NotifyService",
		"rpc ListChannels(",
		"rpc CreateChannel(",
		"rpc UpdateChannel(",
		"rpc DeleteChannel(",
		"rpc ListTemplates(",
		"rpc CreateTemplate(",
		"rpc UpdateTemplate(",
		"rpc DeleteTemplate(",
		"rpc ListNotificationLogs(",
		"rpc SendNotification(",
	}

	for _, snippet := range requiredSnippets {
		if !strings.Contains(proto, snippet) {
			t.Fatalf("notify.proto missing %q", snippet)
		}
	}
}
