package characterrender

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

type hashFixtureRow struct {
	Tenant       string `json:"tenant"`
	Region       string `json:"region"`
	MajorVersion uint16 `json:"majorVersion"`
	MinorVersion uint16 `json:"minorVersion"`
	Skin         int    `json:"skin"`
	Hair         int    `json:"hair"`
	Face         int    `json:"face"`
	Stance       string `json:"stance"`
	Frame        int    `json:"frame"`
	Resize       int    `json:"resize"`
	Items        []int  `json:"items"`
	Canonical    string `json:"canonical"`
	ExpectedHash string `json:"expectedHash"`
}

type hashFixture struct {
	Rows []hashFixtureRow `json:"rows"`
}

func loadHashFixture(t *testing.T) hashFixture {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "loadout-hashes.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var f hashFixture
	if err := json.Unmarshal(b, &f); err != nil {
		t.Fatalf("parse fixture: %v", err)
	}
	return f
}

func TestCanonicalLoadoutStringMatchesFixture(t *testing.T) {
	f := loadHashFixture(t)
	for _, row := range f.Rows {
		t.Run(row.Tenant+"-"+row.Stance, func(t *testing.T) {
			got := CanonicalLoadoutString(
				row.Tenant, row.Region, row.MajorVersion, row.MinorVersion,
				row.Skin, row.Hair, row.Face, row.Stance, row.Frame, row.Resize,
				row.Items,
			)
			if got != row.Canonical {
				t.Fatalf("canonical mismatch:\n got = %q\nwant = %q", got, row.Canonical)
			}
		})
	}
}
