package objectid

// PlayerNpcObjectIdBase reserves the client oid band [100000, 999999] for
// Player NPCs (imitated NPCs): oids derived deterministically from the
// Player NPC's script id via PlayerNpcObjectIdFor, rather than allocated by
// Allocate. Deterministic derivation means a Player NPC's oid needs no
// counter and survives a Redis flush.
//
// This band sits above the static WZ NPC oid range (assigned per-map
// starting at 1, typically under 100 per map) and strictly below MinId, the
// first id the tenant-scoped Allocator ever hands out. Lowering MinId below
// 1,000,000 would collide with this reservation -- see design D-5.
const PlayerNpcObjectIdBase = uint32(100000)

// playerNpcScriptIdBase is the lowest Player NPC script id (design §4.2's
// pool spans 9901000-9906599); PlayerNpcObjectIdFor maps script ids
// downward from PlayerNpcObjectIdBase starting here.
const playerNpcScriptIdBase = uint32(9900000)

// PlayerNpcObjectIdFor derives the client-visible oid for a Player NPC from
// its script id: PlayerNpcObjectIdBase + (scriptId - 9900000). The mapping
// is deterministic, so the same script id always yields the same oid.
//
// scriptId values below 9900000 are guarded and return PlayerNpcObjectIdBase
// unchanged -- the allocator never produces such an id (design §4.2's pool
// is 9901000-9906599), and the caller must not be handed a value that
// wrapped around uint32's zero.
func PlayerNpcObjectIdFor(scriptId uint32) uint32 {
	if scriptId < playerNpcScriptIdBase {
		return PlayerNpcObjectIdBase
	}
	return PlayerNpcObjectIdBase + (scriptId - playerNpcScriptIdBase)
}

// playerNpcImitateTemplateMin and playerNpcImitateTemplateMax bound the
// Player NPC imitate pool (design §4.2: 9901000-9906599). WZ Hall of Fame
// maps carry one placeholder NPC life entry per pool slot, one per possible
// deployed Player NPC; the client's CNpcPool::OnNpcImitateData overlay is
// keyed on template id, not oid, so a deployed Player NPC (spawned with its
// own SPAWN_NPC, outside this pool) and its unoccupied placeholder siblings
// would otherwise render as duplicates (task-251 bug report §2).
const (
	playerNpcImitateTemplateMin = uint32(9901000)
	playerNpcImitateTemplateMax = uint32(9906599)
)

// IsPlayerNpcImitateTemplate reports whether templateId falls within the
// Player NPC imitate pool (design §4.2: 9901000-9906599). Callers building a
// per-map NPC list use this to drop the WZ placeholder entries so they never
// spawn or receive controller elections alongside a deployed Player NPC.
func IsPlayerNpcImitateTemplate(templateId uint32) bool {
	return templateId >= playerNpcImitateTemplateMin && templateId <= playerNpcImitateTemplateMax
}
