package workers

// Registered is the canonical worker list. Order matches design §3.8 + plan §8.4.
var Registered = []Worker{
	Item{},
	Mob{},
	Npc{},
	Reactor{},
	Skill{},
	Quest{},
	String{},
	Map{},
	Character{},
	UI{},
	Commodity{},
}

// RegisteredNames returns the Name() of every registered worker, in Registered
// order. This is the denominator for ingest run-progress records: consumers
// take a plain []string so they need no dependency on this package's WZ/gorm/
// minio graph, and adding a worker widens the denominator with no second edit.
func RegisteredNames() []string {
	out := make([]string, 0, len(Registered))
	for _, w := range Registered {
		out = append(out, w.Name())
	}
	return out
}
