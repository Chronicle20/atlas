package model

import (
	"atlas-channel/asset"
	"context"

	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	model "github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

func NewAsset(zeroPosition bool, a asset.Model) packetmodel.Asset {
	base := packetmodel.NewAsset(zeroPosition, a.Slot(), a.TemplateId(), a.Expiration())

	if a.IsEquipment() {
		base = base.SetEquipmentStats(
			a.Strength(), a.Dexterity(), a.Intelligence(), a.Luck(),
			a.Hp(), a.Mp(),
			a.WeaponAttack(), a.MagicAttack(), a.WeaponDefense(), a.MagicDefense(),
			a.Accuracy(), a.Avoidability(), a.Hands(), a.Speed(), a.Jump(),
		)
		base = base.SetEquipmentMeta(a.Slots(), a.LevelType(), a.Level(), a.Experience(), a.HammersApplied(), a.Flag())
	}

	if a.IsCashEquipment() || a.IsCash() {
		base = base.SetCashId(a.CashId())
	}

	if a.IsStackable() || (a.IsCash() && !a.IsPet()) {
		base = base.SetStackableInfo(a.Quantity(), a.Flag(), a.Rechargeable())
	}

	if a.IsPet() {
		base = base.SetPetInfo(a.PetId(), a.PetName(), a.PetLevel(), a.Fullness(), a.Closeness())
	}

	return base
}

func NewAssetWriter(l logrus.FieldLogger, ctx context.Context, options map[string]interface{}, w *response.Writer) func(zeroPosition bool) model.Operator[asset.Model] {
	return func(zeroPosition bool) model.Operator[asset.Model] {
		return func(a asset.Model) error {
			am := NewAsset(zeroPosition, a)
			w.WriteByteArray(am.Encode(l, ctx)(options))
			return nil
		}
	}
}
