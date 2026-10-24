package admin

import (
	"encoding/json"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func parseSupportContactConfig(raw string) service.SupportContactConfig {
	var cfg service.SupportContactConfig
	_ = json.Unmarshal([]byte(service.NormalizeSupportContactConfigJSON(raw)), &cfg)
	return cfg
}
