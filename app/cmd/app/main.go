package main

import (
	"app/internal/app/config"
	"app/internal/app/dsn"
	"app/internal/app/handler"
	"app/internal/app/redis"
	"app/internal/app/repository"
	"app/internal/app/session"
	"app/internal/pkg/app"
	"context"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// @title Телескопная API
// @version 1.0
// @BasePath /api
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Введите Bearer, пробел и токен из POST /api/users/login.
func main() {
	router := gin.Default()
	conf, err := config.NewConfig()
	if err != nil {
		logrus.Fatalf("error loading config: %v", err)
	}

	postgresString := dsn.FromEnv()
	useSSL := false
	if value := os.Getenv("MINIO_SECURE"); value != "" {
		useSSL, err = strconv.ParseBool(value)
		if err != nil {
			logrus.Fatal("MINIO_SECURE must be true or false")
		}
	}

	rep, errRep := repository.New(&repository.RepositorySettings{
		PostgresDSN:     postgresString,
		MinioEndpoint:   os.Getenv("MINIO_ENDPOINT"),
		MinioAccessKey:  os.Getenv("MINIO_ACCESS_KEY"),
		MinioSecretKey:  os.Getenv("MINIO_SECRET_KEY"),
		MinioBucketName: os.Getenv("MINIO_BUCKET_NAME"),
		MinioSecure:     useSSL,
	})
	if errRep != nil {
		logrus.Fatalf("error initializing repository: %v", errRep)
	}

	tokens, err := session.NewTokens(conf.JWT.Secret, conf.JWT.ExpiresIn)
	if err != nil {
		logrus.Fatal(err)
	}
	blacklist, err := redis.New(context.Background(), conf.Redis)
	if err != nil {
		logrus.Fatal(err)
	}
	defer blacklist.Close()

	hand := handler.NewHandler(rep, tokens, blacklist)

	application := app.NewApp(conf, router, hand, tokens, blacklist)
	application.RunApp()
}
