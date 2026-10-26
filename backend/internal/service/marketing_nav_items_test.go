//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestMarketingNavItemsSemantics locks the marketing navbar semantics, which
// differ from user_menu_items: missing/empty/invalid input falls back to all
// four entries, while an explicit empty JSON array is preserved to hide
// everything. Order is fixed (models, docs, resources, partner) and unknown
// items are dropped.
func TestMarketingNavItemsSemantics(t *testing.T) {
	allItems := []string{"models", "docs", "resources", "partner"}

	tests := []struct {
		name      string
		raw       string
		wantItems []string
		wantJSON  string
	}{
		{
			name:      "empty string falls back to all items",
			raw:       "",
			wantItems: allItems,
			wantJSON:  `["models","docs","resources","partner"]`,
		},
		{
			name:      "whitespace-only falls back to all items",
			raw:       "   ",
			wantItems: allItems,
			wantJSON:  `["models","docs","resources","partner"]`,
		},
		{
			name:      "invalid json falls back to all items",
			raw:       "not-json",
			wantItems: allItems,
			wantJSON:  `["models","docs","resources","partner"]`,
		},
		{
			name:      "explicit empty array hides all (opposite of user_menu_items)",
			raw:       `[]`,
			wantItems: []string{},
			wantJSON:  `[]`,
		},
		{
			name:      "out of order with unknown items filters and orders",
			raw:       `["partner","bogus","docs","models","models",""]`,
			wantItems: []string{"models", "docs", "partner"},
			wantJSON:  `["models","docs","partner"]`,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.wantItems, ParseMarketingNavItems(tc.raw))
			require.Equal(t, tc.wantJSON, NormalizeMarketingNavItemsJSON(tc.raw))
		})
	}

	// Round-trip: an admin selecting the empty set must not be re-expanded by
	// a subsequent normalize of the stored "[]".
	hidden := MarketingNavItemsJSON([]string{})
	require.Equal(t, `[]`, hidden)
	require.Equal(t, `[]`, NormalizeMarketingNavItemsJSON(hidden))
	require.Empty(t, ParseMarketingNavItems(hidden))
}
