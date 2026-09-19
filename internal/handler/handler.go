package handler

import (
	"net/http"

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
