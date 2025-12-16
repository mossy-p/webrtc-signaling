package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// LoginRequest represents the login request body
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse represents the login response
type LoginResponse struct {
	Token  string `json:"token"`
	UserID string `json:"user_id"`
}

// JWTClaims represents the claims in the JWT token
type JWTClaims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

// Login handles user login and JWT generation
// For demo purposes, accepts any username/password combination
func Login(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid request body",
			})
			return
		}

		// For demo: accept any username/password
		// In production, validate against a user database
		userID := req.Username

		// Generate JWT token
		claims := JWTClaims{
			UserID: userID,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
				NotBefore: jwt.NewNumericDate(time.Now()),
			},
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString([]byte(jwtSecret))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to generate token",
			})
			return
		}

		c.JSON(http.StatusOK, LoginResponse{
			Token:  tokenString,
			UserID: userID,
		})
	}
}
