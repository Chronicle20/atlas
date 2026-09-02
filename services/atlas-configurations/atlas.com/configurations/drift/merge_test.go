package drift

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestValidateSections(t *testing.T) {
	tests := []struct {
		name      string
		sections  []string
		wantErr   bool
		wantNamed string
	}{
		{name: "NilIsWholeDocument", sections: nil, wantErr: false},
		{name: "EmptyIsWholeDocument", sections: []string{}, wantErr: false},
		{name: "EveryNamedSection", sections: []string{"properties", "socket", "characters", "npcs", "cashShop", "mapleLife"}, wantErr: false},
		{name: "RejectsWorlds", sections: []string{"worlds"}, wantErr: true, wantNamed: "worlds"},
		{name: "RejectsDiagnostics", sections: []string{"diagnostics"}, wantErr: true, wantNamed: "diagnostics"},
		{name: "RejectsRegion", sections: []string{"region"}, wantErr: true, wantNamed: "region"},
		{name: "RejectsId", sections: []string{"id"}, wantErr: true, wantNamed: "id"},
		{name: "RejectsEnvironment", sections: []string{"environment"}, wantErr: true, wantNamed: "environment"},
		{name: "RejectsUsesPinAlias", sections: []string{"usesPin"}, wantErr: true, wantNamed: "usesPin"},
		{name: "RejectsGibberish", sections: []string{"socket", "nonsense"}, wantErr: true, wantNamed: "nonsense"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateSections(tc.sections)
			if !tc.wantErr {
				if err != nil {
					t.Fatalf("ValidateSections(%v) = %v, want nil", tc.sections, err)
				}
				return
			}
			if !errors.Is(err, ErrUnknownSection) {
				t.Fatalf("ValidateSections(%v) = %v, want errors.Is ErrUnknownSection", tc.sections, err)
			}
			if !strings.Contains(err.Error(), tc.wantNamed) {
				t.Errorf("error %q does not contain offending section %q", err.Error(), tc.wantNamed)
			}
		})
	}
}

func TestMerge(t *testing.T) {
	tests := []struct {
		name     string
		stored   string
		base     string
		sections []string
		want     string
	}{
		{
			name:     "EmptySectionsReplacesEverything",
			stored:   `{"usesPin":false,"npcs":[{"npcId":9}],"socket":{"handlers":[{"opCode":2}]}}`,
			base:     `{"usesPin":true,"npcs":[{"npcId":1}],"socket":{"handlers":[{"opCode":1}]}}`,
			sections: nil,
			want:     `{"usesPin":true,"npcs":[{"npcId":1}],"socket":{"handlers":[{"opCode":1}]}}`,
		},
		{
			name:     "NamedSectionOnly",
			stored:   `{"usesPin":false,"npcs":[{"npcId":9}],"socket":{"handlers":[{"opCode":2}]}}`,
			base:     `{"usesPin":true,"npcs":[{"npcId":1}],"socket":{"handlers":[{"opCode":1}]}}`,
			sections: []string{"socket"},
			want:     `{"usesPin":false,"npcs":[{"npcId":9}],"socket":{"handlers":[{"opCode":1}]}}`,
		},
		{
			name:     "ReplacementNotFieldMerge",
			stored:   `{"cashShop":{"commodities":{"hourlyExpirations":[{"templateId":9,"hours":2}]},"surprise":{"boxTemplateIds":[7]}}}`,
			base:     `{"cashShop":{"commodities":{"hourlyExpirations":[{"templateId":1,"hours":1}]}}}`,
			sections: []string{"cashShop"},
			want:     `{"cashShop":{"commodities":{"hourlyExpirations":[{"templateId":1,"hours":1}]}}}`,
		},
		{
			name:     "PropertiesReplacesResidualScalars",
			stored:   `{"usesPin":false,"npcs":[{"npcId":9}]}`,
			base:     `{"usesPin":true,"npcs":[{"npcId":1}]}`,
			sections: []string{"properties"},
			want:     `{"usesPin":true,"npcs":[{"npcId":9}]}`,
		},
		{
			name:     "PropertiesAddsKeyPresentOnlyInBase",
			stored:   `{"usesPin":false}`,
			base:     `{"usesPin":true,"enableAutoRegister":true}`,
			sections: []string{"properties"},
			want:     `{"usesPin":true,"enableAutoRegister":true}`,
		},
		{
			name:     "PropertiesRemovesKeyPresentOnlyInStored",
			stored:   `{"usesPin":false,"legacyFlag":true}`,
			base:     `{"usesPin":true}`,
			sections: []string{"properties"},
			want:     `{"usesPin":true}`,
		},
		{
			name:     "NamedSectionAbsentInBaseIsRemoved",
			stored:   `{"usesPin":true,"npcs":[{"npcId":9}]}`,
			base:     `{"usesPin":true}`,
			sections: []string{"npcs"},
			want:     `{"usesPin":true}`,
		},
		{
			name:     "UnrequestedSectionsUntouched",
			stored:   `{"usesPin":false,"mapleLife":{"looks":[{"gender":1}]}}`,
			base:     `{"usesPin":true,"mapleLife":{"looks":[{"gender":0}]}}`,
			sections: []string{"properties"},
			want:     `{"usesPin":true,"mapleLife":{"looks":[{"gender":1}]}}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stored := canon(t, tc.stored)
			base := canon(t, tc.base)
			want := canon(t, tc.want)

			got, err := Merge(stored, base, tc.sections)
			if err != nil {
				t.Fatalf("Merge: %v", err)
			}

			gotJSON, err := json.Marshal(got)
			if err != nil {
				t.Fatalf("json.Marshal(got): %v", err)
			}
			wantJSON, err := json.Marshal(want)
			if err != nil {
				t.Fatalf("json.Marshal(want): %v", err)
			}

			if string(gotJSON) != string(wantJSON) {
				t.Errorf("Merge() = %s, want %s", gotJSON, wantJSON)
			}
		})
	}
}

func TestMergeIsIdempotent(t *testing.T) {
	stored := canon(t, `{"usesPin":false,"npcs":[{"npcId":9}],"socket":{"handlers":[{"opCode":2}]}}`)
	base := canon(t, `{"usesPin":true,"npcs":[{"npcId":1}],"socket":{"handlers":[{"opCode":1}]}}`)

	first, err := Merge(stored, base, nil)
	if err != nil {
		t.Fatalf("Merge (first): %v", err)
	}
	second, err := Merge(first, base, nil)
	if err != nil {
		t.Fatalf("Merge (second): %v", err)
	}

	firstJSON, err := json.Marshal(first)
	if err != nil {
		t.Fatalf("json.Marshal(first): %v", err)
	}
	secondJSON, err := json.Marshal(second)
	if err != nil {
		t.Fatalf("json.Marshal(second): %v", err)
	}

	if string(firstJSON) != string(secondJSON) {
		t.Errorf("Merge not idempotent: first %s, second %s", firstJSON, secondJSON)
	}
}

func TestMergeRejectsUnknownSection(t *testing.T) {
	stored := canon(t, `{"usesPin":false}`)
	base := canon(t, `{"usesPin":true}`)

	_, err := Merge(stored, base, []string{"worlds"})
	if !errors.Is(err, ErrUnknownSection) {
		t.Fatalf("Merge with unknown section = %v, want errors.Is ErrUnknownSection", err)
	}
}
