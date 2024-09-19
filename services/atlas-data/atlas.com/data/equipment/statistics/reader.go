package statistics

import (
	"atlas-data/wz"
	"atlas-data/xml"
	"errors"
	"fmt"
	"github.com/Chronicle20/atlas-tenant"
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

	return &Model{
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
	}, nil
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
