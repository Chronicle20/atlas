package job

import (
	"testing"
	"time"
)

func TestJobBuilderAndGetters(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	j := NewJobBuilder().
		SetId("job-1").
		SetTenantId("tenant-1").
		SetRegion("GMS").
		SetMajorVersion(83).
		SetMinorVersion(1).
		SetStatus(JobPending).
		SetUnitsTotal(11).
		SetXmlOnly(true).
		SetImagesOnly(false).
		SetCreatedAt(now).
		SetUpdatedAt(now).
		Build()

	if j.Id() != "job-1" {
		t.Fatalf("Id: got %s", j.Id())
	}
	if j.UnitsTotal() != 11 {
		t.Fatalf("UnitsTotal: got %d", j.UnitsTotal())
	}
	if j.XmlOnly() != true || j.ImagesOnly() != false {
		t.Fatalf("flags: %v %v", j.XmlOnly(), j.ImagesOnly())
	}
	if j.Status() != JobPending {
		t.Fatalf("Status: got %s", j.Status())
	}
}

func TestUnitBuilderAndGetters(t *testing.T) {
	u := NewUnitBuilder().
		SetWzFile("Map.wz").
		SetStatus(UnitPending).
		Build()
	if u.WzFile() != "Map.wz" || u.Status() != UnitPending {
		t.Fatalf("Unit fields: %v", u)
	}
}
