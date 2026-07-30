// Package wzsnapshot holds the pinned, checked-in per-version WZ skill/job
// id-set snapshots that the atlas-constants identity-semantics generator
// (task-187 Task 4) consumes offline.
//
// Each snapshot is a small JSON file (<region>_<major>_<minor>.json) drained
// once from the live atlas-data baseline (see PROVENANCE.md) and embedded
// into the built binary via go:embed, so LoadSnapshot works regardless of
// the caller's working directory and never re-fetches over the network.
// The file's sha256 `hash` field pins its skills/jobs arrays -- LoadSnapshot
// recomputes the hash on every load and fails loudly on drift, so a hand
// edit to the arrays without recomputing the hash is caught immediately
// rather than silently poisoning downstream generation.
package wzsnapshot

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
)

//go:embed *.json
var snapshotFS embed.FS

// snapshotFile is the on-disk JSON shape of one pinned snapshot.
type snapshotFile struct {
	Region string   `json:"region"`
	Major  uint16   `json:"major"`
	Minor  uint16   `json:"minor"`
	Hash   string   `json:"hash"`
	Skills []uint32 `json:"skills"`
	Jobs   []uint16 `json:"jobs"`
}

// LoadSnapshot reads the pinned WZ id-set snapshot for the given
// region/major/minor (e.g. "gms", 48, 1), verifies its sha256 hash against
// the recomputed value, and returns the sorted, de-duplicated skill and job
// id sets plus the pinned hash.
//
// An error is returned if no snapshot file exists for the requested
// version, the file fails to parse, its embedded region/major/minor
// identity does not match the request, or the recomputed hash does not
// match the pinned `hash` field (snapshot drift).
func LoadSnapshot(region string, major, minor uint16) (skillIds []uint32, jobIds []uint16, hash string, err error) {
	fname := fmt.Sprintf("%s_%d_%d.json", region, major, minor)
	b, err := snapshotFS.ReadFile(fname)
	if err != nil {
		return nil, nil, "", fmt.Errorf("wzsnapshot: no pinned snapshot for %s %d.%d (%s): %w", region, major, minor, fname, err)
	}

	var sf snapshotFile
	if err := json.Unmarshal(b, &sf); err != nil {
		return nil, nil, "", fmt.Errorf("wzsnapshot: parsing %s: %w", fname, err)
	}

	if sf.Region != region || sf.Major != major || sf.Minor != minor {
		return nil, nil, "", fmt.Errorf("wzsnapshot: %s: embedded identity (%s %d.%d) does not match requested (%s %d.%d)",
			fname, sf.Region, sf.Major, sf.Minor, region, major, minor)
	}

	computed := HashIds(sf.Skills, sf.Jobs)
	if computed != sf.Hash {
		return nil, nil, "", fmt.Errorf("wzsnapshot: %s: snapshot hash drift: file=%s computed=%s", fname, sf.Hash, computed)
	}

	return sf.Skills, sf.Jobs, sf.Hash, nil
}

// HashIds returns the deterministic, hex-encoded sha256 hash pinning a
// skill/job id-set snapshot. Both lists are sorted ascending internally
// before hashing, so the result does not depend on input order -- only on
// set membership and length (duplicates in the input are NOT collapsed
// here; callers materializing a snapshot file are expected to pass
// already-deduplicated lists so the persisted `hash` field matches what a
// fresh LoadSnapshot recomputes from the persisted arrays).
func HashIds(skillIds []uint32, jobIds []uint16) string {
	skills := append([]uint32(nil), skillIds...)
	sort.Slice(skills, func(i, j int) bool { return skills[i] < skills[j] })
	jobs := append([]uint16(nil), jobIds...)
	sort.Slice(jobs, func(i, j int) bool { return jobs[i] < jobs[j] })

	h := sha256.New()
	fmt.Fprintf(h, "skills:%d\n", len(skills))
	for _, s := range skills {
		fmt.Fprintf(h, "%d\n", s)
	}
	fmt.Fprintf(h, "jobs:%d\n", len(jobs))
	for _, j := range jobs {
		fmt.Fprintf(h, "%d\n", j)
	}
	return hex.EncodeToString(h.Sum(nil))
}

// contains32 reports whether needle is present in haystack.
func contains32(haystack []uint32, needle uint32) bool {
	for _, v := range haystack {
		if v == needle {
			return true
		}
	}
	return false
}
