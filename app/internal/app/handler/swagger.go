package handler

import (
	_ "embed"
	"net/http"

	"github.com/gin-gonic/gin"
)

//go:embed swagger/swagger.json
var swaggerSpec []byte

//go:embed swagger/index.html
var swaggerPage []byte

func (h *Handler) RegisterSwagger(router *gin.Engine) {
	router.GET("/swagger", func(ctx *gin.Context) {
		ctx.Redirect(http.StatusFound, "/swagger/index.html")
	})
	router.GET("/swagger/index.html", func(ctx *gin.Context) {
		ctx.Data(http.StatusOK, "text/html; charset=utf-8", swaggerPage)
	})
	router.GET("/swagger/doc.json", func(ctx *gin.Context) {
		ctx.Data(http.StatusOK, "application/json; charset=utf-8", swaggerSpec)
	})
}
