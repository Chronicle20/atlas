package monster

// Classified is the three-way partition of a map's monster life entries.
//
// The partition is computed once, in memory, from a single GetSpawnPoints
// drain. Two filtered providers would mean two paginated HTTP fetches of the
// same atlas-data endpoint per field initialization.
type Classified struct {
	Recurring []SpawnPoint // MobTime >= 0, not hidden
	OneTime   []SpawnPoint // MobTime <  0, not hidden
	Hidden    []SpawnPoint // Hide == true, either MobTime
}

// Classify partitions points into recurring, one-time and hidden buckets,
// preserving input order within each bucket.
//
// A hidden point is excluded from spawning entirely (FR-1.4), so the Hide
// test comes first. The one-time predicate is MobTime < 0 rather than
// MobTime == -1: -1 is the only negative value in the GMS 83.1 dataset, but
// an unexpected -2 must behave as one-time rather than falling through to
// the recurring path (FR-1.2).
func Classify(points []SpawnPoint) Classified {
	var c Classified
	for _, p := range points {
		switch {
		case p.Hide:
			c.Hidden = append(c.Hidden, p)
		case p.MobTime < 0:
			c.OneTime = append(c.OneTime, p)
		default:
			c.Recurring = append(c.Recurring, p)
		}
	}
	return c
}
