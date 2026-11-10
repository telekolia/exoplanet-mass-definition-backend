package handler

import (
	_ "app/docs"
	"net/http"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func (h *Handler) RegisterSwagger(router *gin.Engine) {
	router.GET("/swagger", func(ctx *gin.Context) {
		ctx.Redirect(http.StatusFound, "/swagger/index.html")
	})
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}
