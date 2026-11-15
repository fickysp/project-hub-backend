package middleware

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/url"
	"strings"
)

// OriginPolicy uses exact origins, including scheme and port. Wildcards are
// deliberately unsupported because the frontend sends credentials/cookies.
type OriginPolicy struct{ allowed map[string]bool }

func NewOriginPolicy(origins []string) (*OriginPolicy, error) {
	p := &OriginPolicy{allowed: make(map[string]bool)}
	for _, origin := range origins {
		origin = strings.TrimRight(strings.TrimSpace(origin), "/")
		if origin == "" {
			continue
		}
		u, err := url.Parse(origin)
		if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || strings.Contains(origin, "*") {
			return nil, fmt.Errorf("invalid CORS origin %q: use an exact http(s) origin without a path", origin)
		}
		p.allowed[origin] = true
	}
	return p, nil
}
func (p *OriginPolicy) Allows(origin string) bool { return p.allowed[origin] }

func CORS(policy *OriginPolicy) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Responses can differ by Origin, even for a denied or missing Origin.
		c.Writer.Header().Add("Vary", "Origin")
		origin := c.GetHeader("Origin")
		if origin == "" {
			c.Next()
			return
		} // curl, server clients, and health checks
		if !policy.Allows(origin) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"success": false, "message": "Origin is not allowed"})
			return
		}
		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Expose-Headers", "X-Cache")
		if c.Request.Method == http.MethodOptions && c.GetHeader("Access-Control-Request-Method") != "" {
			c.Writer.Header().Add("Vary", "Access-Control-Request-Method")
			c.Writer.Header().Add("Vary", "Access-Control-Request-Headers")
			methods := map[string]bool{"GET": true, "POST": true, "PATCH": true, "DELETE": true, "OPTIONS": true}
			if !methods[c.GetHeader("Access-Control-Request-Method")] {
				c.AbortWithStatusJSON(403, gin.H{"success": false, "message": "CORS method is not allowed"})
				return
			}
			for _, h := range strings.Split(c.GetHeader("Access-Control-Request-Headers"), ",") {
				switch strings.ToLower(strings.TrimSpace(h)) {
				case "", "authorization", "content-type", "accept":
				default:
					c.AbortWithStatusJSON(403, gin.H{"success": false, "message": "CORS header is not allowed"})
					return
				}
			}
			c.Header("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, Accept")
			c.Header("Access-Control-Max-Age", "600")
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
