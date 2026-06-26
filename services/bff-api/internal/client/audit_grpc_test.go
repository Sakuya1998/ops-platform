package client

import (
	"context"
	"net"
	"testing"
	"time"

	auditv1 "github.com/Sakuya1998/ops-platform/pkg/proto/audit/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestAuditGRPCClientListLogsAndEventTypes(t *testing.T) {
	conn := newAuditBufConn(t, &stubAuditServer{})
	defer conn.Close()

	c := NewAuditGRPCClient(conn)
	logs, total, err := c.ListLogs(testContext(t), AuditLogQuery{
		OrgID:     "org_1",
		EventType: "user.login",
		Page:      2,
		PageSize:  10,
	}, UserContext{OrgID: "org_1"})
	if err != nil {
		t.Fatalf("list logs: %v", err)
	}
	if total != 1 || len(logs) != 1 || logs[0]["event_type"] != "user.login" || logs[0]["id"] != "log_1" {
		t.Fatalf("unexpected logs total=%d logs=%+v", total, logs)
	}
	types, err := c.ListEventTypes(testContext(t), UserContext{})
	if err != nil || len(types) != 2 || types[0] != "user.login" {
		t.Fatalf("types=%+v err=%v", types, err)
	}
}

type stubAuditServer struct {
	auditv1.UnimplementedAuditServiceServer
}

func (s *stubAuditServer) ListAuditLogs(ctx context.Context, req *auditv1.ListAuditLogsRequest) (*auditv1.ListAuditLogsResponse, error) {
	if req.OrgId != "org_1" || req.EventType != "user.login" || req.Page != 2 || req.PageSize != 10 {
		return nil, errUnexpectedRequest
	}
	return &auditv1.ListAuditLogsResponse{
		Total: 1,
		Logs: []*auditv1.AuditLog{{
			Id: "log_1", OrgId: "org_1", UserId: "u_1", Username: "admin",
			EventType: "user.login", Action: "login", ResourceType: "auth", Detail: "ok",
			CreatedAt: timestamppb.New(time.Unix(100, 0)),
		}},
	}, nil
}

func (s *stubAuditServer) ListEventTypes(ctx context.Context, req *auditv1.ListEventTypesRequest) (*auditv1.ListEventTypesResponse, error) {
	return &auditv1.ListEventTypesResponse{EventTypes: []string{"user.login", "role.created"}}, nil
}

func newAuditBufConn(t *testing.T, server auditv1.AuditServiceServer) *grpc.ClientConn {
	t.Helper()
	lis := bufconn.Listen(1024 * 1024)
	grpcServer := grpc.NewServer()
	auditv1.RegisterAuditServiceServer(grpcServer, server)
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			t.Logf("audit bufconn server stopped: %v", err)
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
