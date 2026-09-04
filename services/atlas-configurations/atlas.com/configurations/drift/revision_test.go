package drift

import (
	"regexp"
	"testing"
)

func TestSectionsAlwaysCarriesSixKeys(t *testing.T) {
	d := canon(t, "{}")
	sections, err := Sections(d)
	if err != nil {
		t.Fatalf("Sections: %v", err)
	}

	want := []string{"properties", "socket", "characters", "npcs", "cashShop", "mapleLife"}
	if len(sections) != len(want) {
		t.Fatalf("got %d keys, want %d: %v", len(sections), len(want), sections)
	}
	for _, k := range want {
		if _, ok := sections[k]; !ok {
			t.Errorf("Sections missing key %q", k)
		}
	}
}

func TestSectionsIsHexSHA256(t *testing.T) {
	d := canon(t, `{"usesPin":true}`)
	sections, err := Sections(d)
	if err != nil {
		t.Fatalf("Sections: %v", err)
	}

	hexPattern := regexp.MustCompile(`^[0-9a-f]{64}$`)
	for name, v := range sections {
		if len(v) != 64 {
			t.Errorf("section %q: len(%q) = %d, want 64", name, v, len(v))
		}
		if !hexPattern.MatchString(v) {
			t.Errorf("section %q: %q does not match ^[0-9a-f]{64}$", name, v)
		}
	}
}

func TestPropertiesIsDefinedBySubtraction(t *testing.T) {
	tests := []struct {
		name          string
		docA          string
		docB          string
		differSection string
		equalSections []string
	}{
		{
			name:          "UsesPinLandsInProperties",
			docA:          `{"usesPin":true}`,
			docB:          `{"usesPin":false}`,
			differSection: "properties",
			equalSections: []string{"socket", "characters", "npcs", "cashShop", "mapleLife"},
		},
		{
			name:          "UnknownScalarLandsInProperties",
			docA:          `{"usesPin":true,"enableAutoRegister":true}`,
			docB:          `{"usesPin":true}`,
			differSection: "properties",
			equalSections: []string{"socket", "characters", "npcs", "cashShop", "mapleLife"},
		},
		{
			name:          "NamedSectionDoesNotLeakIntoProperties",
			docA:          `{"usesPin":true,"npcs":[{"npcId":1}]}`,
			docB:          `{"usesPin":true,"npcs":[{"npcId":2}]}`,
			differSection: "npcs",
			equalSections: []string{"properties", "socket", "characters", "cashShop", "mapleLife"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			a := canon(t, tc.docA)
			b := canon(t, tc.docB)

			sa, err := Sections(a)
			if err != nil {
				t.Fatalf("Sections(a): %v", err)
			}
			sb, err := Sections(b)
			if err != nil {
				t.Fatalf("Sections(b): %v", err)
			}

			if sa[tc.differSection] == sb[tc.differSection] {
				t.Errorf("section %q: expected hashes to differ, both %q", tc.differSection, sa[tc.differSection])
			}
			for _, name := range tc.equalSections {
				if sa[name] != sb[name] {
					t.Errorf("section %q: expected hashes equal, got %q vs %q", name, sa[name], sb[name])
				}
			}
		})
	}
}

func TestAggregateChangesWithAnySection(t *testing.T) {
	base := canon(t, `{"usesPin":true,"npcs":[{"npcId":1}]}`)
	npcDiff := canon(t, `{"usesPin":true,"npcs":[{"npcId":2}]}`)
	propDiff := canon(t, `{"usesPin":false,"npcs":[{"npcId":1}]}`)

	baseAgg, err := Aggregate(base)
	if err != nil {
		t.Fatalf("Aggregate(base): %v", err)
	}
	npcAgg, err := Aggregate(npcDiff)
	if err != nil {
		t.Fatalf("Aggregate(npcDiff): %v", err)
	}
	propAgg, err := Aggregate(propDiff)
	if err != nil {
		t.Fatalf("Aggregate(propDiff): %v", err)
	}

	if baseAgg == npcAgg {
		t.Errorf("expected aggregate to change with npcs section, got equal: %q", baseAgg)
	}
	if baseAgg == propAgg {
		t.Errorf("expected aggregate to change with properties section, got equal: %q", baseAgg)
	}
}

func TestCompareIsolatesOneSection(t *testing.T) {
	base := canon(t, `{"usesPin":true,"socket":{"handlers":[{"opCode":1}]},"npcs":[{"npcId":1}]}`)
	stored := canon(t, `{"usesPin":true,"socket":{"handlers":[{"opCode":2}]},"npcs":[{"npcId":1}]}`)

	agg, per, err := Compare(base, stored)
	if err != nil {
		t.Fatalf("Compare: %v", err)
	}

	if !agg {
		t.Errorf("agg = false, want true")
	}
	if !per["socket"] {
		t.Errorf("per[socket] = false, want true")
	}
	for _, name := range []string{"properties", "characters", "npcs", "cashShop", "mapleLife"} {
		if per[name] {
			t.Errorf("per[%q] = true, want false", name)
		}
	}
}

func TestCompareIdenticalDocsReportNoDrift(t *testing.T) {
	raw := `{"usesPin":true,"socket":{"handlers":[{"opCode":1}]},"npcs":[{"npcId":1}]}`
	base := canon(t, raw)
	stored := canon(t, raw)

	agg, per, err := Compare(base, stored)
	if err != nil {
		t.Fatalf("Compare: %v", err)
	}

	if agg {
		t.Errorf("agg = true, want false")
	}
	for _, name := range All() {
		if per[name] {
			t.Errorf("per[%q] = true, want false", name)
		}
	}
}
