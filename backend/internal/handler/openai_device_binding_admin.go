package handler

import (
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/gin-gonic/gin"
)

func (h *OpenAIGatewayHandler) AdminListDeviceBindings(c *gin.Context) {
	bindings, err := h.gatewayService.ListOpenAIDeviceBindings(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to list OpenAI device bindings")
		return
	}
	response.Success(c, gin.H{
		"items":                   bindings,
		"total":                   len(bindings),
		"enabled":                 h.cfg.Gateway.OpenAIDeviceBinding.Enabled,
		"max_devices_per_account": h.cfg.Gateway.OpenAIDeviceBinding.MaxDevicesPerAccount,
		"idle_ttl_days":           h.cfg.Gateway.OpenAIDeviceBinding.IdleTTLDays,
	})
}

func (h *OpenAIGatewayHandler) AdminDeleteDeviceBinding(c *gin.Context) {
	deviceHash := strings.TrimSpace(c.Param("device_hash"))
	decoded, err := hex.DecodeString(deviceHash)
	if err != nil || len(decoded) != 32 {
		response.BadRequest(c, "Invalid device hash")
		return
	}
	if err := h.gatewayService.DeleteOpenAIDeviceBinding(c.Request.Context(), deviceHash); err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to remove OpenAI device binding")
		return
	}
	response.Success(c, gin.H{"device_hash": deviceHash})
}
