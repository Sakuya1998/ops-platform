package jwt

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrTokenExpired = errors.New("token expired")
	ErrTokenInvalid = errors.New("token invalid")
)

type Claims struct {
	UserID    string `json:"user_id"`
	OrgID     string `json:"org_id"`
	SessionID string `json:"session_id,omitempty"`
	jwt.RegisteredClaims
}

type Manager struct {
	secret     []byte
	expireHour int
	issuer     string
}

func NewManager(secret string, expireHour int, issuer string) *Manager {
	return &Manager{
		secret:     []byte(secret),
		expireHour: expireHour,
		issuer:     issuer,
	}
}

func (m *Manager) Generate(userID, orgID string) (string, error) {
	return m.GenerateWithJTI(userID, orgID, "")
}

func (m *Manager) GenerateWithJTI(userID, orgID, jti string) (string, error) {
	return m.GenerateWithSession(userID, orgID, "", jti)
}

func (m *Manager) GenerateWithSession(userID, orgID, sessionID, jti string) (string, error) {
	now := time.Now()
	claims := &Claims{
		UserID:    userID,
		OrgID:     orgID,
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(m.expireHour) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    m.issuer,
			Subject:   userID,
			ID:        jti,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

func (m *Manager) Validate(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return m.secret, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrTokenInvalid
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrTokenInvalid
	}
	return claims, nil
}
