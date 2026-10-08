package utils

import (
	"errors"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

// VerifyToken parses an HMAC-signed JWT and validates its "type" and "sub" claims.
// Pass an empty tokenType to skip the "type" claim check.
func VerifyToken(tokenStr, secret, tokenType string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}

	if tokenType != "" {
		jwtType, ok := claims["type"].(string)
		if !ok || jwtType != tokenType {
			return nil, errors.New("invalid token type")
		}
	}

	_, ok = claims["sub"].(string)
	if !ok {
		return nil, errors.New("invalid token sub")
	}

	return claims, nil
}
