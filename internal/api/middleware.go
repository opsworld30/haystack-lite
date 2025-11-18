package api

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		log.Printf("[%s] %s %d %v",
			c.Request.Method,
			path,
			status,
			latency,
		)
	}
}

func Recovery() gin.HandlerFunc {
	return gin.Recovery()
}
