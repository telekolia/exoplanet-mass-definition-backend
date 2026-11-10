package ds

import (
	"app/internal/app/role"

	"github.com/golang-jwt/jwt/v5"
)

type JWTClaims struct {
	jwt.RegisteredClaims
	UserID uint      `json:"user_id"`
	Role   role.Role `json:"role"`
}
