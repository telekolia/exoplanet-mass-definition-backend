package main

import (
	"app/internal/app/config"
	"app/internal/app/dsn"
	"app/internal/app/handler"
	"app/internal/app/repository"
	"app/internal/app/session"
	"app/internal/pkg"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
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

	redisHost := os.Getenv("REDIS_HOST")
	if redisHost == "" {
		redisHost = "127.0.0.1"
	}
	redisPort := os.Getenv("REDIS_PORT")
	if redisPort == "" {
		redisPort = "6379"
	}
	tokens, err := session.NewTokens(os.Getenv("JWT_SECRET"), &redis.Options{
		Addr:     net.JoinHostPort(redisHost, redisPort),
		Username: os.Getenv("REDIS_USER"), Password: os.Getenv("REDIS_PASSWORD"),
		DialTimeout: 3 * time.Second, ReadTimeout: 3 * time.Second, WriteTimeout: 3 * time.Second,
	})
	if err != nil {
		logrus.Fatal(err)
	}
	defer tokens.Close()

	hand := handler.NewHandler(rep, tokens)

	application := pkg.NewApp(conf, router, hand)
	application.RunApp()
}
