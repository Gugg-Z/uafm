//go:build unit

package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func seedanceMatrix(mode string) map[string]map[string]map[string]map[string]float64 {
	return map[string]map[string]map[string]map[string]float64{"video": {mode: {
		"480p": {"none": 1, "input": 2}, "720p": {"none": 3, "input": 4}, "1080p": {"none": 5, "input": 6},
	}}}
}
func TestSeedanceMatrixSelectionAndSnapshot(t *testing.T) {
	for _, mode := range []string{"token", "video", "per_request"} {
		group := &Group{SeedanceVideoPrices: seedanceMatrix(mode)}
		for _, resolution := range []string{"480p", "720p", "1080p"} {
			for _, material := range []bool{false, true} {
				selected, prices, configured, err := seedancePriceSnapshot(group, "video", resolution, material)
				require.NoError(t, err)
				require.True(t, configured)
				require.Equal(t, mode, selected)
				kind := "none"
				if material {
					kind = "input"
				}
				require.Equal(t, group.SeedanceVideoPrices["video"][mode][resolution][kind], prices[resolution])
			}
		}
		selected, prices, _, err := seedancePriceSnapshot(group, "video", "720p", true)
		require.NoError(t, err)
		group.SeedanceVideoPrices["video"][mode]["720p"]["input"] = 99
		require.Equal(t, 4.0, prices["720p"])
		svc := &OpenAIGatewayService{}
		require.Equal(t, selected, svc.ResolveSeedanceBillingMode(context.Background(), &APIKey{Group: group}, "video"))
		_, _, configured, err := seedancePriceSnapshot(group, "other", "720p", false)
		require.NoError(t, err)
		require.False(t, configured)
	}
}
func TestSeedanceMatrixValidation(t *testing.T) {
	require.NoError(t, ValidateSeedanceVideoPrices(seedanceMatrix("token")))
	for _, p := range []float64{-1, math.NaN(), math.Inf(1)} {
		matrix := seedanceMatrix("video")
		matrix["video"]["video"]["720p"]["none"] = p
		require.Error(t, ValidateSeedanceVideoPrices(matrix))
	}
	matrix := seedanceMatrix("video")
	matrix["video"]["token"] = matrix["video"]["video"]
	require.Error(t, ValidateSeedanceVideoPrices(matrix))
	matrix = seedanceMatrix("token")
	delete(matrix["video"]["token"]["480p"], "input")
	require.NoError(t, ValidateSeedanceVideoPrices(matrix))
	_, _, configured, err := seedancePriceSnapshot(&Group{SeedanceVideoPrices: matrix}, "video", "480p", true)
	require.True(t, configured)
	require.Error(t, err)
	matrix = seedanceMatrix("token")
	matrix["video"]["token"]["1440p"] = map[string]float64{"none": 7}
	require.NoError(t, ValidateSeedanceVideoPrices(matrix))
	selected, prices, configured, err := seedancePriceSnapshot(&Group{SeedanceVideoPrices: matrix}, "video", "1440p", false)
	require.NoError(t, err)
	require.True(t, configured)
	require.Equal(t, "token", selected)
	require.Equal(t, 7.0, prices["1440p"])
	clone := CloneSeedanceVideoPrices(matrix)
	clone["video"]["token"]["480p"]["input"] = 10
	require.Equal(t, 2.0, matrix["video"]["token"]["480p"]["input"])
}
func TestSeedanceMatrixMaterial(t *testing.T) {
	for _, kind := range []string{"image_url", "video_url", "audio_url"} {
		info, err := ParseSeedanceRequest([]byte(fmt.Sprintf(`{"model":"video","content":[{"type":"%s","%s":{"url":"https://example.com/input"}}]}`, kind, kind)))
		require.NoError(t, err)
		require.True(t, info.SeedanceHasMaterial)
		_, err = ParseSeedanceRequest([]byte(fmt.Sprintf(`{"model":"video","content":[{"type":"%s","%s":{"url":""}}]}`, kind, kind)))
		require.Error(t, err)
	}
	info, err := ParseSeedanceRequest([]byte(`{"model":"video","prompt":"waves","generate_audio":true}`))
	require.NoError(t, err)
	require.False(t, info.SeedanceHasMaterial)
}
func TestSeedanceMatrixSettlement(t *testing.T) {
	for _, mode := range []string{"token", "video", "per_request"} {
		t.Run(mode, func(t *testing.T) {
			s, store, billing, logs := videoTaskTestService(t)
			task := store.task
			task.SeedanceMatrix = true
			task.Mode = mode
			task.Prices = map[string]float64{"720p": 4}
			// Snapshot survives repository JSON round trips and later group edits.
			task.UnitPrice = new(float64)
			*task.UnitPrice = 4
			b, err := json.Marshal(task)
			require.NoError(t, err)
			require.NoError(t, json.Unmarshal(b, task))
			task.Group.SeedanceVideoPrices = seedanceMatrix("per_request")
			s.httpUpstream = &grokMediaContentUpstreamStub{response: grokMediaContentStatusResponse(`{"status":"succeeded","resolution":"720p","actualDuration":2.5,"usage":{"completion_tokens":250000}}`)}
			result, err := s.PollVideoTask(context.Background(), task.ID)
			require.NoError(t, err)
			require.True(t, result.Settled)
			want := 4.0
			if mode == "token" {
				want = 1
			}
			if mode == "video" {
				want = 10
			}
			cost, err := videoTaskCost(result, result.Result)
			require.NoError(t, err)
			require.Equal(t, want, cost.TotalCost)
			require.Equal(t, want*2, cost.ActualCost)
			require.InDelta(t, want*2, billing.lastCmd.BalanceCost, 1e-8)
			require.Equal(t, want*2, logs.lastLog.ActualCost)
			require.Equal(t, 1, billing.calls)
			_, err = s.PollVideoTask(context.Background(), task.ID)
			require.NoError(t, err)
			require.Equal(t, 1, billing.calls)
			free := 0.0
			result.UnitPrice = &free
			cost, err = videoTaskCost(result, result.Result)
			require.NoError(t, err)
			require.Zero(t, cost.ActualCost)
		})
	}
}

