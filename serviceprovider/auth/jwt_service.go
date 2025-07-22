package auth

import (
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/pkg/errors"
	"os"
	"time"
)

type JWTService interface {
	GenerateJWT(userID string, roles []string) (string, error)
	GenerateRefreshToken(userID string) (string, error)
	ParseJWT(tokenStr string) (string, []string, error)
	ParseRefreshToken(tokenStr string) (string, error)
}

type jwtService struct {
	jwtSecret          []byte
	refreshSecret      []byte
	tokenExpiry        time.Duration
	refreshTokenExpiry time.Duration
}

func NewJWTService() JWTService {
	return &jwtService{
		jwtSecret:          []byte(os.Getenv("SECRET_KEY")),
		refreshSecret:      []byte(os.Getenv("REFRESH_TOKEN")),
		tokenExpiry:        5 * time.Minute,
		refreshTokenExpiry: 7 * 24 * time.Hour,
	}
}

func (j *jwtService) GenerateJWT(userID string, roles []string) (string, error) {
	claims := jwt.MapClaims{
		"sub":   userID,
		"roles": roles,
		"typ":   "access",
		"exp":   time.Now().Add(j.tokenExpiry).Unix(),
		"iat":   time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.jwtSecret)
}

func (j *jwtService) GenerateRefreshToken(userID string) (string, error) {
	claims := jwt.MapClaims{
		"sub": userID,
		"typ": "refresh",
		"exp": time.Now().Add(j.refreshTokenExpiry).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.refreshSecret)
}

func (j *jwtService) ParseJWT(tokenStr string) (string, []string, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		return j.jwtSecret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))

	if err != nil || !token.Valid {
		return "", nil, fmt.Errorf("invalid or expired token: %w", err)
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", nil, errors.New("invalid token claims")
	}

	sub, ok := claims["sub"].(string)
	if !ok {
		return "", nil, errors.New("invalid 'sub' claim")
	}

	var roles []string
	if rolesClaim, ok := claims["roles"]; ok {
		if rolesSlice, ok := rolesClaim.([]interface{}); ok {
			for _, r := range rolesSlice {
				if roleStr, ok := r.(string); ok {
					roles = append(roles, roleStr)
				}
			}
		}
	}
	return sub, roles, nil
}

func (j *jwtService) ParseRefreshToken(tokenStr string) (string, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		return j.refreshSecret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))

	if err != nil || !token.Valid {
		return "", errors.New("invalid or expired refresh token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || claims["typ"] != "refresh" {
		return "", errors.New("invalid refresh token")
	}

	sub, ok := claims["sub"].(string)
	if !ok {
		return "", errors.New("invalid 'sub' claim")
	}
	return sub, nil
}
