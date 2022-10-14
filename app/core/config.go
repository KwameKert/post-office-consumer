package core

import (
	"log"
	"os"
	"strconv"
)

type Environment string

const (
	Development Environment = "dev"
	Staging     Environment = "staging"
)

type Config struct {
	CONNECTION  string
	DATABASE    string
	ENVIRONMENT Environment
	PORT        int
}

func Get(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func GetInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		i, err := strconv.Atoi(v)
		if err != nil {
			log.Printf("%s: %s", key, err)
			return fallback
		}
		return i
	}
	return fallback
}

func GetEnvironment() Environment {
	if env := Get("ENV", ""); env == "" {
		return Development
	} else {
		return Environment(env)
	}
}

func NewConfig() *Config {
	return &Config{
		CONNECTION:  Get("CONNECTION", "mongodb://localhost:27017"),
		DATABASE:    Get("DATABASE", "agerp-post-office"),
		PORT:        GetInt("PORT", 6363),
		ENVIRONMENT: GetEnvironment(),
	}
}
