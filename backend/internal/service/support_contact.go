package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
)

const defaultSupportContactConfigJSON = `{"enabled":true,"title":"","description":"","contacts":[]}`
const maxSupportContactItems = 3

type SupportContactConfig struct {
	Enabled     bool                 `json:"enabled"`
	Title       string               `json:"title"`
	Description string               `json:"description"`
	Contacts    []SupportContactItem `json:"contacts"`
}

type SupportContactItem struct {
	Name      string `json:"name"`
	WorkHours string `json:"work_hours"`
	QRImage   string `json:"qr_image"`
}

func NormalizeSupportContactConfigJSON(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultSupportContactConfigJSON
	}
	var decoded struct {
		Enabled     *bool                `json:"enabled"`
		Title       string               `json:"title"`
		Description string               `json:"description"`
		Contacts    []SupportContactItem `json:"contacts"`
	}
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		return defaultSupportContactConfigJSON
	}

	cfg := SupportContactConfig{
		Enabled:     true,
		Title:       decoded.Title,
		Description: decoded.Description,
		Contacts:    decoded.Contacts,
	}
	if decoded.Enabled != nil {
		cfg.Enabled = *decoded.Enabled
	}

	cfg.Title = strings.TrimSpace(cfg.Title)
	cfg.Description = strings.TrimSpace(cfg.Description)

	contacts := make([]SupportContactItem, 0, min(len(cfg.Contacts), maxSupportContactItems))
	for _, item := range cfg.Contacts {
		normalized := SupportContactItem{
			Name:      strings.TrimSpace(item.Name),
			WorkHours: strings.TrimSpace(item.WorkHours),
			QRImage:   strings.TrimSpace(item.QRImage),
		}
		if normalized.Name == "" && normalized.WorkHours == "" && normalized.QRImage == "" {
			continue
		}
		contacts = append(contacts, normalized)
		if len(contacts) >= maxSupportContactItems {
			break
		}
	}
	cfg.Contacts = contacts

	data, err := json.Marshal(cfg)
	if err != nil {
		return defaultSupportContactConfigJSON
	}
	return string(data)
}

func (s *SettingService) GetSupportContactConfig(ctx context.Context) (string, error) {
	if s == nil || s.settingRepo == nil {
		return NormalizeSupportContactConfigJSON(""), nil
	}
	raw, err := s.settingRepo.GetValue(ctx, SettingKeySupportContactConfig)
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) {
			return NormalizeSupportContactConfigJSON(""), nil
		}
		return "", err
	}
	return NormalizeSupportContactConfigJSON(raw), nil
}
