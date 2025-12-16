package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// OriginFilter creates middleware that filters requests based on allowed origins
func OriginFilter(allowedOrigins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		// If no origin header (direct WebSocket connection), check Sec-WebSocket-Origin
		if origin == "" {
			origin = c.GetHeader("Sec-WebSocket-Origin")
		}

		// Check if origin is allowed
		allowed := false
		for _, allowedOrigin := range allowedOrigins {
			if origin == allowedOrigin {
				allowed = true
				break
			}
		}

		if !allowed && origin != "" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "Origin not allowed",
			})
			return
		}

		// Set CORS headers for allowed origins
		if allowed {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		c.Next()
	}
}
