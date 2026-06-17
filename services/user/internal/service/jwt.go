package service

import (
	"crypto/rsa"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/ridego/services/user/internal/models"
)

type Claims struct {
	jwt.RegisteredClaims
	UserID string      `json:"uid"`
	Role   models.Role `json:"role"`
	Email  string      `json:"email"`
}

type JWTService struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
}

func NewJWT(priv *rsa.PrivateKey, pub *rsa.PublicKey) *JWTService {
	return &JWTService{privateKey: priv, publicKey: pub}
}

// Sign creates a signed RS256 JWT for the given user.
func (j *JWTService) Sign(u *models.User, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
		UserID: u.ID.String(),
		Role:   u.Role,
		Email:  u.Email,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(j.privateKey)
}

// Verify parses and validates a JWT, returning the embedded claims.
func (j *JWTService) Verify(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{},
		func(t *jwt.Token) (interface{}, error) { // we can use any in place of interface{}
			if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return j.publicKey, nil
		},
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
	)

	if err != nil || !token.Valid {
		return nil, errors.Join(ErrInvalidToken, err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, ErrInvalidToken
	}
	return claims, nil

}
