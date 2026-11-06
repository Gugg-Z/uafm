package service

import (
	"fmt"
	"math"
	"strings"
)

// Each requested model has exactly one active mode in this group. Prices are
// USD per million output tokens, per second, or per successful task.
func ValidateSeedanceVideoPrices(prices map[string]map[string]map[string]map[string]float64) error {
	for model, modes := range prices {
		if strings.TrimSpace(model) == "" || model != strings.TrimSpace(model) {
			return fmt.Errorf("invalid Seedance model name")
		}
		if len(modes) != 1 {
			return fmt.Errorf("Seedance %s must have exactly one billing mode", model)
		}
		for mode, tiers := range modes {
			if mode != "token" && mode != "video" && mode != "per_request" {
				return fmt.Errorf("invalid Seedance billing mode: %s", mode)
			}
			if len(tiers) == 0 {
				return fmt.Errorf("Seedance %s requires at least one resolution", model)
			}
			for resolution, material := range tiers {
				if strings.TrimSpace(resolution) == "" || len(material) == 0 {
					return fmt.Errorf("Seedance %s / %s requires at least one price", model, resolution)
				}
				for kind, p := range material {
					if kind != "none" && kind != "input" || p < 0 || math.IsNaN(p) || math.IsInf(p, 0) {
						return fmt.Errorf("invalid Seedance price: %s / %s / %s", model, resolution, kind)
					}
				}
			}
		}
	}
	return nil
}

func CloneSeedanceVideoPrices(value map[string]map[string]map[string]map[string]float64) map[string]map[string]map[string]map[string]float64 {
	if value == nil {
		return nil
	}
	copy := make(map[string]map[string]map[string]map[string]float64, len(value))
	for model, modes := range value {
		copy[model] = make(map[string]map[string]map[string]float64, len(modes))
		for mode, tiers := range modes {
			copy[model][mode] = make(map[string]map[string]float64, len(tiers))
			for resolution, materials := range tiers {
				copy[model][mode][resolution] = make(map[string]float64, len(materials))
				for kind, price := range materials {
					copy[model][mode][resolution][kind] = price
				}
			}
		}
	}
	return copy
}

func seedancePriceSnapshot(group *Group, model, resolution string, hasMaterial bool) (string, map[string]float64, bool, error) {
	if group == nil {
		return "", nil, false, nil
	}
	modes, ok := group.SeedanceVideoPrices[model]
	if !ok {
		return "", nil, false, nil
	}
	if err := ValidateSeedanceVideoPrices(map[string]map[string]map[string]map[string]float64{model: modes}); err != nil {
		return "", nil, true, err
	}
	kind := "none"
	if hasMaterial {
		kind = "input"
	}
	for mode, tiers := range modes {
		material, tierExists := tiers[resolution]
		if !tierExists {
			return "", nil, true, fmt.Errorf("unpriced Seedance resolution: %s", resolution)
		}
		price, priceExists := material[kind]
		if !priceExists {
			return "", nil, true, fmt.Errorf("unpriced Seedance material type: %s / %s", resolution, kind)
		}
		filtered := map[string]float64{resolution: price}
		return mode, filtered, true, nil
	}
	return "", nil, true, fmt.Errorf("missing Seedance billing mode")
}