func TestSeedanceMatrixPrepareUsesAuthenticatedGroup(t *testing.T) {
	for _, mode := range []string{"token", "video", "per_request"} {
		s, _, _, _ := videoTaskTestService(t)
		gid := int64(3)
		key := &APIKey{ID: 2, User: &User{ID: 1}, GroupID: &gid, Group: &Group{ID: gid, RateMultiplier: 2, SubscriptionType: "subscription", SeedanceVideoPrices: seedanceMatrix(mode)}}
		task, err := s.PrepareVideoTask(context.Background(), key, nil, SeedanceEndpointCreate, "video", "720p", "token", "", "", true)
		require.NoError(t, err)
		require.True(t, task.SeedanceMatrix)
		require.True(t, task.HasMaterial)
		require.Equal(t, mode, task.Mode)
		require.Equal(t, 4.0, *task.UnitPrice)
		key.Group.SeedanceVideoPrices["video"][mode]["720p"]["input"] = 99
		require.Equal(t, 4.0, task.Prices["720p"])
	}
}

func TestSeedanceMatrixRejectsUnpricedRequestedResolution(t *testing.T) {
	s, _, _, _ := videoTaskTestService(t)
	gid := int64(3)
	matrix := map[string]map[string]map[string]map[string]float64{"video": {"video": {"480p": {"none": 1, "input": 2}}}}
	key := &APIKey{ID: 2, User: &User{ID: 1}, GroupID: &gid, Group: &Group{ID: gid, SeedanceVideoPrices: matrix}}
	_, err := s.PrepareVideoTask(context.Background(), key, nil, SeedanceEndpointCreate, "video", "720p", "video", "", "", false)
	require.ErrorContains(t, err, "unpriced Seedance resolution")
}
