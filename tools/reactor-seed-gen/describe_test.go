package main

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestDescribe(t *testing.T) {
	cases := []struct {
		name     string
		id       string
		comment  string
		want     string
		wantFrom string // override table id, when set the case checks against descriptionOverrides[wantFrom]
	}{
		{
			name:    "full boilerplate then purpose",
			id:      "1021000",
			comment: "This file is part of the HeavenMS MapleStory Server Copyleft (L) 2016 - 2019 RonanLana This program is free software: ... see <http://www.gnu.org/licenses/>. @Author Ronan * * 1021000.js: relic room fail www.gnu.org/licenses/>.",
			want:    "Relic room fail",
		},
		{
			name:    "author then purpose",
			id:      "1012000",
			comment: "@Author Lerk * * 1012000.js: Ellinia Plant - drops meso, tree branches, red pots, and Plant Samples (quest item) www.gnu.org/licenses/>.",
			want:    "Ellinia Plant - drops meso, tree branches, red pots, and Plant Samples (quest item)",
		},
		{
			name:    "purpose tag",
			id:      "9999001",
			comment: "* * @author BubblesDev * @purpose Flower 1 www.gnu.org/licenses/>.",
			want:    "Flower 1",
		},
		{
			name:    "blogspot debris",
			id:      "9999002",
			comment: "* Tombstone in Forest of Dead Trees I MSEA reference: http://mymapleland.blogspot.com/2009/09/kill-lich-at-forest-of-dead-trees-i-to.html www.gnu.org/licenses/>. mymapleland.blogspot.com/2009/09/kill-lich-at-forest-of-dead-trees-i-to.html If the chest is destroyed before Riche, killing him should yield no exp",
		},
		{
			name:     "no comment, override present",
			id:       "1002008",
			comment:  "",
			wantFrom: "1002008",
		},
		{
			name:     "reduces to nothing, override present",
			id:       "8001000",
			comment:  "www.gnu.org/licenses/>.",
			wantFrom: "8001000",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := describe(tc.id, tc.comment)
			if err != nil {
				t.Fatalf("describe(%q, %q): %v", tc.id, tc.comment, err)
			}

			switch tc.name {
			case "blogspot debris":
				if !strings.Contains(got, "Tombstone in Forest of Dead Trees I") {
					t.Errorf("describe() = %q, want to contain %q", got, "Tombstone in Forest of Dead Trees I")
				}
				if strings.Contains(got, "blogspot") {
					t.Errorf("describe() = %q, must not contain %q", got, "blogspot")
				}
				return
			}

			want := tc.want
			if tc.wantFrom != "" {
				var ok bool
				want, ok = descriptionOverrides[tc.wantFrom]
				if !ok {
					t.Fatalf("descriptionOverrides[%q] missing", tc.wantFrom)
				}
			}
			if got != want {
				t.Errorf("describe(%q, %q) = %q, want %q", tc.id, tc.comment, got, want)
			}
		})
	}
}

func TestDescribe_MissingOverrideAborts(t *testing.T) {
	_, err := describe("9999999", "")
	if err == nil {
		t.Fatal("describe: want error for reactor with empty comment and no override, got nil")
	}
	if !strings.Contains(err.Error(), "9999999") {
		t.Errorf("describe error = %q, want it to name reactor %q", err.Error(), "9999999")
	}
}

func TestDescribe_NoBoilerplateLeaks(t *testing.T) {
	b, err := os.ReadFile("../../docs/tasks/task-291-reactor-tier1-conversion/tier1-inventory.md")
	if err != nil {
		t.Fatalf("read real inventory: %v", err)
	}

	scripts, err := parseInventory(b)
	if err != nil {
		t.Fatalf("parseInventory: %v", err)
	}
	if len(scripts) != 159 {
		t.Fatalf("got %d scripts, want 159", len(scripts))
	}

	forbidden := []string{
		"gnu.org",
		"Copyleft",
		"@author",
		"@Author",
		"@purpose",
		"blogspot",
		"WITHOUT ANY WARRANTY",
		"HeavenMS",
		"OdinMS",
		".js:",
		"http",
		"://",
		"  ",
		"@",
	}

	for _, s := range scripts {
		got, err := describe(s.Id, s.Comment)
		if err != nil {
			t.Errorf("describe(%q): %v", s.Id, err)
			continue
		}
		if got == "" {
			t.Errorf("describe(%q) = empty", s.Id)
			continue
		}
		for _, bad := range forbidden {
			if strings.Contains(got, bad) {
				t.Errorf("describe(%q) = %q leaks %q", s.Id, got, bad)
			}
		}
	}
}

// capitalizedWordPattern extracts individual capitalized words (proper-noun
// candidates) from an override description, e.g. "Snow" and "Witch" out of
// "Altar - weakens the Snow Witch".
var capitalizedWordPattern = regexp.MustCompile(`\b[A-Z][a-z]+\b`)

// nonLocativeAllowlist is the short, literal list of capitalized common
// words that legitimately open a description without naming a place, NPC,
// or quest. Every one of these is a sentence-initial common noun/verb, not a
// proper noun, so it is exempt from the "must appear in the corpus" check
// below. Do not add a place, NPC, or quest name to this list — that would
// smuggle an ungrounded proper noun past the audit this test exists to run.
var nonLocativeAllowlist = map[string]bool{
	"Box":    true,
	"Spawns": true,
	"Drops":  true,
	"Sprays": true,
	"Altar":  true,
	"Relic":  true,
}

// TestDescribe_OverridesAreGrounded programmatically audits every entry in
// descriptionOverrides for the defect class two prior reviews found by hand
// and missed by hand: a proper noun (map, NPC, or quest name) that does not
// appear anywhere in the checked-in tier1-inventory.md corpus. For every
// override value, every capitalized word not on nonLocativeAllowlist must
// appear somewhere in the real inventory file, or the override is inventing
// a name the corpus does not support.
func TestDescribe_OverridesAreGrounded(t *testing.T) {
	corpus, err := os.ReadFile("../../docs/tasks/task-291-reactor-tier1-conversion/tier1-inventory.md")
	if err != nil {
		t.Fatalf("read real inventory: %v", err)
	}
	corpusText := string(corpus)

	for id, description := range descriptionOverrides {
		for _, word := range capitalizedWordPattern.FindAllString(description, -1) {
			if nonLocativeAllowlist[word] {
				continue
			}
			if !strings.Contains(corpusText, word) {
				t.Errorf("descriptionOverrides[%q] = %q: proper-noun candidate %q does not appear anywhere in tier1-inventory.md", id, description, word)
			}
		}
	}
}
