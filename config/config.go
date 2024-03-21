package config

import (
	"flag"
	"log"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

type (
	Server struct {
		Host string `yaml:"host" env:"SERVER_HOST"`
		Port int    `yaml:"port" env:"SERVER_PORT"`
	}

	Database struct {
		Username string `yaml:"username" env:"DB_USERNAME"`
		Password string `yaml:"password" env:"DB_PASSWORD"`
		DBName   string `yaml:"dbname" env:"DB_NAME"`
		Host     string `yaml:"host" env:"DB_HOST"`
		SSLMode  string `yaml:"sslmode" env:"DB_SSLMODE"`
	}

	RabbitMQ struct {
		Username    string `yaml:"username" env:"RABBITMQ_USERNAME"`
		Password    string `yaml:"password" env:"RABBITMQ_PASSWORD"`
		Host        string `yaml:"host" env:"RABBITMQ_HOST"`
		VirtualHost string `yaml:"virtual_host" env:"RABBITMQ_VIRTUAL_HOST"`
	}

	Config struct {
		Server   Server   `yaml:"server"`
		Database Database `yaml:"database"`
		RabbitMQ RabbitMQ `yaml:"rabbitmq"`
	}
)

func GetConfig() Config {
	var cfg Config
	var envFile string

	flag.StringVar(&envFile, "env-file", "config.yaml", "Read environments variables from...")
	flag.Parse()

	if envFile != "" {
		if err := cleanenv.ReadConfig(envFile, &cfg); err != nil {
			log.Fatalf("unable to read config file from %s: %s", envFile, err)
		}
	} else {
		cfg = Config{
			Server: Server{
				Host: os.Getenv("SERVER_HOST"),
			},
			Database: Database{
				Username: os.Getenv("DB_USERNAME"),
				Password: os.Getenv("DB_PASSWORD"),
				DBName:   os.Getenv("DB_NAME"),
				Host:     os.Getenv("DB_HOST"),
				SSLMode:  os.Getenv("DB_SSLMODE"),
			},
			RabbitMQ: RabbitMQ{
				Username:    os.Getenv("RABBITMQ_USERNAME"),
				Password:    os.Getenv("RABBITMQ_PASSWORD"),
				Host:        os.Getenv("RABBITMQ_HOST"),
				VirtualHost: os.Getenv("RABBITMQ_VIRTUAL_HOST"),
			},
		}
	}

	return cfg
}
