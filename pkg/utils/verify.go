package utils

import (
	"errors"

	"	github.com/golang-jwt/jwt/v5"
)

func VerifyToken(tokenStr, secret, tokenType string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenStr, func(_ *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})

	if err != nil || !token.Valid {
		return nil, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}

	jwtType, ok := claims["type"].(string)
	if !ok || jwtType != tokenType {
		return nil, errors.New("invalid token type")
	}

	_, ok = claims["sub"].(string)
	if !ok {
		return nil, errors.New("invalid token sub")
	}

	return claims, nil
}
