package formula

import (
	"encoding/csv"
	"os"
	"strconv"
	"testing"
)

// TestArchiveCorpus replays every distinct (expression, level) pair found in
// the GMS v95.1 Skill.wz common nodes (md5 2d77583108eb928b65f2904196a894ef).
// The expected column was produced by this evaluator AND cross-checked against
// a standard-precedence reference implementation; every disagreement was
// adjudicated against the client (design §10.3, docs/tasks/task-192-.../
// archive-census.md). The 163 MB archive is not committed — this corpus is.
func TestArchiveCorpus(t *testing.T) {
	f, err := os.Open("testdata/common_corpus.csv")
	if err != nil {
		t.Fatalf("open corpus: %v", err)
	}
	defer func() { _ = f.Close() }()

	r := csv.NewReader(f)
	records, err := r.ReadAll()
	if err != nil {
		t.Fatalf("read corpus: %v", err)
	}
	if len(records) < 2 {
		t.Fatal("corpus is empty")
	}

	cache := map[string]Expr{}
	for i, rec := range records[1:] {
		if len(rec) != 3 {
			t.Fatalf("row %d has %d fields, want 3", i+2, len(rec))
		}
		src, levelText, wantText := rec[0], rec[1], rec[2]
		level, err := strconv.Atoi(levelText)
		if err != nil {
			t.Fatalf("row %d: bad level %q", i+2, levelText)
		}
		want, err := strconv.ParseInt(wantText, 10, 64)
		if err != nil {
			t.Fatalf("row %d: bad expected %q", i+2, wantText)
		}
		e, ok := cache[src]
		if !ok {
			e, err = Parse(src)
			if err != nil {
				t.Fatalf("row %d: Parse(%q) error = %v", i+2, src, err)
			}
			cache[src] = e
		}
		got, err := e.Evaluate(level)
		if err != nil {
			t.Fatalf("row %d: Evaluate(%q, %d) error = %v", i+2, src, level, err)
		}
		if got != want {
			t.Fatalf("row %d: Parse(%q).Evaluate(%d) = %d, want %d", i+2, src, level, got, want)
		}
	}
}
