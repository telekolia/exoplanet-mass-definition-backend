package handler

import (
	"app/internal/app/ds"
	"app/internal/app/media"
	"app/internal/app/repository"
	"app/internal/app/session"
	"errors"
	"fmt"
	"math"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Handler struct {
	Repository *repository.Repository
}

func NewHandler(r *repository.Repository) *Handler {
	return &Handler{Repository: r}
}

func (h *Handler) RegisterHandler(router *gin.Engine) {
	binding.EnableDecoderDisallowUnknownFields = true
	router.HandleMethodNotAllowed = true
	api := router.Group("/api")
	api.GET("/telescopes", h.GetTelescopesAPI)
	api.GET("/telescopes/feed", h.GetTelescopeFeedAPI)
	api.GET("/telescopes/draft", h.GetTelescopeDraftAPI)
	api.POST("/telescopes", h.CreateTelescopeDraftAPI)
	api.PUT("/telescopes/:id", h.PublishTelescopeAPI)
	api.DELETE("/telescopes/:id", h.DeleteTelescopeAPI)
	api.POST("/telescopes/:id/likes", h.LikeTelescopeAPI)
	api.POST("/users", h.RegisterUserAPI)
	api.POST("/users/login", h.LoginUserAPI)
	api.POST("/users/logout", h.LogoutUserAPI)
	router.NoRoute(func(ctx *gin.Context) {
		h.errorHandler(ctx, http.StatusNotFound, errors.New("маршрут не найден"))
	})
	router.NoMethod(func(ctx *gin.Context) {
		h.errorHandler(ctx, http.StatusMethodNotAllowed, errors.New("метод не поддерживается"))
	})
}

func (h *Handler) RegisterStatic(router *gin.Engine) {
	router.Static("/static", "./resources")
}

func (h *Handler) errorHandler(ctx *gin.Context, code int, err error) {
	if code >= 500 {
		logrus.Error(err)
	}
	ctx.Status(code)
}

func (h *Handler) repositoryError(ctx *gin.Context, err error) {
	code := http.StatusInternalServerError
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		code = http.StatusNotFound
	case errors.Is(err, repository.ErrForbidden):
		code = http.StatusForbidden
	case errors.Is(err, repository.ErrConflict), errors.Is(err, gorm.ErrDuplicatedKey):
		code = http.StatusConflict
	}
	h.errorHandler(ctx, code, err)
}

func (h *Handler) requestID(ctx *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil || id == 0 {
		h.errorHandler(ctx, http.StatusBadRequest, errors.New("ID должен быть положительным целым числом"))
		return 0, false
	}
	return uint(id), true
}

func (h *Handler) bindJSON(ctx *gin.Context, request interface{}) bool {
	if ctx.ContentType() != "application/json" {
		h.errorHandler(ctx, http.StatusUnsupportedMediaType, errors.New("ожидается application/json"))
		return false
	}
	ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, 1<<20)
	if err := ctx.ShouldBindJSON(request); err != nil {
		code := http.StatusBadRequest
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			code = http.StatusRequestEntityTooLarge
		}
		h.errorHandler(ctx, code, fmt.Errorf("некорректные поля запроса: %w", err))
		return false
	}
	return true
}

