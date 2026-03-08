package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func registerHealthRoute(router *gin.Engine, ping func() error) {
	router.GET("/health", func(c *gin.Context) {
		if ping != nil {
			if err := ping(); err != nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{
					"status": "degraded",
					"error":  err.Error(),
				})
				return
			}
		}

		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
}
