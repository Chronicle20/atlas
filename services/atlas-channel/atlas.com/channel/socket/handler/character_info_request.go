package handler

import (
	"atlas-channel/cashshop/wishlist"
	"atlas-channel/character"
	"atlas-channel/guild"
	"atlas-channel/mount"
	"atlas-channel/pet"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	charcb "github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound"
	charsb "github.com/Chronicle20/atlas/libs/atlas-packet/character/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/sirupsen/logrus"
	petpkt "github.com/Chronicle20/atlas/libs/atlas-packet/pet/clientbound"
)

func CharacterInfoRequestHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := charsb.InfoRequest{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		cp := character.NewProcessor(l, ctx)
		decorators := make([]model.Decorator[character.Model], 0)
		if p.PetInfo() {
			decorators = append(decorators, cp.PetAssetEnrichmentDecorator)
		}
		decorators = append(decorators, cp.MonsterBookDecorator)
		c, err := cp.GetById(decorators...)(p.CharacterId())
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve character [%d] being requested.", p.CharacterId())
			return
		}
		g, _ := guild.NewProcessor(l, ctx).GetByMemberId(p.CharacterId())

		var wl []wishlist.Model
		wl, err = wishlist.NewProcessor(l, ctx).GetByCharacterId(p.CharacterId())
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve wishlist for character [%d].", p.CharacterId())
			wl = make([]wishlist.Model, 0)
		}

		if p.CharacterId() != s.CharacterId() {
			var ps []pet.Model
			ps, err = pet.NewProcessor(l, ctx).GetByOwner(p.CharacterId())
			if err != nil {
				l.WithError(err).Errorf("Unable to retrieve pet [%d] being requested.", p.CharacterId())
			}

			for _, pe := range ps {
				excludeIds := make([]uint32, len(pe.Excludes()))
				for i, e := range pe.Excludes() {
					excludeIds[i] = e.ItemId()
				}
				_ = session.Announce(l)(ctx)(wp)(petpkt.PetExcludeResponseWriter)(petpkt.NewPetExcludeResponse(pe.OwnerId(), pe.Slot(), uint64(pe.Id()), excludeIds).Encode)(s)
			}
		}

		// Tamed-mob (mount) block: shown only for characters with a tamed-mob
		// equipped (slot tamingMob), mirroring the v83/v87/v95 client's own gate.
		// Gating here also avoids atlas-mounts' default-on-read creating a mount row
		// for every character whose info is viewed. When no mount: inactive (single 0
		// byte). A non-nil fetch error is a real transport/5xx failure (atlas-mounts
		// default-creates, so 404 is unreachable) and is logged, not silently dropped.
		mountInfo := charcb.MountInfo{}
		if tms, sErr := slot.GetSlotByType("tamingMob"); sErr == nil {
			// Equipment().Get returns ok for every defined slot (the map is
			// pre-populated), so test the actual equipped item, not just ok.
			if em, ok := c.Equipment().Get(tms.Type); ok && em.Equipable != nil {
				if mm, mErr := mount.NewProcessor(l, ctx).GetByCharacterId(p.CharacterId()); mErr != nil {
					l.WithError(mErr).Warnf("Unable to retrieve mount for character [%d]; omitting mount block from character info.", p.CharacterId())
				} else {
					mountInfo = charcb.MountInfo{
						Active:    true,
						Level:     uint32(mm.Level()),
						Exp:       uint32(mm.Exp()),
						Tiredness: uint32(mm.Tiredness()),
					}
				}
			}
		}

		err = session.Announce(l)(ctx)(wp)(charcb.CharacterInfoWriter)(writer.CharacterInfoBody(c, g, wl, mountInfo))(s)
		if err != nil {
			l.WithError(err).Errorf("Unable to write character information.")
		}
	}
}
