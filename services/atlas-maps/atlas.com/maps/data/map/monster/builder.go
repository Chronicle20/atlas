package monster

// Builder constructs a SpawnPoint via fluent setters.
//
// The struct's fields stay exported: SpawnPoint is constructed as a literal
// across the service (rest.go's Extract, map/monster's toStored/fromStored)
// and in dozens of tests. This Builder is added alongside the existing
// exported fields, not in place of them.
type Builder struct {
	id       uint32
	template uint32
	mobTime  int32
	team     int8
	cy       int16
	f        uint32
	fh       int16
	rx0      int16
	rx1      int16
	x        int16
	y        int16
	hide     bool
}

// NewBuilder constructs a Builder with zero-valued fields.
func NewBuilder() *Builder {
	return &Builder{}
}

// SetId sets the spawn point's unique identifier.
func (b *Builder) SetId(id uint32) *Builder {
	b.id = id
	return b
}

// SetTemplate sets the monster template ID to spawn.
func (b *Builder) SetTemplate(template uint32) *Builder {
	b.template = template
	return b
}

// SetMobTime sets the time-related spawn behavior.
func (b *Builder) SetMobTime(mobTime int32) *Builder {
	b.mobTime = mobTime
	return b
}

// SetTeam sets the team assignment for spawned monsters.
func (b *Builder) SetTeam(team int8) *Builder {
	b.team = team
	return b
}

// SetCy sets the Y coordinate for spawn behavior.
func (b *Builder) SetCy(cy int16) *Builder {
	b.cy = cy
	return b
}

// SetF sets the flags for spawn behavior.
func (b *Builder) SetF(f uint32) *Builder {
	b.f = f
	return b
}

// SetFh sets the foothold for spawned monsters.
func (b *Builder) SetFh(fh int16) *Builder {
	b.fh = fh
	return b
}

// SetRx0 sets the left boundary of the spawn area.
func (b *Builder) SetRx0(rx0 int16) *Builder {
	b.rx0 = rx0
	return b
}

// SetRx1 sets the right boundary of the spawn area.
func (b *Builder) SetRx1(rx1 int16) *Builder {
	b.rx1 = rx1
	return b
}

// SetX sets the X coordinate for spawn position.
func (b *Builder) SetX(x int16) *Builder {
	b.x = x
	return b
}

// SetY sets the Y coordinate for spawn position.
func (b *Builder) SetY(y int16) *Builder {
	b.y = y
	return b
}

// SetHide sets the WZ life `hide` flag; a hidden point is never auto-spawned (FR-1.4).
func (b *Builder) SetHide(hide bool) *Builder {
	b.hide = hide
	return b
}

// Build returns the constructed SpawnPoint.
func (b *Builder) Build() SpawnPoint {
	return SpawnPoint{
		Id:       b.id,
		Template: b.template,
		MobTime:  b.mobTime,
		Team:     b.team,
		Cy:       b.cy,
		F:        b.f,
		Fh:       b.fh,
		Rx0:      b.rx0,
		Rx1:      b.rx1,
		X:        b.x,
		Y:        b.y,
		Hide:     b.hide,
	}
}
