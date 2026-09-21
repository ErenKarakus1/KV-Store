package main

import (
	"log"
	"strconv"
	"time"

	"github.com/ErenKarakus1/KV-Store/internal/config"
	"github.com/ErenKarakus1/KV-Store/internal/handler"
	"github.com/ErenKarakus1/KV-Store/internal/store"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.LoadConfig()
	storeCapacity, err := strconv.Atoi(cfg.StoreCapacity)
	if err != nil {
		log.Fatal("invalid store capacity")
	}
	cleanupInterval, err := time.ParseDuration(cfg.CleanupInterval)
	if err != nil {
		log.Fatal("invalid cleanup interval")
	}

	s := store.NewStore(storeCapacity)

	stopCleanup := s.StartCleanup(cleanupInterval)
	defer stopCleanup()

	router := gin.Default()

	router.GET("/kv/:key/exists", handler.ExistsHandler(s))
	router.POST("/kv/:key/increment", handler.IncrementHandler(s))
	router.GET("/kv/:key", handler.GetHandler(s))
	router.PUT("/kv/:key", handler.SetHandler(s))
	router.DELETE("/kv/:key", handler.DeleteHandler(s))

	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}
}
