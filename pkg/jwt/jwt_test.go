package jwt

import (
	"testing"
	"time"
)

func TestGenerateAndValidate(t *testing.T) {
	m := NewManager("test-secret-key", 2, "ops-test")
	token, err := m.Generate("user-1", "org-1")
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if token == "" {
		t.Fatal("Token should not be empty")
	}

	claims, err := m.Validate(token)
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}
	if claims.UserID != "user-1" {
		t.Errorf("Expected user-1, got %s", claims.UserID)
	}
	if claims.OrgID != "org-1" {
		t.Errorf("Expected org-1, got %s", claims.OrgID)
	}
}

func TestGenerateWithJTI(t *testing.T) {
	m := NewManager("test-secret-key", 2, "ops-test")
	token, err := m.GenerateWithSession("user-1", "org-1", "session-1", "token-id-1")
	if err != nil {
		t.Fatalf("GenerateWithSession failed: %v", err)
	}
	claims, err := m.Validate(token)
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}
	if claims.ID != "token-id-1" {
		t.Errorf("Expected jti token-id-1, got %s", claims.ID)
	}
	if claims.SessionID != "session-1" {
		t.Errorf("Expected session_id session-1, got %s", claims.SessionID)
	}
}

func TestInvalidToken(t *testing.T) {
	m := NewManager("secret-a", 2, "test")
	_, err := m.Validate("invalid.jwt.token")
	if err != ErrTokenInvalid {
		t.Errorf("Expected ErrTokenInvalid, got %v", err)
	}
}

func TestWrongSecret(t *testing.T) {
	m1 := NewManager("secret-a", 2, "test")
	m2 := NewManager("secret-b", 2, "test")
	token, _ := m1.Generate("u1", "o1")
	_, err := m2.Validate(token)
	if err != ErrTokenInvalid {
		t.Errorf("Expected ErrTokenInvalid for wrong secret, got %v", err)
	}
}

func TestExpiredToken(t *testing.T) {
	m := NewManager("test-secret", 0, "test")
	token, err := m.Generate("u1", "o1")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	time.Sleep(100 * time.Millisecond)
	_, err = m.Validate(token)
	if err != ErrTokenExpired {
		t.Errorf("Expected ErrTokenExpired, got %v", err)
	}
}
