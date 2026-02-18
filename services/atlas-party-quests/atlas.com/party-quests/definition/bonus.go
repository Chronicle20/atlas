package definition

type BonusEntry string

const (
	BonusEntryAuto   BonusEntry = "auto"
	BonusEntryManual BonusEntry = "manual"
)

type Bonus struct {
	mapId           uint32
	duration        uint64
	entry           BonusEntry
	completionMapId uint32
	properties      map[string]any
}

func (b Bonus) MapId() uint32              { return b.mapId }
func (b Bonus) Duration() uint64           { return b.duration }
func (b Bonus) Entry() BonusEntry          { return b.entry }
func (b Bonus) CompletionMapId() uint32    { return b.completionMapId }
func (b Bonus) Properties() map[string]any { return b.properties }
