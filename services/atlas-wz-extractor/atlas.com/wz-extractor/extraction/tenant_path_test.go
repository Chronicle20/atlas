package extraction

import (
	"path/filepath"
	"testing"

	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
)

func TestTenantPath_HappyPath(t *testing.T) {
	id := uuid.New()
	tt, err := tenant.Create(id, "GMS", 83, 1)
	if err != nil {
		t.Fatalf("unable to create tenant: %v", err)
	}
	want := filepath.Join(id.String(), "GMS", "83.1")
	if got := TenantPath(tt); got != want {
		t.Errorf("TenantPath = %q, want %q", got, want)
	}
}

func TestResolveTenantInputDir(t *testing.T) {
	id := uuid.New()
	tt, err := tenant.Create(id, "KMS", 92, 3)
	if err != nil {
		t.Fatalf("unable to create tenant: %v", err)
	}
	root := "/var/lib/wz"
	want := filepath.Join(root, id.String(), "KMS", "92.3")
	if got := ResolveTenantInputDir(root, tt); got != want {
		t.Errorf("ResolveTenantInputDir = %q, want %q", got, want)
	}
	if got := ResolveTenantOutputDir(root, tt); got != want {
		t.Errorf("ResolveTenantOutputDir = %q, want %q", got, want)
	}
}

func TestTenantPath_UnusualRegion(t *testing.T) {
	cases := []struct {
		region string
		major  uint16
		minor  uint16
	}{
		{"JMS", 200, 0},
		{"TWMS", 5, 123},
		{"SEA", 1, 1},
	}
	for _, c := range cases {
		id := uuid.New()
		tt, err := tenant.Create(id, c.region, c.major, c.minor)
		if err != nil {
			t.Fatalf("unable to create tenant: %v", err)
		}
		got := TenantPath(tt)
		want := filepath.Join(id.String(), c.region, fmtVersion(c.major, c.minor))
		if got != want {
			t.Errorf("region=%s: TenantPath = %q, want %q", c.region, got, want)
		}
	}
}

func fmtVersion(major, minor uint16) string {
	return intStr(int(major)) + "." + intStr(int(minor))
}

func intStr(i int) string {
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	var b [20]byte
	pos := len(b)
	for i > 0 {
		pos--
		b[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		b[pos] = '-'
	}
	return string(b[pos:])
}

func TestTenantKey_IsStable(t *testing.T) {
	id := uuid.New()
	tt1, _ := tenant.Create(id, "GMS", 83, 1)
	tt2, _ := tenant.Create(id, "GMS", 83, 1)
	if TenantKey(tt1) != TenantKey(tt2) {
		t.Errorf("TenantKey not stable across equivalent tenants")
	}
}

func TestTenantKey_DifferentiatesRegion(t *testing.T) {
	id := uuid.New()
	tt1, _ := tenant.Create(id, "GMS", 83, 1)
	tt2, _ := tenant.Create(id, "KMS", 83, 1)
	if TenantKey(tt1) == TenantKey(tt2) {
		t.Errorf("TenantKey should differ between regions")
	}
}
