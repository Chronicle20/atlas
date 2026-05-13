package version

import (
	"testing"

	"github.com/google/uuid"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func mk(region string, major uint16) tenant.Model {
	t, _ := tenant.Create(uuid.New(), region, major, 1)
	return t
}

func TestAtLeast(t *testing.T) {
	if !AtLeast(mk("GMS", 95), 95) {
		t.Error("v95 >= 95")
	}
	if AtLeast(mk("GMS", 83), 95) {
		t.Error("v83 < 95")
	}
}

func TestBetween(t *testing.T) {
	if !Between(mk("GMS", 90), 87, 95) {
		t.Error("90 in [87,95]")
	}
	if Between(mk("GMS", 100), 87, 95) {
		t.Error("100 not in [87,95]")
	}
}

func TestRegionOf(t *testing.T) {
	if RegionOf(mk("GMS", 95)) != GMS {
		t.Error("region GMS")
	}
}
