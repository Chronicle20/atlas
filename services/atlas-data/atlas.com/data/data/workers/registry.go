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
