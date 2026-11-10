package app

import (
	"app/internal/app/config"
	"app/internal/app/handler"
	"app/internal/app/redis"
	"app/internal/app/role"
	"app/internal/app/session"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type Application struct {
	Config    *config.Config
	Router    *gin.Engine
	Handler   *handler.Handler
	tokens    *session.Tokens
	blacklist *redis.Client
}

func NewApp(c *config.Config, r *gin.Engine, h *handler.Handler, tokens *session.Tokens, blacklist *redis.Client) *Application {
	return &Application{
		Config:    c,
		Router:    r,
		Handler:   h,
		tokens:    tokens,
		blacklist: blacklist,
	}
}

func (a *Application) RunApp() {
	logrus.Info("Server start up")

	a.Handler.RegisterHandler(a.Router, a.WithAuthCheck(role.Guest, role.Creator), a.WithAuthCheck(role.Creator))
	a.Handler.RegisterStatic(a.Router)
	a.Handler.RegisterSwagger(a.Router)

	serverAddress := fmt.Sprintf("%s:%d", a.Config.ServiceHost, a.Config.ServicePort)
	if err := a.Router.Run(serverAddress); err != nil {
		logrus.Fatal(err)
	}
	logrus.Info("Server down")
}
