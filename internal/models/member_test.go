package models_test

import (
	"testing"

	"open-library-explorer/internal/models"
)

func TestIsValidMemberTier(t *testing.T) {
	tests := []struct {
		name    string
		tier    string
		isValid bool
	}{
		{"Valid Premium Tier", string(models.TierPremium), true},
		{"Valid Standard Tier", string(models.TierStandard), true},
		{"Invalid Tier", "GOLD", false},
		{"Empty Tier", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := models.IsValidMemberTier(tt.tier); got != tt.isValid {
				t.Errorf("IsValidMemberTier() = %v, want %v", got, tt.isValid)
			}
		})
	}
}
