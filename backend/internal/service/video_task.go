package service

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

var ErrVideoTaskLimit = errors.New("pending video task limit reached")
var ErrVideoTaskBusy = errors.New("video task query is already in progress")
var ErrVideoTaskStoreUnavailable = errors.New("video task store unavailable")

// VideoTask contains billing context, never upstream credentials or API key secrets.
type VideoTask struct {
	GlobalPendingLimit                   bool `json:"global_pending_limit,omitempty"`
	ID                                   string
	UserID, APIKeyID, GroupID, AccountID int64
	UpstreamID                           string
	Provider                             string
	Mode                                 string
	Model, UpstreamModel, Resolution     string
	Group                                *Group
	Key                                  *APIKey
	Subscription                         *UserSubscription
	UnitPrice                            *float64
	TokenPricing                         *ResolvedPricing
	Prices                               map[string]float64
	Multiplier                           float64
	CreatedAt, Deadline, NextPollAt      time.Time
	LastPolledAt                         time.Time
	Status                               string
	Body                                 json.RawMessage
	Result                               *OpenAIForwardResult
	InboundEndpoint, UpstreamEndpoint    string
	Settled, Closed                      bool
	LastError                            string
	HoldID                               string
	HoldAmount                           float64
	LeaseToken                           string `json:"-"`
}

type VideoTaskRepository interface {
	Create(context.Context, *VideoTask, int) error
	Get(context.Context, string) (*VideoTask, error)
	Find(context.Context, int64, int64, int64, string) (*VideoTask, error)
	Due(context.Context, int) ([]string, error)
	Claim(context.Context, string, string, time.Duration) (bool, error)
	Save(context.Context, *VideoTask) error
	Release(context.Context, string, string) error
}
