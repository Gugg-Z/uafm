package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"
)

var helpCenterIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{1,63}$`)

type HelpCenterConfig struct {
	Enabled          bool                       `json:"enabled"`
	BaseURL          string                     `json:"base_url"`
	Title            string                     `json:"title"`
	Description      string                     `json:"description"`
	KeyCreatedPrompt HelpCenterKeyCreatedPrompt `json:"key_created_prompt"`
	Tutorials        []HelpCenterTutorial       `json:"tutorials"`
	FAQs             []HelpCenterFAQ            `json:"faqs"`
}

type HelpCenterKeyCreatedPrompt struct {
	Enabled              bool   `json:"enabled"`
	Title                string `json:"title"`
	Description          string `json:"description"`
	PrimaryActionLabel   string `json:"primary_action_label"`
	PrimaryActionURL     string `json:"primary_action_url"`
	SecondaryActionLabel string `json:"secondary_action_label"`
	SecondaryActionURL   string `json:"secondary_action_url"`
	DismissLabel         string `json:"dismiss_label"`
}

type HelpCenterTutorial struct {
	ID          string                 `json:"id"`
	Enabled     bool                   `json:"enabled"`
	SortOrder   int                    `json:"sort_order"`
	Title       string                 `json:"title"`
	Badge       string                 `json:"badge"`
	Summary     string                 `json:"summary"`
	ContentMD   string                 `json:"content_md"`
	Steps       []HelpCenterStep       `json:"steps"`
	CodeBlocks  []HelpCenterCodeBlock  `json:"code_blocks"`
	Links       []HelpCenterLink       `json:"links"`
	Attachments []HelpCenterAttachment `json:"attachments"`
}

type HelpCenterStep struct {
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	CodeBlocks  []HelpCenterCodeBlock  `json:"code_blocks"`
	Images      []HelpCenterAttachment `json:"images"`
	Attachments []HelpCenterAttachment `json:"attachments"`
}

type HelpCenterCodeBlock struct {
	Title    string `json:"title"`
	Language string `json:"language"`
	Content  string `json:"content"`
}

type HelpCenterLink struct {
	Label string `json:"label"`
	URL   string `json:"url"`
}

type HelpCenterAttachment struct {
	Label    string `json:"label"`
	URL      string `json:"url"`
	FileName string `json:"file_name"`
}

type HelpCenterFAQ struct {
	ID        string   `json:"id"`
	Enabled   bool     `json:"enabled"`
	SortOrder int      `json:"sort_order"`
	Question  string   `json:"question"`
	AnswerMD  string   `json:"answer_md"`
	Tags      []string `json:"tags"`
}

type HelpCenterService struct {
	settingRepo SettingRepository
}

func NewHelpCenterService(settingRepo SettingRepository) *HelpCenterService {
	return &HelpCenterService{settingRepo: settingRepo}
}

const defaultHelpCenterConfigLiteral = `{"enabled":true,"base_url":"","title":"","description":"","key_created_prompt":{"enabled":true,"title":"","description":"","primary_action_label":"","primary_action_url":"/help-center","secondary_action_label":"","secondary_action_url":"/keys","dismiss_label":""},"tutorials":[{"id":"getting-started","enabled":true,"sort_order":10,"title":"API 接入","badge":"","summary":"创建密钥并配置客户端","content_md":"在 API 密钥页创建密钥，使用该页面的配置入口选择客户端和模型。","steps":[{"title":"准备 API Key","description":"前往 API 密钥页创建或选择密钥。","code_blocks":[],"images":[],"attachments":[]}],"code_blocks":[],"links":[{"label":"API 密钥","url":"/keys"}],"attachments":[]}],"faqs":[{"id":"where-to-create-key","enabled":true,"sort_order":10,"question":"在哪里创建 API Key？","answer_md":"进入 [API 密钥](/keys) 页面创建或选择密钥。","tags":[]}]}`

func DefaultHelpCenterConfig() HelpCenterConfig {
	var cfg HelpCenterConfig
	_ = json.Unmarshal([]byte(defaultHelpCenterConfigLiteral), &cfg)
	return cfg
}

func DefaultHelpCenterConfigJSON() string {
	cfg := normalizedDefaultHelpCenterConfig()
	data, err := json.Marshal(cfg)
	if err != nil {
		return "{}"
	}
	return string(data)
}

func normalizedDefaultHelpCenterConfig() HelpCenterConfig {
	cfg, err := NormalizeHelpCenterConfig(DefaultHelpCenterConfig())
	if err != nil {
		return DefaultHelpCenterConfig()
	}
	return cfg
}

func NormalizeHelpCenterConfig(cfg HelpCenterConfig) (HelpCenterConfig, error) {
	cfg.Title = strings.TrimSpace(cfg.Title)
	if cfg.Title == "" {
		cfg.Title = "帮助中心"
	}
	cfg.Description = strings.TrimSpace(cfg.Description)
	cfg.BaseURL = strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if cfg.BaseURL != "" && !isAllowedHelpCenterURL(cfg.BaseURL) {
		return HelpCenterConfig{}, fmt.Errorf("invalid url for base_url")
	}

	cfg.KeyCreatedPrompt.Title = strings.TrimSpace(cfg.KeyCreatedPrompt.Title)
	cfg.KeyCreatedPrompt.Description = strings.TrimSpace(cfg.KeyCreatedPrompt.Description)
	cfg.KeyCreatedPrompt.PrimaryActionLabel = strings.TrimSpace(cfg.KeyCreatedPrompt.PrimaryActionLabel)
	cfg.KeyCreatedPrompt.PrimaryActionURL = strings.TrimSpace(cfg.KeyCreatedPrompt.PrimaryActionURL)
	cfg.KeyCreatedPrompt.SecondaryActionLabel = strings.TrimSpace(cfg.KeyCreatedPrompt.SecondaryActionLabel)
	cfg.KeyCreatedPrompt.SecondaryActionURL = strings.TrimSpace(cfg.KeyCreatedPrompt.SecondaryActionURL)
	cfg.KeyCreatedPrompt.DismissLabel = strings.TrimSpace(cfg.KeyCreatedPrompt.DismissLabel)
	if cfg.KeyCreatedPrompt.Title == "" {
		cfg.KeyCreatedPrompt.Title = "API 密钥已创建"
	}
	if cfg.KeyCreatedPrompt.PrimaryActionLabel == "" {
		cfg.KeyCreatedPrompt.PrimaryActionLabel = "查看帮助中心"
	}
	if cfg.KeyCreatedPrompt.PrimaryActionURL == "" {
		cfg.KeyCreatedPrompt.PrimaryActionURL = "/help-center"
	}
	if cfg.KeyCreatedPrompt.SecondaryActionLabel == "" {
		cfg.KeyCreatedPrompt.SecondaryActionLabel = "留在 API 密钥页"
	}
	if cfg.KeyCreatedPrompt.SecondaryActionURL == "" {
		cfg.KeyCreatedPrompt.SecondaryActionURL = "/keys"
	}
	if cfg.KeyCreatedPrompt.DismissLabel == "" {
		cfg.KeyCreatedPrompt.DismissLabel = "不再提示"
	}
	if !isAllowedHelpCenterURL(cfg.KeyCreatedPrompt.PrimaryActionURL) {
		return HelpCenterConfig{}, fmt.Errorf("invalid url for key_created_prompt.primary_action_url")
	}
	if !isAllowedHelpCenterURL(cfg.KeyCreatedPrompt.SecondaryActionURL) {
		return HelpCenterConfig{}, fmt.Errorf("invalid url for key_created_prompt.secondary_action_url")
	}

	if cfg.Tutorials == nil {
		cfg.Tutorials = []HelpCenterTutorial{}
	}
	seen := make(map[string]struct{}, len(cfg.Tutorials))
	for i := range cfg.Tutorials {
		tutorial := &cfg.Tutorials[i]
		tutorial.ID = strings.ToLower(strings.TrimSpace(tutorial.ID))
		if !helpCenterIDPattern.MatchString(tutorial.ID) {
			return HelpCenterConfig{}, fmt.Errorf("invalid tutorial id %q", tutorial.ID)
		}
		if _, ok := seen[tutorial.ID]; ok {
			return HelpCenterConfig{}, fmt.Errorf("duplicate tutorial id %q", tutorial.ID)
		}
		seen[tutorial.ID] = struct{}{}

		tutorial.Title = strings.TrimSpace(tutorial.Title)
		if tutorial.Enabled && tutorial.Title == "" {
			return HelpCenterConfig{}, fmt.Errorf("tutorial %q title is required", tutorial.ID)
		}
		tutorial.Badge = strings.TrimSpace(tutorial.Badge)
		tutorial.Summary = strings.TrimSpace(tutorial.Summary)
		tutorial.ContentMD = strings.TrimSpace(tutorial.ContentMD)
		if tutorial.Steps == nil {
			tutorial.Steps = []HelpCenterStep{}
		}
		for j := range tutorial.Steps {
			step := &tutorial.Steps[j]
			step.Title = strings.TrimSpace(step.Title)
			step.Description = strings.TrimSpace(step.Description)
			if step.CodeBlocks == nil {
				step.CodeBlocks = []HelpCenterCodeBlock{}
			}
			for k := range step.CodeBlocks {
				step.CodeBlocks[k].Title = strings.TrimSpace(step.CodeBlocks[k].Title)
				step.CodeBlocks[k].Language = strings.TrimSpace(step.CodeBlocks[k].Language)
			}
			if step.Images == nil {
				step.Images = []HelpCenterAttachment{}
			}
			for k := range step.Images {
				step.Images[k].Label = strings.TrimSpace(step.Images[k].Label)
				step.Images[k].URL = strings.TrimSpace(step.Images[k].URL)
				step.Images[k].FileName = strings.TrimSpace(step.Images[k].FileName)
				if step.Images[k].Label == "" || step.Images[k].URL == "" {
					return HelpCenterConfig{}, fmt.Errorf("tutorial %q step image label and url are required", tutorial.ID)
				}
				if !isAllowedHelpCenterURL(step.Images[k].URL) {
					return HelpCenterConfig{}, fmt.Errorf("invalid url for tutorial %q step image", tutorial.ID)
				}
			}
			if step.Attachments == nil {
				step.Attachments = []HelpCenterAttachment{}
			}
			for k := range step.Attachments {
				step.Attachments[k].Label = strings.TrimSpace(step.Attachments[k].Label)
				step.Attachments[k].URL = strings.TrimSpace(step.Attachments[k].URL)
				step.Attachments[k].FileName = strings.TrimSpace(step.Attachments[k].FileName)
				if step.Attachments[k].Label == "" || step.Attachments[k].URL == "" {
					return HelpCenterConfig{}, fmt.Errorf("tutorial %q step attachment label and url are required", tutorial.ID)
				}
				if !isAllowedHelpCenterURL(step.Attachments[k].URL) {
					return HelpCenterConfig{}, fmt.Errorf("invalid url for tutorial %q step attachment", tutorial.ID)
				}
			}
		}
		if tutorial.CodeBlocks == nil {
			tutorial.CodeBlocks = []HelpCenterCodeBlock{}
		}
		for j := range tutorial.CodeBlocks {
			tutorial.CodeBlocks[j].Title = strings.TrimSpace(tutorial.CodeBlocks[j].Title)
			tutorial.CodeBlocks[j].Language = strings.TrimSpace(tutorial.CodeBlocks[j].Language)
		}
		if tutorial.Links == nil {
			tutorial.Links = []HelpCenterLink{}
		}
		for j := range tutorial.Links {
			tutorial.Links[j].Label = strings.TrimSpace(tutorial.Links[j].Label)
			tutorial.Links[j].URL = strings.TrimSpace(tutorial.Links[j].URL)
			if tutorial.Links[j].Label == "" || tutorial.Links[j].URL == "" {
				return HelpCenterConfig{}, fmt.Errorf("tutorial %q link label and url are required", tutorial.ID)
			}
			if !isAllowedHelpCenterURL(tutorial.Links[j].URL) {
				return HelpCenterConfig{}, fmt.Errorf("invalid url for tutorial %q link", tutorial.ID)
			}
		}
		if tutorial.Attachments == nil {
			tutorial.Attachments = []HelpCenterAttachment{}
		}
		for j := range tutorial.Attachments {
			tutorial.Attachments[j].Label = strings.TrimSpace(tutorial.Attachments[j].Label)
			tutorial.Attachments[j].URL = strings.TrimSpace(tutorial.Attachments[j].URL)
			tutorial.Attachments[j].FileName = strings.TrimSpace(tutorial.Attachments[j].FileName)
			if tutorial.Attachments[j].Label == "" || tutorial.Attachments[j].URL == "" {
				return HelpCenterConfig{}, fmt.Errorf("tutorial %q attachment label and url are required", tutorial.ID)
			}
			if !isAllowedHelpCenterURL(tutorial.Attachments[j].URL) {
				return HelpCenterConfig{}, fmt.Errorf("invalid url for tutorial %q attachment", tutorial.ID)
			}
		}
	}
	sort.SliceStable(cfg.Tutorials, func(i, j int) bool {
		if cfg.Tutorials[i].SortOrder == cfg.Tutorials[j].SortOrder {
			return cfg.Tutorials[i].ID < cfg.Tutorials[j].ID
		}
		return cfg.Tutorials[i].SortOrder < cfg.Tutorials[j].SortOrder
	})

	if cfg.FAQs == nil {
		cfg.FAQs = []HelpCenterFAQ{}
	}
	seenFAQs := make(map[string]struct{}, len(cfg.FAQs))
	for i := range cfg.FAQs {
		faq := &cfg.FAQs[i]
		faq.ID = strings.ToLower(strings.TrimSpace(faq.ID))
		if !helpCenterIDPattern.MatchString(faq.ID) {
			return HelpCenterConfig{}, fmt.Errorf("invalid faq id %q", faq.ID)
		}
		if _, ok := seenFAQs[faq.ID]; ok {
			return HelpCenterConfig{}, fmt.Errorf("duplicate faq id %q", faq.ID)
		}
		seenFAQs[faq.ID] = struct{}{}
		faq.Question = strings.TrimSpace(faq.Question)
		faq.AnswerMD = strings.TrimSpace(faq.AnswerMD)
		if faq.Enabled && (faq.Question == "" || faq.AnswerMD == "") {
			return HelpCenterConfig{}, fmt.Errorf("faq %q question and answer are required", faq.ID)
		}
		tags := faq.Tags[:0]
		seenTags := map[string]struct{}{}
		for _, tag := range faq.Tags {
			tag = strings.TrimSpace(tag)
			if tag == "" {
				continue
			}
			if _, ok := seenTags[tag]; ok {
				continue
			}
			seenTags[tag] = struct{}{}
			tags = append(tags, tag)
		}
		faq.Tags = tags
	}
	sort.SliceStable(cfg.FAQs, func(i, j int) bool {
		if cfg.FAQs[i].SortOrder == cfg.FAQs[j].SortOrder {
			return cfg.FAQs[i].ID < cfg.FAQs[j].ID
		}
		return cfg.FAQs[i].SortOrder < cfg.FAQs[j].SortOrder
	})

	return cfg, nil
}

func (s *HelpCenterService) GetDraft(ctx context.Context) (HelpCenterConfig, error) {
	return s.getConfig(ctx, SettingKeyHelpCenterDraftConfig)
}

func (s *HelpCenterService) SaveDraft(ctx context.Context, cfg HelpCenterConfig) error {
	normalized, err := NormalizeHelpCenterConfig(cfg)
	if err != nil {
		return err
	}
	return s.saveConfig(ctx, SettingKeyHelpCenterDraftConfig, normalized)
}

func (s *HelpCenterService) GetPublished(ctx context.Context) (HelpCenterConfig, error) {
	return s.getConfig(ctx, SettingKeyHelpCenterPublishedConfig)
}

func (s *HelpCenterService) PublishDraft(ctx context.Context) (HelpCenterConfig, error) {
	draft, err := s.GetDraft(ctx)
	if err != nil {
		return HelpCenterConfig{}, err
	}
	normalized, err := NormalizeHelpCenterConfig(draft)
	if err != nil {
		return HelpCenterConfig{}, err
	}
	if err := s.saveConfig(ctx, SettingKeyHelpCenterPublishedConfig, normalized); err != nil {
		return HelpCenterConfig{}, err
	}
	return normalized, nil
}

func (s *HelpCenterService) getConfig(ctx context.Context, key string) (HelpCenterConfig, error) {
	if s == nil || s.settingRepo == nil {
		return normalizedDefaultHelpCenterConfig(), nil
	}
	value, err := s.settingRepo.GetValue(ctx, key)
	if err != nil {
		if err == ErrSettingNotFound {
			return normalizedDefaultHelpCenterConfig(), nil
		}
		return HelpCenterConfig{}, err
	}
	var cfg HelpCenterConfig
	if err := json.Unmarshal([]byte(value), &cfg); err != nil {
		return normalizedDefaultHelpCenterConfig(), nil
	}
	normalized, err := NormalizeHelpCenterConfig(cfg)
	if err != nil {
		return normalizedDefaultHelpCenterConfig(), nil
	}
	return normalized, nil
}

func (s *HelpCenterService) saveConfig(ctx context.Context, key string, cfg HelpCenterConfig) error {
	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	return s.settingRepo.Set(ctx, key, string(data))
}

func isAllowedHelpCenterURL(raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return true
	}
	if strings.HasPrefix(raw, "/") && !strings.HasPrefix(raw, "//") {
		return true
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}
