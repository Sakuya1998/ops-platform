package client

import (
	"context"
	"net"
	"testing"
	"time"

	notifyv1 "github.com/Sakuya1998/ops-platform/pkg/proto/notify/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestNotifyGRPCClientChannelsTemplatesAndLogs(t *testing.T) {
	conn := newNotifyBufConn(t, &stubNotifyServer{})
	defer conn.Close()

	c := NewNotifyGRPCClient(conn)
	userCtx := UserContext{OrgID: "org_1"}
	channels, err := c.ListChannels(testContext(t), userCtx)
	if err != nil || len(channels) != 1 || channels[0]["id"] != "ch_1" {
		t.Fatalf("channels=%+v err=%v", channels, err)
	}
	channel, err := c.CreateChannel(testContext(t), NotificationChannelRequest{
		Name: "Webhook", ChannelType: "webhook", Config: map[string]any{"url": "https://example.com"},
	}, userCtx)
	if err != nil || channel["id"] != "ch_2" {
		t.Fatalf("channel=%+v err=%v", channel, err)
	}
	if _, err := c.UpdateChannel(testContext(t), "ch_1", NotificationChannelRequest{Name: "Email", ChannelType: "email", Config: map[string]any{"host": "smtp"}}, userCtx); err != nil {
		t.Fatalf("update channel: %v", err)
	}
	if err := c.DeleteChannel(testContext(t), "ch_1", userCtx); err != nil {
		t.Fatalf("delete channel: %v", err)
	}
	templates, err := c.ListTemplates(testContext(t), userCtx)
	if err != nil || len(templates) != 1 || templates[0]["id"] != "tpl_1" {
		t.Fatalf("templates=%+v err=%v", templates, err)
	}
	if _, err := c.CreateTemplate(testContext(t), NotificationTemplateRequest{Name: "Deploy", ChannelType: "webhook", BodyTemplate: "body"}, userCtx); err != nil {
		t.Fatalf("create template: %v", err)
	}
	if _, err := c.UpdateTemplate(testContext(t), "tpl_1", NotificationTemplateRequest{Name: "Alert", ChannelType: "email", BodyTemplate: "body"}, userCtx); err != nil {
		t.Fatalf("update template: %v", err)
	}
	if err := c.DeleteTemplate(testContext(t), "tpl_1", userCtx); err != nil {
		t.Fatalf("delete template: %v", err)
	}
	logs, total, err := c.ListNotificationLogs(testContext(t), NotificationLogQuery{Status: "success", Page: 1, PageSize: 20}, userCtx)
	if err != nil || total != 1 || len(logs) != 1 || logs[0]["id"] != "log_1" {
		t.Fatalf("logs=%+v total=%d err=%v", logs, total, err)
	}
}

type stubNotifyServer struct {
	notifyv1.UnimplementedNotifyServiceServer
}

func (s *stubNotifyServer) ListChannels(ctx context.Context, req *notifyv1.ListChannelsRequest) (*notifyv1.ListChannelsResponse, error) {
	if req.OrgId != "org_1" {
		return nil, errUnexpectedRequest
	}
	return &notifyv1.ListChannelsResponse{Channels: []*notifyv1.NotificationChannel{{Id: "ch_1", OrgId: "org_1", Name: "Email", ChannelType: "email", IsEnabled: true, CreatedAt: timestamppb.Now()}}}, nil
}

func (s *stubNotifyServer) CreateChannel(ctx context.Context, req *notifyv1.CreateChannelRequest) (*notifyv1.NotificationChannel, error) {
	if req.OrgId != "org_1" || req.ChannelType != "webhook" || req.ConfigJson == "" {
		return nil, errUnexpectedRequest
	}
	return &notifyv1.NotificationChannel{Id: "ch_2", OrgId: req.OrgId, Name: req.Name, ChannelType: req.ChannelType, ConfigJson: req.ConfigJson, IsEnabled: true}, nil
}

func (s *stubNotifyServer) UpdateChannel(ctx context.Context, req *notifyv1.UpdateChannelRequest) (*notifyv1.NotificationChannel, error) {
	if req.Id != "ch_1" || req.ConfigJson == "" {
		return nil, errUnexpectedRequest
	}
	return &notifyv1.NotificationChannel{Id: req.Id, Name: req.Name, ChannelType: req.ChannelType, ConfigJson: req.ConfigJson}, nil
}

func (s *stubNotifyServer) DeleteChannel(ctx context.Context, req *notifyv1.DeleteChannelRequest) (*notifyv1.Empty, error) {
	if req.Id != "ch_1" {
		return nil, errUnexpectedRequest
	}
	return &notifyv1.Empty{}, nil
}

func (s *stubNotifyServer) ListTemplates(ctx context.Context, req *notifyv1.ListTemplatesRequest) (*notifyv1.ListTemplatesResponse, error) {
	return &notifyv1.ListTemplatesResponse{Templates: []*notifyv1.NotificationTemplate{{Id: "tpl_1", Name: "Alert", ChannelType: "email", CreatedAt: timestamppb.Now()}}}, nil
}

func (s *stubNotifyServer) CreateTemplate(ctx context.Context, req *notifyv1.CreateTemplateRequest) (*notifyv1.NotificationTemplate, error) {
	if req.Name == "" || req.BodyTemplate == "" {
		return nil, errUnexpectedRequest
	}
	return &notifyv1.NotificationTemplate{Id: "tpl_2", Name: req.Name, ChannelType: req.ChannelType, BodyTemplate: req.BodyTemplate}, nil
}

func (s *stubNotifyServer) UpdateTemplate(ctx context.Context, req *notifyv1.UpdateTemplateRequest) (*notifyv1.NotificationTemplate, error) {
	if req.Id != "tpl_1" {
		return nil, errUnexpectedRequest
	}
	return &notifyv1.NotificationTemplate{Id: req.Id, Name: req.Name, ChannelType: req.ChannelType, BodyTemplate: req.BodyTemplate}, nil
}

func (s *stubNotifyServer) DeleteTemplate(ctx context.Context, req *notifyv1.DeleteTemplateRequest) (*notifyv1.Empty, error) {
	if req.Id != "tpl_1" {
		return nil, errUnexpectedRequest
	}
	return &notifyv1.Empty{}, nil
}

func (s *stubNotifyServer) ListNotificationLogs(ctx context.Context, req *notifyv1.ListNotificationLogsRequest) (*notifyv1.ListNotificationLogsResponse, error) {
	if req.Status != "success" || req.Page != 1 || req.PageSize != 20 {
		return nil, errUnexpectedRequest
	}
	return &notifyv1.ListNotificationLogsResponse{
		Total: 1,
		Logs:  []*notifyv1.NotificationLog{{Id: "log_1", Status: "success", CreatedAt: timestamppb.New(time.Unix(100, 0))}},
	}, nil
}

func newNotifyBufConn(t *testing.T, server notifyv1.NotifyServiceServer) *grpc.ClientConn {
	t.Helper()
	lis := bufconn.Listen(1024 * 1024)
	grpcServer := grpc.NewServer()
	notifyv1.RegisterNotifyServiceServer(grpcServer, server)
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			t.Logf("notify bufconn server stopped: %v", err)
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
