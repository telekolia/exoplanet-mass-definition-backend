package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const userIDKey = "auth.user_id"
const claimsKey = "auth.claims"

func currentUserID(ctx *gin.Context) uint {
	id, _ := ctx.Get(userIDKey)
	userID, _ := id.(uint)
	return userID
}

func (h *Handler) authenticate(required bool) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		header := ctx.GetHeader("Authorization")
		if header == "" && !required {
			return
		}
		scheme, raw, ok := strings.Cut(header, " ")
		if !ok || !strings.EqualFold(scheme, "Bearer") || raw == "" {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		userID, claims, err := h.tokens.Parse(raw)
		if err != nil {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		revoked, err := h.tokens.Revoked(ctx.Request.Context(), claims.ID)
		if err != nil {
			h.errorHandler(ctx, http.StatusServiceUnavailable, err)
			ctx.Abort()
			return
		}
		if revoked {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		ctx.Set(userIDKey, userID)
		ctx.Set(claimsKey, claims)
	}
}

func tokenClaims(ctx *gin.Context) *jwt.RegisteredClaims {
	value, _ := ctx.Get(claimsKey)
	claims, _ := value.(*jwt.RegisteredClaims)
	return claims
}
