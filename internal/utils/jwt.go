package utils

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrInvalidToken = errors.New("invalid or expired token")
)

// Claims kustom untuk access token. jti dipakai untuk blacklist saat logout.
type Claims struct {
	AdminID string `json:"admin_id"`
	Email   string `json:"email"`
	RoleID  *int   `json:"role_id,omitempty"`
	Type    string `json:"type"` // "access" | "refresh"
	jwt.RegisteredClaims
}

type JWTManager struct {
	accessSecret  []byte
	refreshSecret []byte
	accessTTL     time.Duration
	refreshTTL    time.Duration
	issuer        string
}

func NewJWTManager(accessSecret, refreshSecret string, accessTTL, refreshTTL time.Duration, issuer string) *JWTManager {
	return &JWTManager{
		accessSecret:  []byte(accessSecret),
		refreshSecret: []byte(refreshSecret),
		accessTTL:     accessTTL,
		refreshTTL:    refreshTTL,
		issuer:        issuer,
	}
}

// GenerateAccessToken mengembalikan token, jti (dipakai key blacklist), dan error.
func (m *JWTManager) GenerateAccessToken(adminID, email string, roleID *int) (string, string, error) {
	jti := uuid.NewString()
	now := time.Now()

	claims := Claims{
		AdminID: adminID,
		Email:   email,
		RoleID:  roleID,
		Type:    "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti,
			Issuer:    m.issuer,
			Subject:   adminID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.accessTTL)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.accessSecret)
	return signed, jti, err
}

// GenerateRefreshToken mengembalikan token, jti (key penyimpanan di Redis), dan error.
func (m *JWTManager) GenerateRefreshToken(adminID string) (string, string, error) {
	jti := uuid.NewString()
	now := time.Now()

	claims := Claims{
		AdminID: adminID,
		Type:    "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti,
			Issuer:    m.issuer,
			Subject:   adminID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.refreshTTL)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.refreshSecret)
	return signed, jti, err
}

func (m *JWTManager) ParseAccessToken(tokenStr string) (*Claims, error) {
	return m.parse(tokenStr, m.accessSecret, "access")
}

func (m *JWTManager) ParseRefreshToken(tokenStr string) (*Claims, error) {
	return m.parse(tokenStr, m.refreshSecret, "refresh")
}

func (m *JWTManager) parse(tokenStr string, secret []byte, expectType string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return secret, nil
	})
	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}
	if claims.Type != expectType {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// AccessTTL & RefreshTTL diekspos agar handler bisa mengembalikan expires_in.
func (m *JWTManager) AccessTTL() time.Duration  { return m.accessTTL }
func (m *JWTManager) RefreshTTL() time.Duration { return m.refreshTTL }
