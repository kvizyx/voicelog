package config

import (
	"fmt"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

type Config struct {
	Env      string `env:"ENV"`
	BotToken string `env:"BOT_TOKEN"`

	S3      S3
	HTTP    HTTP
	Discord Discord
}

type S3 struct {
	Addr      string `env:"STORAGE_S3_ADDR"`
	AccessKey string `env:"STORAGE_S3_ACCESS_KEY"`
	SecretKey string `env:"STORAGE_S3_SECRET_KEY"`
	Bucket    string `env:"STORAGE_S3_BUCKET"`
	Region    string `env:"STORAGE_S3_REGION"`
}

type HTTP struct {
	Port         uint16        `env:"HTTP_PORT"`
	IdleTimeout  time.Duration `env:"HTTP_IDLE_TIMEOUT"`
	ReadTimeout  time.Duration `env:"HTTP_READ_TIMEOUT"`
	WriteTimeout time.Duration `env:"HTTP_WRITE_TIMEOUT"`
}

type Discord struct {
	ClientID     string `env:"DISCORD_CLIENT_ID"`
	ClientSecret string `env:"DISCORD_CLIENT_SECRET"`
}

func New(path string) (Config, error) {
	var (
		config Config
		files  = make([]string, 1)
	)

	if len(path) != 0 {
		files[0] = path
	}

	// if path to config is not defined then
	if err := godotenv.Load(files...); err != nil {
		return Config{}, fmt.Errorf("failed to load config: %w", err)
	}

	if err := cleanenv.ReadEnv(&config); err != nil {
		return Config{}, fmt.Errorf("failed to read config: %w", err)
	}

	return config, nil
}
