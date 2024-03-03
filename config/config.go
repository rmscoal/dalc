package config

import (
	"log"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Server struct {
		Host string `yaml:"host" env:"SERVER_HOST"`
		Port int    `yaml:"port" env:"SERVER_PORT"`
	} `yaml:"server"`
	Database struct {
		URL string `yaml:"url" env:"DB_URL"`
	} `yaml:"database"`
	RabbitMQ struct {
		URL string `yaml:"url" env:"RABBITMQ_URL"`
	} `yaml:"rabbitmq"`
}

func GetConfig(path string) Config {
	var cfg Config

	if err := cleanenv.ReadConfig(path, &cfg); err != nil {
		log.Fatalf("unable to read config file from %s: %s", path, err)
	}

	return cfg
}
