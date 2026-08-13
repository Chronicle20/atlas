package character

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
)

// atlas-character serves the SP table as a comma-AND-SPACE separated string
// ("0, 0, 0, 0, 0, 0, 0, 0, 0, 0"). Sp must parse every entry: strconv.ParseUint
// rejects a leading space, so an untrimmed split silently yields a 1-element
// slice, and RemainingSp then indexes it by skillBook() — 9 for an Evan at job
// 2218 — panicking with "index out of range [9] with length 1" and aborting the
// character-list encode before anything is sent.
func TestSpParsesSpaceSeparatedTable(t *testing.T) {
	m := NewBuilder().SetJobId(2218).SetSp("0, 0, 0, 0, 0, 0, 0, 0, 0, 0").Build()

	if got := len(m.Sp()); got != 10 {
		t.Fatalf("Sp() length: got %d, want 10", got)
	}
}

func TestSpParsesValuesInOrder(t *testing.T) {
	m := NewBuilder().SetJobId(2218).SetSp("1, 2, 3, 4, 5, 6, 7, 8, 9, 10").Build()

	want := []uint16{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	got := m.Sp()
	if len(got) != len(want) {
		t.Fatalf("Sp() length: got %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Sp()[%d]: got %d, want %d", i, got[i], want[i])
		}
	}
}

// RemainingSp reads the per-growth book for Evan jobs 2210..2218 (index 1..9)
// and book 0 for everything else.
func TestRemainingSpPerSkillBook(t *testing.T) {
	sp := "1, 2, 3, 4, 5, 6, 7, 8, 9, 10"
	for _, c := range []struct {
		jobId job.Id
		want  uint16
	}{
		{112, 1},   // Hero — book 0
		{2001, 1},  // Evan beginner — book 0
		{2210, 2},  // Evan 1st growth — book 1
		{2214, 6},  // Evan 5th growth — book 5
		{2218, 10}, // Evan 9th growth — book 9
	} {
		m := NewBuilder().SetJobId(c.jobId).SetSp(sp).Build()
		if got := m.RemainingSp(); got != c.want {
			t.Errorf("job %d: RemainingSp() = %d, want %d", c.jobId, got, c.want)
		}
	}
}

// A short or empty table must not panic — RemainingSp is on the character-list
// encode path, where a panic drops the whole packet.
func TestRemainingSpShortTableDoesNotPanic(t *testing.T) {
	for _, sp := range []string{"", "0", "5", "1, 2, 3"} {
		m := NewBuilder().SetJobId(2218).SetSp(sp).Build()
		if got := m.RemainingSp(); got != 0 {
			t.Errorf("sp %q: RemainingSp() = %d, want 0", sp, got)
		}
	}
}
