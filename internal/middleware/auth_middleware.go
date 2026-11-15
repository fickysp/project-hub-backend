package middleware

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	myjwt "github.com/fickysp/project-hub-backend.git/pkg/jwt"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/patrickmn/go-cache"
)

type AuthMiddleware interface {
	RequireToken() gin.HandlerFunc
	RequireRole(allowedRoles ...string) gin.HandlerFunc
}

type authMiddleware struct {
	jwtSecret      string
	blacklistCache *cache.Cache
}

func NewAuthMiddleware(jwtSecret string, blacklistCache *cache.Cache) AuthMiddleware {
	return &authMiddleware{
		jwtSecret:      jwtSecret,
		blacklistCache: blacklistCache,
	}
}

func (m *authMiddleware) RequireToken() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			slog.Info("access_token tidak ditemukan")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "Missing authorization token",
			})
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		if m.blacklistCache != nil {
			if _, found := m.blacklistCache.Get(tokenString); found {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"success": false,
					"message": "Token has been revoked (logged out)",
				})
				return
			}
		}

		token, err := jwt.ParseWithClaims(tokenString, &myjwt.Claims{}, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return []byte(m.jwtSecret), nil
		})

		if err != nil || token == nil || !token.Valid {
			slog.Error("Validasi token gagal", "error", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "Token invalid or expired",
			})
			return
		}

		claims, ok := token.Claims.(*myjwt.Claims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "Failed to parse token claims",
			})
			return
		}

		if claims.Type != "access_token" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "Token type invalid",
			})
			return
		}

		c.Set("user_id", claims.ID)
		c.Set("name", claims.Name)

		c.Next()
	}
}

func (m *authMiddleware) RequireRole(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleVal, exists := c.Get("role")
		userRole, ok := roleVal.(string)
		if !exists || !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "User role not found in session",
			})
			return
		}

		for _, role := range allowedRoles {
			if strings.EqualFold(role, userRole) {
				c.Next()
				return
			}
		}

		slog.Warn("Akses role ditolak", "user_role", userRole, "allowed_roles", allowedRoles)
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"success": false,
			"message": "Access denied for this role",
		})
	}
}
