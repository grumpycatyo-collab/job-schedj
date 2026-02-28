package config

import (
	"fmt"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Redis RedisConfig `yaml:"redis"`
}

type RedisConfig struct {
	Addr     string `yaml:"address"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

func NewConfig(filePath string) (*Config, error) {
	var cfg Config
	if err := cleanenv.ReadConfig(filePath, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse configuration file", err)
	}

	return &cfg, nil
}
