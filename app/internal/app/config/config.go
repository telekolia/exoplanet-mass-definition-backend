package config

import (
	"os"
	"time"

	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type Config struct {
	ServiceHost string
	ServicePort int
	JWT         JWTConfig
	Redis       RedisConfig
}

type JWTConfig struct {
	Secret    string
	ExpiresIn time.Duration
}

type RedisConfig struct {
	Host         string
	Port         string
	User         string
	Password     string
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

func NewConfig() (*Config, error) {
	var err error

	configName := "config"
	_ = godotenv.Load()
	if os.Getenv("CONFIG_NAME") != "" {
		configName = os.Getenv("CONFIG_NAME")
	}

	viper.SetConfigName(configName)
	viper.SetConfigType("toml")
	viper.SetDefault("JWT.ExpiresIn", "1h")
	viper.SetDefault("Redis.DialTimeout", "3s")
	viper.SetDefault("Redis.ReadTimeout", "3s")
	viper.SetDefault("Redis.WriteTimeout", "3s")
	viper.AddConfigPath("config")
	viper.AddConfigPath(".")
	viper.WatchConfig()

	err = viper.ReadInConfig()
	if err != nil {
		return nil, err
	}

	cfg := &Config{}
	err = viper.Unmarshal(cfg)
	if err != nil {
		return nil, err
	}

	cfg.JWT.Secret = os.Getenv("JWT_SECRET")
	cfg.Redis.Host = os.Getenv("REDIS_HOST")
	if cfg.Redis.Host == "" {
		cfg.Redis.Host = "127.0.0.1"
	}
	cfg.Redis.Port = os.Getenv("REDIS_PORT")
	if cfg.Redis.Port == "" {
		cfg.Redis.Port = "6379"
	}
	cfg.Redis.User = os.Getenv("REDIS_USER")
	cfg.Redis.Password = os.Getenv("REDIS_PASSWORD")

	log.Info("config parsed")

	return cfg, nil
}
