package minioreconcile

import "testing"

func TestToRequest_DefaultsMinAgeHours(t *testing.T) {
	got := ReconcileInputModel{KeepTenantIDs: []string{"x"}, MinAgeHours: 0, DryRun: true}.ToRequest()
	if got.MinAgeHours != defaultMinAgeHours {
		t.Errorf("MinAgeHours=0 -> got %d, want default %d", got.MinAgeHours, defaultMinAgeHours)
	}

	got = ReconcileInputModel{KeepTenantIDs: []string{"x"}, MinAgeHours: 24, DryRun: true}.ToRequest()
	if got.MinAgeHours != 24 {
		t.Errorf("MinAgeHours=24 -> got %d, want 24 (passthrough)", got.MinAgeHours)
	}
}
