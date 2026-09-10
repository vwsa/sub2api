package middleware

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

const openAIDeviceIDHeader = "x-codex-installation-id"

// OpenAIDeviceIdentity attaches a bounded, stable downstream installation ID
// to the request context. The OpenAI scheduler decides whether it is required.
func OpenAIDeviceIdentity() gin.HandlerFunc {
	return func(c *gin.Context) {
		deviceID := strings.TrimSpace(c.GetHeader(openAIDeviceIDHeader))
		if len(deviceID) >= 8 && len(deviceID) <= 256 {
			ctx := service.WithOpenAIDeviceID(c.Request.Context(), deviceID)
			if apiKey, ok := GetAPIKeyFromContext(c); ok && apiKey != nil {
				ctx = service.WithOpenAIDevicePrincipal(ctx, service.OpenAIDevicePrincipal{
					UserID:     apiKey.UserID,
					APIKeyID:   apiKey.ID,
					APIKeyName: apiKey.Name,
				})
			}
			c.Request = c.Request.WithContext(ctx)
		}
		c.Next()
	}
}