func (h *Handler) GetTelescopesAPI(ctx *gin.Context) {
	minLatitude, maxLatitude := -90.0, 90.0
	for _, bound := range []struct {
		name  string
		value *float64
	}{{"min", &minLatitude}, {"max", &maxLatitude}} {
		if values, exists := ctx.Request.URL.Query()[bound.name]; exists {
			if len(values) != 1 || values[0] == "" {
				h.errorHandler(ctx, http.StatusBadRequest, errors.New("каждая граница широты должна быть указана один раз и не быть пустой"))
				return
			}
			parsed, err := strconv.ParseFloat(values[0], 64)
			if err != nil || math.IsNaN(parsed) || parsed < -90 || parsed > 90 {
				h.errorHandler(ctx, http.StatusBadRequest, errors.New("границы широты должны быть числами от -90 до 90"))
				return
			}
			*bound.value = parsed
		}
	}
	if minLatitude > maxLatitude {
		h.errorHandler(ctx, http.StatusBadRequest, errors.New("min не может быть больше max"))
		return
	}
	telescopes, err := h.Repository.GetTelescopes(ctx.Request.Context(), minLatitude, maxLatitude)
	if err != nil {
		h.repositoryError(ctx, err)
		return
	}
	ids := make([]uint, 0, len(telescopes))
	for _, telescope := range telescopes {
		ids = append(ids, telescope.ID)
	}
	counts, err := h.Repository.GetTelescopeLikesCounts(ctx.Request.Context(), ids)
	if err != nil {
		h.repositoryError(ctx, err)
		return
	}
	user := session.GetCurrentUser()
	data := make([]ds.TelescopeSerializer, 0, len(telescopes))
	for _, telescope := range telescopes {
		item := h.serializeTelescope(telescope, user.ID, counts[telescope.ID])
		data = append(data, item)
	}
	ctx.JSON(http.StatusOK, data)
}

func (h *Handler) GetTelescopeFeedAPI(ctx *gin.Context) {
	var id uint
	if values, exists := ctx.Request.URL.Query()["id"]; exists {
		if len(values) != 1 {
			h.errorHandler(ctx, http.StatusBadRequest, errors.New("ID должен быть положительным целым числом"))
			return
		}
		parsed, err := strconv.ParseUint(values[0], 10, 32)
		if err != nil || parsed == 0 {
			h.errorHandler(ctx, http.StatusBadRequest, errors.New("ID должен быть положительным целым числом"))
			return
		}
		id = uint(parsed)
	}
	next := false
	if values, exists := ctx.Request.URL.Query()["next"]; exists {
		if id == 0 || len(values) != 1 || (values[0] != "true" && values[0] != "false") {
			h.errorHandler(ctx, http.StatusBadRequest, errors.New("next требует параметр id и значение true или false"))
			return
		}
		next = values[0] == "true"
	}
	telescope, err := h.Repository.GetTelescopeFeedItem(ctx.Request.Context(), id, next)
	if err != nil {
		h.repositoryError(ctx, err)
		return
	}
	user := session.GetCurrentUser()
	count, err := h.Repository.GetTelescopeLikesCount(ctx.Request.Context(), telescope.ID)
	if err != nil {
		h.repositoryError(ctx, err)
		return
	}
	liked, err := h.Repository.IsTelescopeLiked(ctx.Request.Context(), telescope.ID, user.ID)
	if err != nil {
		h.repositoryError(ctx, err)
		return
	}
	item := h.serializeTelescope(telescope, user.ID, count)
	full := ds.FullTelescopeSerializer{TelescopeSerializer: item}
	if liked {
		full.IsLiked = 1
	}
	ctx.JSON(http.StatusOK, full)
}

func (h *Handler) GetTelescopeDraftAPI(ctx *gin.Context) {
	user := session.GetCurrentUser()
	telescope, err := h.Repository.GetTelescopeDraft(ctx.Request.Context(), user.ID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		ctx.JSON(http.StatusOK, nil)
		return
	}
	if err != nil {
		h.repositoryError(ctx, err)
		return
	}
	item := h.serializeTelescope(telescope, user.ID, 0)
	ctx.JSON(http.StatusOK, item)
}

