package handler

import (
	"net/http"
	"time"

	"github.com/ErenKarakus1/KV-Store/internal/model"
	"github.com/ErenKarakus1/KV-Store/internal/store"
	"github.com/gin-gonic/gin"
)

func GetHandler(s *store.Store) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		key := ctx.Param("key")
		if key == "" {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "key is required"})
			return
		}
		value, ok := s.Get(key)
		if !ok {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "key not found"})
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"key": key, "value": value})
	}
}

func SetHandler(s *store.Store) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		key := ctx.Param("key")
		if key == "" {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "key is required"})
			return
		}
		var req model.SetRequest
		if err := ctx.ShouldBindJSON(&req); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}
		if req.Value == "" {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "value is required"})
			return
		}
		if req.TTL == "" {
			s.Set(key, req.Value)
			ctx.JSON(http.StatusOK, gin.H{"key": key, "value": req.Value})
			return
		}
		duration, err := time.ParseDuration(req.TTL)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid ttl"})
			return
		}
		ok := s.SetWithTTL(key, req.Value, duration)
		if !ok {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid ttl"})
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"key": key, "value": req.Value, "ttl": req.TTL})
	}
}

func DeleteHandler(s *store.Store) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		key := ctx.Param("key")
		if key == "" {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "key is required"})
			return
		}
		ok := s.Delete(key)
		if !ok {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "key not found"})
			return
		}
		ctx.Status(http.StatusNoContent)
	}
}

func ExistsHandler(s *store.Store) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		key := ctx.Param("key")
		if key == "" {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "key is required"})
			return
		}
		exists := s.Exists(key)
		ctx.JSON(http.StatusOK, gin.H{"key": key, "exists": exists})
	}
}

func IncrementHandler(s *store.Store) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		key := ctx.Param("key")
		if key == "" {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "key is required"})
			return
		}
		incrementedValue, ok := s.Increment(key)
		if !ok {
			ctx.JSON(http.StatusConflict, gin.H{"error": "increment failed"})
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"key": key, "value": incrementedValue})
	}
}
