package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestOpenAIDeviceIdentityPassesStableHeaderToServiceContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(OpenAIDeviceIdentity())
	router.GET("/", func(c *gin.Context) {
		require.Equal(t, "installation-123", service.OpenAIDeviceIDFromContext(c.Request.Context()))
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(openAIDeviceIDHeader, "installation-123")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	require.Equal(t, http.StatusNoContent, resp.Code)
}
