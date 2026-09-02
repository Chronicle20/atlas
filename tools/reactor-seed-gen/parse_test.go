package main

import (
	"os"
	"strings"
	"testing"
)

func TestParseInventory(t *testing.T) {
	b, err := os.ReadFile("testdata/inventory-fixture.md")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	scripts, err := parseInventory(b)
	if err != nil {
		t.Fatalf("parseInventory: %v", err)
	}

	t.Run("count", func(t *testing.T) {
		if len(scripts) != 6 {
			t.Fatalf("got %d scripts, want 6", len(scripts))
		}
	})

	t.Run("order preserved", func(t *testing.T) {
		want := []string{"1002008", "1012000", "2119000", "2612004", "9018000", "2511000"}
		for i, id := range want {
			if scripts[i].Id != id {
				t.Errorf("scripts[%d].Id = %q, want %q", i, scripts[i].Id, id)
			}
		}
	})

	t.Run("no comment", func(t *testing.T) {
		if scripts[0].Comment != "" {
			t.Errorf("scripts[0].Comment = %q, want empty", scripts[0].Comment)
		}
	})

	t.Run("comment captured", func(t *testing.T) {
		c := scripts[1].Comment
		if !strings.HasPrefix(c, "@Author Lerk") {
			t.Errorf("scripts[1].Comment = %q, want prefix %q", c, "@Author Lerk")
		}
		if !strings.HasSuffix(c, "www.gnu.org/licenses/>.") {
			t.Errorf("scripts[1].Comment = %q, want suffix %q", c, "www.gnu.org/licenses/>.")
		}
	})

	t.Run("act only", func(t *testing.T) {
		if scripts[0].HitBody != "" {
			t.Errorf("scripts[0].HitBody = %q, want empty", scripts[0].HitBody)
		}
		if !strings.Contains(scripts[0].ActBody, "rm.dropItems();") {
			t.Errorf("scripts[0].ActBody = %q, want to contain %q", scripts[0].ActBody, "rm.dropItems();")
		}
	})

	t.Run("hit and empty act", func(t *testing.T) {
		if !strings.Contains(scripts[2].HitBody, "getState() !== 0") {
			t.Errorf("scripts[2].HitBody = %q, want to contain %q", scripts[2].HitBody, "getState() !== 0")
		}
		if !strings.Contains(scripts[2].HitBody, "weakenAreaBoss") {
			t.Errorf("scripts[2].HitBody = %q, want to contain %q", scripts[2].HitBody, "weakenAreaBoss")
		}
		if scripts[2].ActBody != "" {
			t.Errorf("scripts[2].ActBody = %q, want empty", scripts[2].ActBody)
		}
	})

	t.Run("single-line empty act", func(t *testing.T) {
		if !strings.Contains(scripts[3].HitBody, "rm.sprayItems();") {
			t.Errorf("scripts[3].HitBody = %q, want to contain %q", scripts[3].HitBody, "rm.sprayItems();")
		}
		if scripts[3].ActBody != "" {
			t.Errorf("scripts[3].ActBody = %q, want empty", scripts[3].ActBody)
		}
	})

	t.Run("empty act only", func(t *testing.T) {
		if scripts[4].HitBody != "" {
			t.Errorf("scripts[4].HitBody = %q, want empty", scripts[4].HitBody)
		}
		if scripts[4].ActBody != "" {
			t.Errorf("scripts[4].ActBody = %q, want empty", scripts[4].ActBody)
		}
	})

	t.Run("multi-statement act", func(t *testing.T) {
		lines := 0
		for _, line := range strings.Split(scripts[5].ActBody, "\n") {
			if strings.TrimSpace(line) != "" {
				lines++
			}
		}
		// The real inventory's 2511000 act body has 6 statement lines, not 4
		// as the brief's table describes; the fixture is copied verbatim from
		// docs/tasks/task-291-reactor-tier1-conversion/tier1-inventory.md and
		// this assertion matches that ground truth.
		if lines != 6 {
			t.Errorf("scripts[5].ActBody has %d non-empty lines, want 6:\n%s", lines, scripts[5].ActBody)
		}
	})
}

func TestParseInventory_RealFile(t *testing.T) {
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

	seen := make(map[string]bool, len(scripts))
	for _, s := range scripts {
		if s.Id == "" {
			t.Fatalf("script has empty Id")
		}
		if seen[s.Id] {
			t.Fatalf("duplicate Id %q", s.Id)
		}
		seen[s.Id] = true
	}
}
