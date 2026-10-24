package handler

import (
	"encoding/json"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func (h *SettingHandler) GetSupportContactConfig(c *gin.Context) {
	raw, err := h.settingService.GetSupportContactConfig(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	var cfg service.SupportContactConfig
	_ = json.Unmarshal([]byte(raw), &cfg)
	response.Success(c, cfg)
}
