// Command mksnapshot turns a raw drained id-set into a canonical, hash-pinned
// wzsnapshot file.
//
// It exists so an FR-0-style re-drain is reproducible rather than a hand
// edit: wzsnapshot.LoadSnapshot recomputes the sha256 of the persisted
// arrays on every load and fails loudly on drift, so the `hash` field must
// be computed from the exact arrays that get written. Doing that by hand is
// how a snapshot silently rots.
//
// Usage:
//
//	tools/wzsnapshot-drain.sh <tenant> GMS 95 1 \
//	  | go run ./wzsnapshot/cmd/mksnapshot > wzsnapshot/gms_95_1.json
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/Chronicle20/atlas/libs/atlas-constants/gen/wzsnapshot"
)

// snapshot is the on-disk shape, field-for-field and order-for-order
// identical to wzsnapshot's private snapshotFile.
type snapshot struct {
	Region string   `json:"region"`
	Major  uint16   `json:"major"`
	Minor  uint16   `json:"minor"`
	Hash   string   `json:"hash"`
	Skills []uint32 `json:"skills"`
	Jobs   []uint16 `json:"jobs"`
}

// rawInput is the drain output: the same shape minus the hash.
type rawInput struct {
	Region string   `json:"region"`
	Major  uint16   `json:"major"`
	Minor  uint16   `json:"minor"`
	Skills []uint32 `json:"skills"`
	Jobs   []uint16 `json:"jobs"`
}

func canonicalize(r io.Reader) (snapshot, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return snapshot{}, fmt.Errorf("reading input: %w", err)
	}
	var in rawInput
	if err := json.Unmarshal(b, &in); err != nil {
		return snapshot{}, fmt.Errorf("parsing input: %w", err)
	}
	if in.Region == "" || in.Major == 0 {
		return snapshot{}, fmt.Errorf("input is missing region/major")
	}
	skills := sortedUnique32(in.Skills)
	jobs := sortedUnique16(in.Jobs)
	if len(skills) == 0 {
		return snapshot{}, fmt.Errorf("refusing to write %s %d.%d: empty skill set (drain failed?)", in.Region, in.Major, in.Minor)
	}
	if len(jobs) == 0 {
		return snapshot{}, fmt.Errorf("refusing to write %s %d.%d: empty job set (drain failed?)", in.Region, in.Major, in.Minor)
	}
	return snapshot{
		Region: in.Region,
		Major:  in.Major,
		Minor:  in.Minor,
		Hash:   wzsnapshot.HashIds(skills, jobs),
		Skills: skills,
		Jobs:   jobs,
	}, nil
}

func sortedUnique32(in []uint32) []uint32 {
	seen := make(map[uint32]bool, len(in))
	out := make([]uint32, 0, len(in))
	for _, v := range in {
		if seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func sortedUnique16(in []uint16) []uint16 {
	seen := make(map[uint16]bool, len(in))
	out := make([]uint16, 0, len(in))
	for _, v := range in {
		if seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// render emits the snapshot with the same 2-space indentation the existing
// checked-in snapshots use, so a re-drain diffs as data rather than as
// reformatting.
func render(s snapshot) ([]byte, error) {
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}

func main() {
	s, err := canonicalize(os.Stdin)
	if err != nil {
		fmt.Fprintln(os.Stderr, "mksnapshot:", err)
		os.Exit(1)
	}
	b, err := render(s)
	if err != nil {
		fmt.Fprintln(os.Stderr, "mksnapshot:", err)
		os.Exit(1)
	}
	if _, err := os.Stdout.Write(b); err != nil {
		fmt.Fprintln(os.Stderr, "mksnapshot:", err)
		os.Exit(1)
	}
}
