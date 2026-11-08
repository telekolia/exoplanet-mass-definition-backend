package handler

import (
	"app/internal/app/ds"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterUserAPI(ctx *gin.Context) {
	var request ds.RegisterUserRequest
	if !h.bindJSON(ctx, &request) {
		return
	}
	request.Login = strings.TrimSpace(request.Login)
	if request.Login == "" {
		ctx.Status(http.StatusBadRequest)
		return
	}
	user, err := h.Repository.RegisterUser(ctx.Request.Context(), request.Login, request.Password)
	if err != nil {
		h.repositoryError(ctx, err)
		return
	}
	ctx.JSON(http.StatusCreated, user)
}

func (h *Handler) LoginUserAPI(ctx *gin.Context) {
	ctx.Status(http.StatusNotImplemented)
}

func (h *Handler) LogoutUserAPI(ctx *gin.Context) {
	ctx.Status(http.StatusNotImplemented)
}
