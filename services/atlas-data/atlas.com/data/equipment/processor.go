package equipment

import (
	"atlas-data/xml"
	"context"
	"fmt"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"path/filepath"
	"strconv"
	"strings"
)

func byIdProvider(ctx context.Context) func(mapId uint32) model.Provider[Model] {
	t := tenant.MustFromContext(ctx)
	return func(mapId uint32) model.Provider[Model] {
		return func() (Model, error) {
			m, err := GetEquipmentModelRegistry().Get(t, mapId)
			if err == nil {
				return m, nil
			}
			nt, err := tenant.Create(uuid.Nil, t.Region(), t.MajorVersion(), t.MinorVersion())
			if err != nil {
				return Model{}, err
			}
			return GetEquipmentModelRegistry().Get(nt, mapId)
		}
	}
}

func GetById(ctx context.Context) func(mapId uint32) (Model, error) {
	return func(mapId uint32) (Model, error) {
		return byIdProvider(ctx)(mapId)()
	}
}

func parseItemId(filePath string) (uint32, error) {
	baseName := filepath.Base(filePath)
	if !strings.HasSuffix(baseName, ".img.xml") {
		return 0, fmt.Errorf("file does not match expected format: %s", filePath)
	}
	idStr := strings.TrimSuffix(baseName, ".img.xml")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return 0, err
	}
	return uint32(id), nil

}

func RegisterEquipment(l logrus.FieldLogger) func(ctx context.Context) func(path string) {
	return func(ctx context.Context) func(path string) {
		t := tenant.MustFromContext(ctx)
		return func(path string) {
			m, err := ReadFromFile(l)(ctx)(path)()
			if err == nil {
				l.Debugf("Processed equipment [%d].", m.Id())
				_ = GetEquipmentModelRegistry().Add(t, m)
			}
		}
	}
}

func ReadFromFile(l logrus.FieldLogger) func(ctx context.Context) func(path string) model.Provider[Model] {
	return func(ctx context.Context) func(path string) model.Provider[Model] {
		return func(path string) model.Provider[Model] {
			itemId, err := parseItemId(path)
			if err != nil {
				return model.ErrorProvider[Model](err)
			}

			exml, err := xml.Read(path)
			if err != nil {
				return model.ErrorProvider[Model](err)
			}

			info, err := exml.ChildByName("info")
			if err != nil {
				info, err := exml.ChildByName("0" + strconv.Itoa(int(itemId)))
				if err != nil {
					return model.ErrorProvider[Model](err)
				} else {
					info, err = info.ChildByName("info")
					if err != nil {
						return model.ErrorProvider[Model](err)
					}
				}
			}
			if info == nil {
				return model.FixedProvider(Model{itemId: itemId})
			}

			slotStr := info.GetString("islot", "")

			m := Model{
				itemId:        itemId,
				strength:      info.GetShort("incSTR", 0),
				dexterity:     info.GetShort("incDEX", 0),
				intelligence:  info.GetShort("incINT", 0),
				luck:          info.GetShort("incLUK", 0),
				weaponAttack:  info.GetShort("incPAD", 0),
				weaponDefense: info.GetShort("incPDD", 0),
				magicAttack:   info.GetShort("incMAD", 0),
				magicDefense:  info.GetShort("incMDD", 0),
				accuracy:      info.GetShort("incACC", 0),
				avoidability:  info.GetShort("incEVA", 0),
				speed:         info.GetShort("incSpeed", 0),
				jump:          info.GetShort("incJump", 0),
				hp:            info.GetShort("incMHP", 0),
				mp:            info.GetShort("incMMP", 0),
				slots:         info.GetShort("tuc", 0),
				cash:          info.GetBool("cash", false),
				slotName:      getNameFromWz(slotStr),
				slotWz:        slotStr,
				slotIndex:     getSlotsFromWz(slotStr),
			}
			return model.FixedProvider(m)
		}
	}
}

func getSlotsFromWz(wz string) []int16 {
	switch wz {
	case "Cp":
		return []int16{-1}
	case "HrCp":
		return []int16{-1}
	case "Af":
		return []int16{-2}
	case "Ay":
		return []int16{-3}
	case "Ae":
		return []int16{-4}
	case "Ma":
		return []int16{-5}
	case "MaPn":
		return []int16{-5}
	case "Pn":
		return []int16{-6}
	case "So":
		return []int16{-7}
	case "GlGw":
		return []int16{-8}
	case "Gv":
		return []int16{-8}
	case "Sr":
		return []int16{-9}
	case "Si":
		return []int16{-10}
	case "Wp":
		return []int16{-11}
	case "WpSi":
		return []int16{-11}
	case "WpSp":
		return []int16{-11}
	case "Ri":
		return []int16{-12, -13, -15, -16}
	case "Pe":
		return []int16{-17}
	case "Tm":
		return []int16{-18}
	case "Sd":
		return []int16{-19}
	case "Me":
		return []int16{-49}
	case "Be":
		return []int16{-50}
	default:
		return []int16{0}
	}
}

func getNameFromWz(wz string) string {
	switch wz {
	case "Cp":
		return "HAT"
	case "HrCp":
		return "SPECIAL_HAT"
	case "Af":
		return "FACE_ACCESSORY"
	case "Ay":
		return "EYE_ACCESSORY"
	case "Ae":
		return "EARRINGS"
	case "Ma":
		return "TOP"
	case "MaPn":
		return "OVERALL"
	case "Pn":
		return "PANTS"
	case "So":
		return "SHOES"
	case "GlGw":
		return "GLOVES"
	case "Gv":
		return "CASH_GLOVES"
	case "Sr":
		return "CAPE"
	case "Si":
		return "SHIELD"
	case "Wp":
		return "WEAPON"
	case "WpSi":
		return "WEAPON_2"
	case "WpSp":
		return "LOW_WEAPON"
	case "Ri":
		return "RING"
	case "Pe":
		return "PENDANT"
	case "Tm":
		return "TAMED_MOB"
	case "Sd":
		return "SADDLE"
	case "Me":
		return "MEDAL"
	case "Be":
		return "BELT"
	default:
		return "PET_EQUIP"
	}
}
