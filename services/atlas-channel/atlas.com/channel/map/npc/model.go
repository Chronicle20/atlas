// Package npc is atlas-channel's read client for atlas-maps' map/npc REST
// surface: the scripted NPCs currently placed on a field by the spawn_npc
// saga action (task-290 C14), consumed to replay existing state to a
// character entering the field (task-BC2). It mirrors the shape of the
// neighbouring kite/ client (model.go, builder.go, rest.go, requests.go,
// processor.go) and follows the repo's immutable-model convention --
// unexported fields, accessors, a Builder. No *_testhelpers.go: test setup
// goes through the Builder.
package npc

// Model is one scripted NPC placed on a field, as atlas-channel sees it --
// mirroring atlas-maps' map/npc.Model's REST-visible fields (uniqueId,
// npcId, x, y, fh).
type Model struct {
	uniqueId uint32
	npcId    uint32
	x        int16
	y        int16
	fh       int16
}

func (m Model) UniqueId() uint32 { return m.uniqueId }
func (m Model) NpcId() uint32    { return m.npcId }
func (m Model) X() int16         { return m.x }
func (m Model) Y() int16         { return m.y }
func (m Model) Fh() int16        { return m.fh }
