package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	StoreCapacity string
	Port          string
}

func LoadConfig() Config {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	return Config{
		StoreCapacity: getEnv("STORE_CAPACITY"),
		Port:          getEnv("PORT"),
	}
}

func getEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("%s is not set", key)
	}
	return value
}
