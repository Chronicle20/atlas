package inventory

import (
	"strconv"
	"time"

	"github.com/jtumidanski/api2go/jsonapi"
)

// CompartmentRestModel represents a compartment from atlas-inventory service
type CompartmentRestModel struct {
	Id            string           `json:"-"`
	InventoryType int8             `json:"type"`
	Capacity      uint32           `json:"capacity"`
	Assets        []AssetRestModel `json:"-"`
}

func (r CompartmentRestModel) GetName() string {
	return "compartments"
}

func (r CompartmentRestModel) GetID() string {
	return r.Id
}

func (r *CompartmentRestModel) SetID(strId string) error {
	r.Id = strId
	return nil
}

func (r *CompartmentRestModel) SetToManyReferenceIDs(name string, IDs []string) error {
	if name == "assets" {
		for _, idStr := range IDs {
			id, err := strconv.Atoi(idStr)
			if err != nil {
				return err
			}
			r.Assets = append(r.Assets, AssetRestModel{Id: uint32(id)})
		}
	}
	return nil
}

// GetReferences declares the to-many relationship list so api2go can wire
// `included` resources back to this compartment.
func (r CompartmentRestModel) GetReferences() []jsonapi.Reference {
	return []jsonapi.Reference{
		{
			Type: "assets",
			Name: "assets",
		},
	}
}

// GetReferencedIDs lists the IDs api2go should expect in `included` for the
// declared references. Required by the api2go fork interface even though this
// service only consumes (not produces) compartment documents.
func (r CompartmentRestModel) GetReferencedIDs() []jsonapi.ReferenceID {
	var result []jsonapi.ReferenceID
	for _, v := range r.Assets {
		result = append(result, jsonapi.ReferenceID{
			ID:   v.GetID(),
			Type: v.GetName(),
			Name: v.GetName(),
		})
	}
	return result
}

// GetReferencedStructs returns the embedded asset structs for outbound
// marshalling. Implemented to mirror the in-repo template
// (atlas-npc-shops/.../compartment/rest.go) and avoid surprises if this type
// ever needs to round-trip.
func (r CompartmentRestModel) GetReferencedStructs() []jsonapi.MarshalIdentifier {
	var result []jsonapi.MarshalIdentifier
	for key := range r.Assets {
		result = append(result, r.Assets[key])
	}
	return result
}

// SetToOneReferenceID is a no-op satisfier — CompartmentRestModel has no
// to-one relationships, but the api2go fork interface requires the method.
func (r *CompartmentRestModel) SetToOneReferenceID(_, _ string) error {
	return nil
}

// SetReferencedStructs walks the stub assets that SetToManyReferenceIDs
// populated and replaces each with a fully-hydrated copy from `included`,
// using jsonapi.ProcessIncludeData to populate flat attribute fields
// (slot, hp, mp, strength, …) from the included resource's attributes object.
//
// Without this method, every AssetRestModel in r.Assets retains its zero-value
// Slot/Hp/Mp, causing the downstream IsEquipped() (Slot < 0) gate in
// character/initializer.go::fetchEquipmentBonuses to skip every equipped
// asset and silently drop all equipment bonuses.
func (r *CompartmentRestModel) SetReferencedStructs(references map[string]map[string]jsonapi.Data) error {
	if refMap, ok := references["assets"]; ok {
		assets := make([]AssetRestModel, 0)
		for _, ri := range r.Assets {
			if ref, ok := refMap[ri.GetID()]; ok {
				wip := ri
				err := jsonapi.ProcessIncludeData(&wip, ref, references)
				if err != nil {
					return err
				}
				assets = append(assets, wip)
			}
		}
		r.Assets = assets
	}
	return nil
}

// AssetRestModel represents an asset from atlas-inventory service
type AssetRestModel struct {
	Id            uint32    `json:"-"`
	Slot          int16     `json:"slot"`
	TemplateId    uint32    `json:"templateId"`
	Expiration    time.Time `json:"expiration"`
	OwnerId       uint32    `json:"ownerId"`
	Strength      uint16    `json:"strength"`
	Dexterity     uint16    `json:"dexterity"`
	Intelligence  uint16    `json:"intelligence"`
	Luck          uint16    `json:"luck"`
	Hp            uint16    `json:"hp"`
	Mp            uint16    `json:"mp"`
	WeaponAttack  uint16    `json:"weaponAttack"`
	MagicAttack   uint16    `json:"magicAttack"`
	WeaponDefense uint16    `json:"weaponDefense"`
	MagicDefense  uint16    `json:"magicDefense"`
	Accuracy      uint16    `json:"accuracy"`
	Avoidability  uint16    `json:"avoidability"`
	Hands         uint16    `json:"hands"`
	Speed         uint16    `json:"speed"`
	Jump          uint16    `json:"jump"`
}

func (r AssetRestModel) GetName() string {
	return "assets"
}

func (r AssetRestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *AssetRestModel) SetID(strId string) error {
	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

// EquipableRestData contains equipment stats
type EquipableRestData struct {
	OwnerId       uint32 `json:"ownerId"`
	Strength      uint16 `json:"strength"`
	Dexterity     uint16 `json:"dexterity"`
	Intelligence  uint16 `json:"intelligence"`
	Luck          uint16 `json:"luck"`
	Hp            uint16 `json:"hp"`
	Mp            uint16 `json:"mp"`
	WeaponAttack  uint16 `json:"weaponAttack"`
	MagicAttack   uint16 `json:"magicAttack"`
	WeaponDefense uint16 `json:"weaponDefense"`
	MagicDefense  uint16 `json:"magicDefense"`
	Accuracy      uint16 `json:"accuracy"`
	Avoidability  uint16 `json:"avoidability"`
	Hands         uint16 `json:"hands"`
	Speed         uint16 `json:"speed"`
	Jump          uint16 `json:"jump"`
}

// IsEquipped returns true if this asset is in an equipped slot (negative slot number)
func (r AssetRestModel) IsEquipped() bool {
	return r.Slot < 0
}

// GetEquipableData returns the equipment stats from this asset's flat fields
func (r AssetRestModel) GetEquipableData() (EquipableRestData, bool) {
	if !r.IsEquipped() {
		return EquipableRestData{}, false
	}
	return EquipableRestData{
		OwnerId:       r.OwnerId,
		Strength:      r.Strength,
		Dexterity:     r.Dexterity,
		Intelligence:  r.Intelligence,
		Luck:          r.Luck,
		Hp:            r.Hp,
		Mp:            r.Mp,
		WeaponAttack:  r.WeaponAttack,
		MagicAttack:   r.MagicAttack,
		WeaponDefense: r.WeaponDefense,
		MagicDefense:  r.MagicDefense,
		Accuracy:      r.Accuracy,
		Avoidability:  r.Avoidability,
		Hands:         r.Hands,
		Speed:         r.Speed,
		Jump:          r.Jump,
	}, true
}
