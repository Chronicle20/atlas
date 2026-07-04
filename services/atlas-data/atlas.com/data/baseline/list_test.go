package baseline

import (
	"testing"
)

func TestParseDumpKeyRoundTrip(t *testing.T) {
	cases := []struct {
		region string
		major  int
		minor  int
	}{
		{"GMS", 83, 1},
		{"GMS", 84, 1},
		{"JMS", 185, 1},
	}
	for _, c := range cases {
		key := DumpKey(c.region, c.major, c.minor)
		region, major, minor, ok := parseDumpKey(key)
		if !ok {
			t.Fatalf("parseDumpKey(%q) not ok", key)
		}
		if region != c.region || major != c.major || minor != c.minor {
			t.Fatalf("parseDumpKey(%q) = %s/%d.%d", key, region, major, minor)
		}
	}
}

func TestParseDumpKeyRejectsMalformed(t *testing.T) {
	bad := []string{
		"",
		"baseline/regions/GMS/versions/83.1/other.file",
		"baseline/regions/GMS/versions/83.1",
		"baseline/regions/GMS/versions/831/documents.dump",
		"baseline/regions/GMS/versions/x.y/documents.dump",
		"baseline/regions/GMS/versions/83./documents.dump",
		"baseline/regions/GMS/versions/.1/documents.dump",
		"baseline/regions/GMS/versions/-1.2/documents.dump",
		"baseline/regions/GMS/versions/83.-2/documents.dump",
		"baseline/regions//versions/83.1/documents.dump",
		"shared/regions/GMS/versions/83.1/documents.dump",
		"baseline/other/GMS/versions/83.1/documents.dump",
		"baseline/regions/GMS/versions/83.1/extra/documents.dump",
	}
	for _, key := range bad {
		if _, _, _, ok := parseDumpKey(key); ok {
			t.Fatalf("parseDumpKey(%q) unexpectedly ok", key)
		}
	}
}
