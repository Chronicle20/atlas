package factory

import (
	"atlas-character-factory/character"
	"atlas-character-factory/configuration"
	"atlas-character-factory/tenant"
	"context"
	"errors"
	"github.com/Chronicle20/atlas-model/async"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"time"
)

func Create(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(accountId uint32, worldId byte, name string, gender byte, jobIndex uint32, subJobIndex uint32, face uint32, hair uint32, hairColor uint32, skinColor byte, top uint32, bottom uint32, shoes uint32, weapon uint32) (character.Model, error) {
	return func(accountId uint32, worldId byte, name string, gender byte, jobIndex uint32, subJobIndex uint32, face uint32, hair uint32, hairColor uint32, skinColor byte, top uint32, bottom uint32, shoes uint32, weapon uint32) (character.Model, error) {
		// TODO validate name again.

		if !validGender(gender) {
			return character.Model{}, errors.New("gender must be 0 or 1")
		}

		if !validJob(jobIndex, subJobIndex) {
			return character.Model{}, errors.New("must provide valid job index")
		}

		c, err := configuration.GetConfiguration()
		if err != nil {
			l.WithError(err).Errorf("Unable to find template validation configuration")
			return character.Model{}, err
		}
		tc, err := c.FindTemplate(tenant.Id.String(), jobIndex, subJobIndex, gender)
		if err != nil {
			l.WithError(err).Errorf("Unable to find template validation configuration")
			return character.Model{}, err
		}

		if !validFace(tc.Face, face) {
			l.Errorf("Chosen face [%d] is not valid for job [%d].", face, jobIndex)
			return character.Model{}, errors.New("chosen face is not valid for job")
		}

		if !validHair(tc.Hair, hair) {
			l.Errorf("Chosen hair [%d] is not valid for job [%d].", hair, jobIndex)
			return character.Model{}, errors.New("chosen hair is not valid for job")
		}

		if !validHairColor(tc.HairColor, hairColor) {
			l.Errorf("Chosen hair color [%d] is not valid for job [%d].", hairColor, jobIndex)
			return character.Model{}, errors.New("chosen hair color is not valid for job")
		}

		if !validSkinColor(tc.SkinColor, uint32(skinColor)) {
			l.Errorf("Chosen skin color [%d] is not valid for job [%d]", skinColor, jobIndex)
			return character.Model{}, errors.New("chosen skin color is not valid for job")
		}

		if !validTop(tc.Top, top) {
			l.Errorf("Chosen top [%d] is not valid for job [%d]", top, jobIndex)
			return character.Model{}, errors.New("chosen top is not valid for job")
		}

		if !validBottom(tc.Bottom, bottom) {
			l.Errorf("Chosen bottom [%d] is not valid for job [%d]", bottom, jobIndex)
			return character.Model{}, errors.New("chosen bottom is not valid for job")
		}

		if !validShoes(tc.Shoes, shoes) {
			l.Errorf("Chosen shoes [%d] is not valid for job [%d]", shoes, jobIndex)
			return character.Model{}, errors.New("chosen shoes is not valid for job")
		}

		if !validWeapon(tc.Weapon, weapon) {
			l.Errorf("Chosen weapon [%d] is not valid for job [%d]", weapon, jobIndex)
			return character.Model{}, errors.New("chosen weapon is not valid for job")
		}

		asyncCreate := func(ctx context.Context, rchan chan uint32, echan chan error) {
			character.AwaitCreated(l, tenant)(name)(ctx, rchan, echan)
			_, err = character.Create(l, span, tenant)(accountId, worldId, name, gender, tc.MapId, jobIndex, subJobIndex, face, hair, hairColor, skinColor)
			if err != nil {
				l.WithError(err).Errorf("Unable to create character from seed.")
				echan <- err
			}
		}

		l.Debugf("Beginning character creation for account [%d] in world [%d].", accountId, worldId)
		cid, err := async.Await[uint32](model.FixedProvider[async.Provider[uint32]](asyncCreate), async.SetTimeout(500*time.Millisecond))()
		if err != nil {
			l.WithError(err).Errorf("Unable to create character [%s].", name)
			return character.Model{}, err
		}

		ip := model.FixedProvider([]uint32{top, bottom, shoes, weapon})
		l.Debugf("Beginning item creation for character [%d].", cid)
		items, err := async.AwaitSlice[character.ItemGained](model.SliceMap(ip, asyncItemCreate(l, span, tenant)(cid)), async.SetTimeout(1*time.Second))()
		if err != nil {
			l.WithError(err).Errorf("Error creating an item for character [%d].", cid)
		}

		_, err = async.AwaitSlice[uint32](model.SliceMap(model.FixedProvider(items), asyncEquipItem(l, span, tenant)(cid)), async.SetTimeout(1*time.Second))()
		if err != nil {
			l.WithError(err).Errorf("Error equipping an item for character [%d].", cid)
		}
		return character.GetById(l, span, tenant)(cid)
	}
}

func asyncItemCreate(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(characterId uint32) func(itemId uint32) (async.Provider[character.ItemGained], error) {
	return func(characterId uint32) func(itemId uint32) (async.Provider[character.ItemGained], error) {
		return func(itemId uint32) (async.Provider[character.ItemGained], error) {
			return func(ctx context.Context, rchan chan character.ItemGained, echan chan error) {
				character.AwaitItemGained(l, tenant)(itemId)(ctx, rchan, echan)
				_, err := character.CreateItem(l, span, tenant)(characterId, itemId)
				if err != nil {
					l.WithError(err).Errorf("Unable to create item [%d] from seed for character [%d].", itemId, characterId)
					echan <- err
				}
			}, nil
		}
	}
}

func asyncEquipItem(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(characterId uint32) func(character.ItemGained) (async.Provider[uint32], error) {
	return func(characterId uint32) func(character.ItemGained) (async.Provider[uint32], error) {
		return func(ig character.ItemGained) (async.Provider[uint32], error) {
			return func(ctx context.Context, rchan chan uint32, echan chan error) {
				character.AwaitEquipChanged(l, tenant)(ig.ItemId)(ctx, rchan, echan)
				err := character.EquipItem(l, span, tenant)(characterId, ig.ItemId, ig.Slot)
				if err != nil {
					l.WithError(err).Errorf("Unable to equip item [%d] for character [%d].", ig.ItemId, characterId)
					echan <- err
				}
			}, nil
		}
	}
}

func validWeapon(weapons []uint32, weapon uint32) bool {
	return validOption(weapons, weapon)
}

func validShoes(shoes []uint32, shoe uint32) bool {
	return validOption(shoes, shoe)
}

func validBottom(bottoms []uint32, bottom uint32) bool {
	return validOption(bottoms, bottom)
}

func validTop(tops []uint32, top uint32) bool {
	return validOption(tops, top)
}

func validSkinColor(colors []uint32, color uint32) bool {
	return validOption(colors, color)
}

func validHairColor(colors []uint32, color uint32) bool {
	return validOption(colors, color)
}

func validHair(hairs []uint32, hair uint32) bool {
	return validOption(hairs, hair)
}

func validOption(options []uint32, selection uint32) bool {
	if selection == 0 {
		return true
	}

	for _, option := range options {
		if option == selection {
			return true
		}
	}
	return false
}

func validFace(faces []uint32, face uint32) bool {
	return validOption(faces, face)
}

func validJob(jobIndex uint32, subJobIndex uint32) bool {
	return true
}

func validGender(gender byte) bool {
	return gender == 0 || gender == 1
}
