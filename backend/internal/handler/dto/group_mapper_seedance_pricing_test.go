package dto

import (
	"github.com/Wei-Shaw/sub2api/internal/service"
	"testing"
)

func TestGroupFromServiceAdminIncludesSeedanceVideoPrices(t *testing.T) {
	group := &service.Group{SeedanceVideoPrices: map[string]map[string]map[string]map[string]float64{"seedance-2.0": {"token": {"480p": {"none": 1, "input": 2}}}}}
	out := GroupFromServiceAdmin(group)
	if out == nil || out.SeedanceVideoPrices["seedance-2.0"]["token"]["480p"]["input"] != 2 {
		t.Fatalf("Seedance prices not mapped: %#v", out)
	}
}
