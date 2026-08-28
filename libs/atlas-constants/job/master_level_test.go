package job

import "testing"

// The GMS and JMS clients spell the 43x branch differently — GMS v95
// @0x47cb90 and GMS v92 @0x479260 compute (job-430)/2, JMS 185 @0x47d347
// computes (job%10)/2 — but they agree on every value in 430..434
// (0,0,1,1,2) and diverge only at job >= 435, which no client defines.
// This ports the GMS form; the cases below pin the agreed values.
func TestClientJobLevel(t *testing.T) {
	for _, c := range []struct {
		name  string
		jobId Id
		want  int
	}{
		{"beginner 0", Id(0), 1},
		{"warrior root 100", Id(100), 1},
		{"magician root 200", Id(200), 1},
		{"Evan beginner 2001", Id(2001), 1},
		{"Evan stage 1 (2200)", Id(2200), 1},
		{"fighter 110", Id(110), 2},
		{"crusader 111", Id(111), 3},
		{"hero 112", Id(112), 4},
		{"tier-5 non-Evan 113", Id(113), 0},
		{"Aran 4th 2112", Id(2112), 4},
		{"Aran tier-5 2113", Id(2113), 0},
		{"Blade Recruit 430", Id(430), 2},
		{"Blade Acolyte 431", Id(431), 2},
		{"Blade Specialist 432", Id(432), 3},
		{"Blade Lord 433", Id(433), 3},
		{"Blade Master 434", Id(434), 4},
		{"Evan stage 2 (2210)", Id(2210), 2},
		{"Evan stage 3 (2211)", Id(2211), 3},
		{"Evan stage 5 (2213)", Id(2213), 5},
		{"Evan stage 8 (2216)", Id(2216), 8},
		{"Evan stage 9 (2217)", Id(2217), 9},
		{"Evan stage 10 (2218)", Id(2218), 10},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := ClientJobLevel(c.jobId); got != c.want {
				t.Errorf("ClientJobLevel(%d) = %d, want %d", c.jobId, got, c.want)
			}
		})
	}
}
