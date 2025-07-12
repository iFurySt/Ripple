package service

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pquerna/otp/totp"
	"go.uber.org/zap"
)

type AuthService struct {
	logger     *zap.Logger
	totpSecret string
}

func NewAuthService(logger *zap.Logger, totpSecret string) *AuthService {
	return &AuthService{
		logger:     logger,
		totpSecret: totpSecret,
	}
}

func (a *AuthService) GenerateSecret() (string, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "Ripple Dashboard",
		AccountName: "admin",
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate TOTP key: %w", err)
	}
	
	return key.Secret(), nil
}

func (a *AuthService) GenerateQRCode(issuer, accountName, secret string) (string, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      issuer,
		AccountName: accountName,
		Secret:      []byte(secret),
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate TOTP key: %w", err)
	}
	
	return key.URL(), nil
}

func (a *AuthService) ValidateToken(token string) bool {
	valid := totp.Validate(token, a.totpSecret)
	if valid {
		a.logger.Info("TOTP token validation successful")
	} else {
		a.logger.Warn("TOTP token validation failed", zap.String("token", token))
	}
	return valid
}

func (a *AuthService) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip auth for login page and API auth endpoints
		if c.Request.URL.Path == "/login" || 
		   c.Request.URL.Path == "/api/v1/auth/login" ||
		   c.Request.URL.Path == "/api/v1/auth/setup" {
			c.Next()
			return
		}

		// Check session token
		token, err := c.Cookie("auth_token")
		if err != nil {
			a.redirectToLogin(c)
			return
		}

		// Validate session (simple implementation - in production use proper JWT or session store)
		if !a.isValidSession(token) {
			a.redirectToLogin(c)
			return
		}

		c.Next()
	}
}

func (a *AuthService) isValidSession(token string) bool {
	// Simple implementation - in production use proper session management
	// For now, just check if token is not empty and has reasonable length
	return len(token) > 10
}

func (a *AuthService) redirectToLogin(c *gin.Context) {
	// For API requests, return JSON error
	if c.Request.URL.Path != "/" && (len(c.Request.URL.Path) > 4 && c.Request.URL.Path[:4] == "/api") {
		c.JSON(401, gin.H{"error": "Authentication required"})
		c.Abort()
		return
	}
	
	// For web requests, redirect to login
	c.Redirect(302, "/login")
	c.Abort()
}

func (a *AuthService) CreateSession() string {
	// Simple implementation - in production use proper session management
	return fmt.Sprintf("session_%d", time.Now().Unix())
}