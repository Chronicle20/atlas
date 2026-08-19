package character

import (
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jtumidanski/api2go/jsonapi"

	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

type RestModel struct {
	Id                 uint32   `json:"-"`
	AccountId          uint32   `json:"accountId"`
	WorldId            world.Id `json:"worldId"`
	Name               string   `json:"name"`
	Level              byte     `json:"level"`
	Experience         uint32   `json:"experience"`
	GachaponExperience uint32   `json:"gachaponExperience"`
	Strength           uint16   `json:"strength"`
	Dexterity          uint16   `json:"dexterity"`
	Intelligence       uint16   `json:"intelligence"`
	Luck               uint16   `json:"luck"`
	Hp                 uint16   `json:"hp"`
	MaxHp              uint16   `json:"maxHp"`
	Mp                 uint16   `json:"mp"`
	MaxMp              uint16   `json:"maxMp"`
	Meso               uint32   `json:"meso"`
	HpMpUsed           int      `json:"hpMpUsed"`
	JobId              job.Id   `json:"jobId"`
	SkinColor          byte     `json:"skinColor"`
	Gender             byte     `json:"gender"`
	Fame               int16    `json:"fame"`
	Hair               uint32   `json:"hair"`
	Face               uint32   `json:"face"`
	Ap                 uint16   `json:"ap"`
	Sp                 string   `json:"sp"`
	SpawnPoint         uint32   `json:"spawnPoint"`
	Gm                 int      `json:"gm"`
	X                  int16    `json:"x"`
	Y                  int16    `json:"y"`
	Stance             byte     `json:"stance"`
}

func (r RestModel) GetName() string {
	return "characters"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *RestModel) SetID(strId string) error {
	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

func (r RestModel) GetReferences() []jsonapi.Reference {
	return []jsonapi.Reference{}
}

func (r RestModel) GetReferencedIDs() []jsonapi.ReferenceID {
	var result []jsonapi.ReferenceID
	return result
}

func (r RestModel) GetReferencedStructs() []jsonapi.MarshalIdentifier {
	var result []jsonapi.MarshalIdentifier
	return result
}

func (r *RestModel) SetToOneReferenceID(_, _ string) error {
	return nil
}

func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}

func (r *RestModel) SetReferencedStructs(_ map[string]map[string]jsonapi.Data) error {
	return nil
}

func Extract(m RestModel) (Model, error) {
	return Model{
		id:                 m.Id,
		accountId:          m.AccountId,
		worldId:            m.WorldId,
		name:               m.Name,
		level:              m.Level,
		experience:         m.Experience,
		gachaponExperience: m.GachaponExperience,
		strength:           m.Strength,
		dexterity:          m.Dexterity,
		intelligence:       m.Intelligence,
		luck:               m.Luck,
		hp:                 m.Hp,
		mp:                 m.Mp,
		maxHp:              m.MaxHp,
		maxMp:              m.MaxMp,
		meso:               m.Meso,
		hpMpUsed:           m.HpMpUsed,
		jobId:              m.JobId,
		skinColor:          m.SkinColor,
		gender:             m.Gender,
		fame:               m.Fame,
		hair:               m.Hair,
		face:               m.Face,
		ap:                 m.Ap,
		sp:                 m.Sp,
		gm:                 m.Gm,
		x:                  m.X,
		y:                  m.Y,
		stance:             m.Stance,
	}, nil
}

// EquipSlotExtensionRestModel mirrors atlas-character's equipslot.RestModel
// (services/atlas-character/atlas.com/character/equipslot/rest.go): one
// character's equip-slot extension, read on GET and returned by the POST
// write this task adds. SlotIndex is the Atlas canonical equipped-inventory
// position (derivation-equip-slot.md E1 / R1) -- e.g. the pendant2 constant
// (libs/atlas-constants/inventory/slot) -- never a wire value.
type EquipSlotExtensionRestModel struct {
	Id          string    `json:"-"`
	CharacterId uint32    `json:"characterId"`
	SlotIndex   int16     `json:"slotIndex"`
	ExpiresAt   time.Time `json:"expiresAt"`
}

func (r EquipSlotExtensionRestModel) GetName() string {
	return "equip-slot-extensions"
}

func (r EquipSlotExtensionRestModel) GetID() string {
	return r.Id
}

func (r *EquipSlotExtensionRestModel) SetID(id string) error {
	r.Id = id
	return nil
}

func (r *EquipSlotExtensionRestModel) SetToOneReferenceID(_, _ string) error {
	return nil
}

func (r *EquipSlotExtensionRestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}

// ExtendEquipSlotInputRestModel is the POST body atlas-character's write
// route expects. SlotIndex carries the Atlas canonical position (R1) --
// the caller resolves it (e.g. via slot.GetSlotByType("pendant2")), atlas-
// character never invents it. Days is the extension's length; atlas-
// character converts it to a time.Duration. TransactionId is the purchase's
// own idempotency key (task-240 task 24c): the atlas-character write route
// dedupes on it, so a redelivered EXTEND_EQUIP_SLOT outbox command (the
// outbox is at-least-once) does not double-extend.
type ExtendEquipSlotInputRestModel struct {
	Id            string    `json:"-"`
	SlotIndex     int16     `json:"slotIndex"`
	Days          uint16    `json:"days"`
	TransactionId uuid.UUID `json:"transactionId"`
}

func (r ExtendEquipSlotInputRestModel) GetName() string {
	return "equip-slot-extensions"
}

func (r ExtendEquipSlotInputRestModel) GetID() string {
	return r.Id
}

func (r *ExtendEquipSlotInputRestModel) SetID(id string) error {
	r.Id = id
	return nil
}

func (r *ExtendEquipSlotInputRestModel) SetToOneReferenceID(_, _ string) error {
	return nil
}

func (r *ExtendEquipSlotInputRestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}

// ExtractEquipSlotExtension is the write route's response transformer --
// callers only need the resulting expiry, mirroring pet.Extract's shape
// (pet/rest.go) for a create-style POST.
func ExtractEquipSlotExtension(r EquipSlotExtensionRestModel) (time.Time, error) {
	return r.ExpiresAt, nil
}
