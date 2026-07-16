package incubator

// PickWeighted selects a reward proportional to weight. rollFn receives the
// total weight and must return a value in [0, total). Returns false for an
// empty or zero-weight pool.
func PickWeighted(rewards []Reward, rollFn func(total uint32) uint32) (Reward, bool) {
	var total uint32
	for _, r := range rewards {
		total += r.Weight()
	}
	if total == 0 {
		return Reward{}, false
	}
	roll := rollFn(total)
	var acc uint32
	for _, r := range rewards {
		acc += r.Weight()
		if roll < acc {
			return r, true
		}
	}
	return rewards[len(rewards)-1], true
}

// FilterByEgg returns only the rewards configured for the given Pigmy Egg id.
func FilterByEgg(rewards []Reward, eggId uint32) []Reward {
	out := make([]Reward, 0, len(rewards))
	for _, r := range rewards {
		if r.EggId() == eggId {
			out = append(out, r)
		}
	}
	return out
}
