package slots

import (
	"atlas-data/tenant"
	"atlas-data/wz"
	"atlas-data/xml"
	"errors"
	"fmt"
	"strconv"
)

func Read(tenant tenant.Model, itemId uint32) (*Model, error) {
	i, err := findItem(tenant, itemId)
	if err != nil {
		return nil, err
	}

	exml, err := xml.Read(i.Path())
	if err != nil {
		return nil, err
	}
	return getEquipmentFromInfo(itemId, exml)
}

func getEquipmentFromInfo(itemId uint32, exml *xml.Node) (*Model, error) {
	info, err := exml.ChildByName("info")
	if err != nil {
		info, err := exml.ChildByName("0" + strconv.Itoa(int(itemId)))
		if err != nil {
			return nil, err
		} else {
			info, err = info.ChildByName("info")
			if err != nil {
				return nil, err
			}
		}
	}
	if info == nil {
		return &Model{itemId: itemId}, nil
	}

	slotStr := info.GetString("islot", "")

	return &Model{
		itemId: itemId,
		name:   getNameFromWz(slotStr),
		wz:     slotStr,
		slot:   getSlotsFromWz(slotStr),
	}, nil
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

func findItem(tenant tenant.Model, itemId uint32) (*wz.FileEntry, error) {
	idstr := "0" + strconv.Itoa(int(itemId))
	runes := []rune(idstr)

	if val, ok := wz.GetFileCache().GetFile(tenant, "Character.wz", string(runes[0:4])+".img.xml"); ok == nil {
		return val, nil
	}
	if val, ok := wz.GetFileCache().GetFile(tenant, "Character.wz", string(runes[0:1])+".img.xml"); ok == nil {
		return val, nil
	}
	if val, ok := wz.GetFileCache().GetFile(tenant, "Character.wz", idstr+".img.xml"); ok == nil {
		return val, nil
	}
	return nil, errors.New(fmt.Sprintf("item %d not found", itemId))
}
