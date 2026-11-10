package handler

import (
	"app/internal/app/ds"
	"app/internal/app/session"
	"crypto/subtle"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// @Summary Регистрация пользователя
// @Tags Пользователи
// @Accept json
// @Produce json
// @Param body body ds.RegisterUserRequest true "Логин и пароль"
// @Success 201 {object} ds.User
// @Failure 400
// @Failure 409
// @Failure 413
// @Failure 415
// @Failure 500
// @Router /users [post]
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

// @Summary Получение JWT по логину и паролю
// @Tags Пользователи
// @Accept json
// @Produce json
// @Param body body ds.LoginRequest true "Логин и пароль"
// @Success 200 {object} session.LoginResponse
// @Failure 400
// @Failure 401
// @Failure 413
// @Failure 415
// @Failure 500
// @Router /users/login [post]
func (h *Handler) LoginUserAPI(ctx *gin.Context) {
	var request ds.LoginRequest
	if !h.bindJSON(ctx, &request) {
		return
	}
	user, err := h.Repository.GetUserByLogin(ctx.Request.Context(), strings.TrimSpace(request.Login))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		ctx.Status(http.StatusUnauthorized)
		return
	}
	if err != nil {
		h.repositoryError(ctx, err)
		return
	}
	if subtle.ConstantTimeCompare([]byte(user.Password), []byte(request.Password)) != 1 {
		ctx.Status(http.StatusUnauthorized)
		return
	}
	token, err := h.tokens.Issue(user.ID)
	if err != nil {
		h.repositoryError(ctx, err)
		return
	}
	ctx.Header("Cache-Control", "no-store")
	ctx.JSON(http.StatusOK, token)
}

// @Summary Отзыв текущего JWT
// @Tags Пользователи
// @Security BearerAuth
// @Success 200
// @Failure 401
// @Failure 403
// @Failure 503
// @Router /users/logout [post]
func (h *Handler) LogoutUserAPI(ctx *gin.Context) {
	claims := session.Claims(ctx)
	if claims == nil {
		ctx.Status(http.StatusUnauthorized)
		return
	}
	if err := h.blacklist.WriteJWTToBlacklist(ctx.Request.Context(), claims.ID, claims.ExpiresAt.Time); err != nil {
		h.errorHandler(ctx, http.StatusServiceUnavailable, err)
		return
	}
	ctx.Status(http.StatusOK)
}