func (h *Handler) CreateTelescopeDraftAPI(ctx *gin.Context) {
	if ctx.ContentType() != "multipart/form-data" {
		h.errorHandler(ctx, http.StatusUnsupportedMediaType, errors.New("ожидается multipart/form-data"))
		return
	}
	ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, media.MaxRequestSize)
	err := ctx.Request.ParseMultipartForm(2 << 20)
	if ctx.Request.MultipartForm != nil {
		defer ctx.Request.MultipartForm.RemoveAll()
	}
	if err != nil {
		code := http.StatusBadRequest
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			code = http.StatusRequestEntityTooLarge
		}
		h.errorHandler(ctx, code, err)
		return
	}
	for name, values := range ctx.Request.MultipartForm.Value {
		if name != "title" || len(values) != 1 {
			h.errorHandler(ctx, http.StatusBadRequest, fmt.Errorf("недопустимое текстовое поле: %s; разрешено только title", name))
			return
		}
	}
	for name, files := range ctx.Request.MultipartForm.File {
		if (name != "image" && name != "video") || len(files) != 1 {
			h.errorHandler(ctx, http.StatusBadRequest, errors.New("допустимо не более одного файла image и одного video"))
			return
		}
	}
	title := strings.TrimSpace(ctx.Request.PostForm.Get("title"))
	if title == "" || utf8.RuneCountInString(title) > 127 {
		h.errorHandler(ctx, http.StatusBadRequest, errors.New("название должно содержать от 1 до 127 символов"))
		return
	}
	image, ok := h.formFile(ctx, "image")
	if !ok {
		return
	}
	video, ok := h.formFile(ctx, "video")
	if !ok {
		return
	}
	user := session.GetCurrentUser()
	telescope, err := h.Repository.CreateDraftTelescope(ctx.Request.Context(), user.ID, title, image, video)
	if err != nil {
		h.repositoryError(ctx, err)
		return
	}
	item := h.serializeTelescope(telescope, user.ID, 0)
	ctx.Header("Location", "/api/telescopes/draft")
	ctx.JSON(http.StatusCreated, item)
}

func (h *Handler) formFile(ctx *gin.Context, name string) (*multipart.FileHeader, bool) {
	header, err := ctx.FormFile(name)
	if errors.Is(err, http.ErrMissingFile) {
		return nil, true
	}
	if err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return nil, false
	}
	if _, _, err = media.ValidateFileUpload(header, name); err != nil {
		code := http.StatusBadRequest
		if errors.Is(err, media.ErrTooLarge) {
			code = http.StatusRequestEntityTooLarge
		}
		h.errorHandler(ctx, code, err)
		return nil, false
	}
	return header, true
}

func (h *Handler) PublishTelescopeAPI(ctx *gin.Context) {
	id, ok := h.requestID(ctx)
	if !ok {
		return
	}
	var request ds.PublishTelescopeRequest
	if !h.bindJSON(ctx, &request) {
		return
	}
	request.Description = strings.TrimSpace(request.Description)
	if request.Description == "" {
		h.errorHandler(ctx, http.StatusBadRequest, errors.New("описание не может быть пустым"))
		return
	}
	user := session.GetCurrentUser()
	telescope, err := h.Repository.PublishTelescope(ctx.Request.Context(), id, user.ID, request.Description, *request.Latitude, *request.Longitude)
	if err != nil {
		h.repositoryError(ctx, err)
		return
	}
	item := h.serializeTelescope(telescope, user.ID, 0)
	ctx.JSON(http.StatusOK, item)
}

func (h *Handler) DeleteTelescopeAPI(ctx *gin.Context) {
	id, ok := h.requestID(ctx)
	if !ok {
		return
	}
	user := session.GetCurrentUser()
	if err := h.Repository.DeleteTelescope(ctx.Request.Context(), id, user.ID); err != nil {
		h.repositoryError(ctx, err)
		return
	}
	ctx.Status(http.StatusOK)
}

func (h *Handler) LikeTelescopeAPI(ctx *gin.Context) {
	id, ok := h.requestID(ctx)
	if !ok {
		return
	}
	var request ds.LikeTelescopeRequest
	if !h.bindJSON(ctx, &request) {
		return
	}
	user := session.GetCurrentUser()
	count, err := h.Repository.SetTelescopeLike(ctx.Request.Context(), id, user.ID, *request.Like)
	if err != nil {
		h.repositoryError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"is_liked": *request.Like, "likes_count": count})
}
