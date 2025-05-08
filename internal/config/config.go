package config

import (
	"errors"
	"os"
	"time"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Host          string        `yaml:"host"`
	Port          string        `yaml:"port"`
	Backends      []string      `yaml:"backends"`
	Rate_limiting Rate_limiting `yaml:"rate_limiting"`
	Storage       Storage       `yaml:"storage"`
	HealthChecker HealthChecker `yaml:"healthcheck"`
	Balancer      Balancer      `yaml:"balancer"`
}

type HealthChecker struct {
	Interval time.Duration `yaml:"interval"`
	Timeout  time.Duration `yaml:"timeout"`
}

type Rate_limiting struct {
	Capacity        int `yaml:"capacity"`
	Rate_per_second int `yaml:"rate_per_second"`
}

type Storage struct {
	Redis Redis `yaml:"redis"`
}
type Redis struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
}

type Balancer struct {
	Algorithm string `yaml:"algorithm"`
}

// MustLoad загружает конфигурацию из файла YAML.
// Паникует при возникновении ошибок загрузки или парсинга.
func MustLoad() *Config {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "../config/config.yaml"
	}
	file, err := os.Open(configPath)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	decoder := yaml.NewDecoder(file)
	config := &Config{}
	err = decoder.Decode(config)
	if err != nil {
		panic(err)
	}
	err = validateConfig(config)
	if err != nil {
		panic(err)
	}

	return config
}
func validateConfig(config *Config) error {
	if config.Rate_limiting.Rate_per_second <= 0 {
		return errors.New("rate_per_second must be greater than 0")
	}
	if config.Rate_limiting.Capacity <= 0 {
		return errors.New("capacity must be greater than 0")
	}
	if config.Storage.Redis.Host == "" {
		return errors.New("redis host must be set")
	}
	if config.Storage.Redis.Port == 0 {
		return errors.New("redis port must be set")
	}
	if config.HealthChecker.Interval == 0 {
		return errors.New("healthcheck interval must be set")
	}
	if config.HealthChecker.Timeout == 0 {
		return errors.New("healthcheck timeout must be set")
	}
	if config.Host == "" {
		return errors.New("host must be set")
	}
	if config.Port == "" {
		return errors.New("port must be set")
	}
	if len(config.Backends) == 0 {
		return errors.New("backends must be set")
	}
	for _, backend := range config.Backends {
		if backend == "" {
			return errors.New("backend must be set")
		}
	}
	return nil
}
