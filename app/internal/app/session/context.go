package session

import (
	"app/internal/app/ds"

	"github.com/gin-gonic/gin"
)

const claimsKey = "auth.claims"

func SetClaims(ctx *gin.Context, claims *ds.JWTClaims) {
	ctx.Set(claimsKey, claims)
}

func Claims(ctx *gin.Context) *ds.JWTClaims {
	value, _ := ctx.Get(claimsKey)
	claims, _ := value.(*ds.JWTClaims)
	return claims
}

func CurrentUserID(ctx *gin.Context) uint {
	if claims := Claims(ctx); claims != nil {
		return claims.UserID
	}
	return 0
}
