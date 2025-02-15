package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		Host string `yaml:"host"`
		Port int    `yaml:"port"`
	} `yaml:"server"`

	Database struct {
		User     string `yaml:"user"`
		Password string `yaml:"password"`
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		Name     string `yaml:"name"`
		DSN      string `yaml:"dsn"`
	} `yaml:"database"`

	Log struct {
		Level  string `yaml:"level"`
		Output string `yaml:"output"`
	} `yaml:"log"`

	// RateLimit struct {
	// 	RequestsPerSecond int `yaml:"requests_per_second"`
	// 	Burst            int `yaml:"burst"`
	// } `yaml:"rate_limit"`
}

func Init(path string) (*Config, error) {
	yml, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	c := &Config{}
	if err = yaml.Unmarshal(yml, c); err != nil {
		return nil, err
	}

	return c, nil
}
