package app

import (
	"app/internal/app/role"
	"app/internal/app/session"
	"net/http"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func (a *Application) WithAuthCheck(assignedRoles ...role.Role) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		header := ctx.GetHeader("Authorization")
		if header == "" && slices.Contains(assignedRoles, role.Guest) {
			ctx.Next()
			return
		}
		scheme, raw, ok := strings.Cut(header, " ")
		if !ok || !strings.EqualFold(scheme, "Bearer") || raw == "" {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		claims, err := a.tokens.Parse(raw)
		if err != nil {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		revoked, err := a.blacklist.CheckJWTInBlacklist(ctx.Request.Context(), claims.ID)
		if err != nil {
			logrus.Error(err)
			ctx.AbortWithStatus(http.StatusServiceUnavailable)
			return
		}
		if revoked {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		if claims.Role == role.Guest || !slices.Contains(assignedRoles, claims.Role) {
			ctx.AbortWithStatus(http.StatusForbidden)
			return
		}
		session.SetClaims(ctx, claims)
		ctx.Next()
	}
}
