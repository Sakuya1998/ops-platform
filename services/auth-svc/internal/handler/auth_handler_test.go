package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/Sakuya1998/ops-platform/services/auth-svc/internal/repository"
	"github.com/Sakuya1998/ops-platform/services/auth-svc/internal/service"
	"github.com/Sakuya1998/ops-platform/pkg/cache"
	sharedcfg "github.com/Sakuya1998/ops-platform/pkg/config"
	pkgjwt "github.com/Sakuya1998/ops-platform/pkg/jwt"
)

func TestOIDCStatus_Disabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewAuthHandler(nil, nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/api/v1/auth/oidc/status", nil)
	h.OIDCStatus(c)

	var resp struct {
		Code    int                    `json:"code"`
		Message string                 `json:"message"`
		Data    map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if resp.Code != 0 {
		t.Errorf("Expected code 0, got %d", resp.Code)
	}
	if resp.Data["enabled"] != false {
		t.Errorf("Expected enabled=false, got %v", resp.Data["enabled"])
	}
}

func TestLogoutFallsBackToBearerTokenUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer sqlDB.Close()
	db := openMockDB(t, sqlDB)

	userID := uuid.MustParse("00000000-0000-0000-0000-000000000010")
	orgID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	manager := pkgjwt.NewManager("test-secret", 2, "ops-test")
	token, err := manager.GenerateWithSession(userID.String(), orgID.String(), uuid.New().String(), "jti-1")
	if err != nil {
		t.Fatalf("GenerateWithSession: %v", err)
	}

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "sessions"`).
		WithArgs(sqlmock.AnyArg(), "logout", "revoked", sqlmock.AnyArg(), userID, "active").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE "refresh_tokens"`).
		WithArgs(true, sqlmock.AnyArg(), "logout", userID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	authSvc := service.NewAuthService(repository.NewUserRepository(db), nil, nil, nil, sharedcfg.JWTConfig{
		Secret: "test-secret", ExpireHour: 2, Issuer: "ops-test",
	}, nil, nil, nil)
	h := NewAuthHandler(authSvc, nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	c.Request.Header.Set("Authorization", "Bearer "+token)

	h.Logout(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestOIDCExchangeReturnsFullLoginResult(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oidcSvc := service.NewOIDCServiceWithStateCache(nil, nil, nil, nil, nil, cache.New(cache.Options{}))
	code, err := oidcSvc.IssueLoginCode(httptest.NewRequest(http.MethodPost, "/", nil).Context(), &service.LoginResult{
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		ExpiresIn:    7200,
		SessionID:    "session-1",
		JTI:          "jti-1",
		UserID:       "user-1",
		OrgID:        "org-1",
		Username:     "oidc-user",
		DisplayName:  "OIDC User",
		Email:        "oidc@example.com",
		Roles:        []string{"admin"},
		MFAEnabled:   true,
	})
	if err != nil {
		t.Fatalf("IssueLoginCode: %v", err)
	}

	h := NewAuthHandler(nil, oidcSvc)
	body := bytes.NewBufferString(`{"code":"` + code + `"}`)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodPost, "/api/v1/auth/oidc/exchange", body)
	c.Request.Header.Set("Content-Type", "application/json")

	h.OIDCExchange(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Code int `json:"code"`
		Data struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
			SessionID    string `json:"session_id"`
			User         struct {
				UserID string   `json:"user_id"`
				Roles  []string `json:"roles"`
			} `json:"user"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if resp.Code != 0 || resp.Data.AccessToken != "access-token" || resp.Data.RefreshToken != "refresh-token" {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if resp.Data.SessionID != "session-1" || resp.Data.User.UserID != "user-1" || len(resp.Data.User.Roles) != 1 {
		t.Fatalf("unexpected user/session response: %+v", resp.Data)
	}
}
