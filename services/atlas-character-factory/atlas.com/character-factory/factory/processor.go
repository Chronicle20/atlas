package factory

import (
	"atlas-character-factory/character"
	"atlas-character-factory/configuration"
	"atlas-character-factory/configuration/tenant/characters/template"
	character2 "atlas-character-factory/kafka/consumer/character"
	"atlas-character-factory/kafka/consumer/compartment"
	"atlas-character-factory/kafka/message/seed"
	"atlas-character-factory/kafka/producer"
	seed2 "atlas-character-factory/kafka/producer/seed"
	"context"
	"errors"
	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-model/async"
	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

func Create(l logrus.FieldLogger) func(ctx context.Context) func(input RestModel) error {
	return func(ctx context.Context) func(input RestModel) error {
		return func(input RestModel) error {
			// TODO validate name again.

			if !validGender(input.Gender) {
				return errors.New("gender must be 0 or 1")
			}

			if !validJob(input.JobIndex, input.SubJobIndex) {
				return errors.New("must provide valid job index")
			}

			t := tenant.MustFromContext(ctx)
			tc, err := configuration.GetTenantConfig(t.Id())
			if err != nil {
				l.WithError(err).Errorf("Unable to find template validation configuration")
				return err
			}

			var found = false
			var template template.RestModel
			for _, ref := range tc.Characters.Templates {
				if ref.JobIndex == input.JobIndex && ref.SubJobIndex == input.SubJobIndex && ref.Gender == input.Gender {
					found = true
					template = ref
				}
			}
			if !found {
				l.WithError(err).Errorf("Unable to find template validation configuration")
				return err
			}

			if !validFace(template.Faces, input.Face) {
				l.Errorf("Chosen face [%d] is not valid for job [%d].", input.Face, input.JobIndex)
				return errors.New("chosen face is not valid for job")
			}

			if !validHair(template.Hairs, input.Hair) {
				l.Errorf("Chosen hair [%d] is not valid for job [%d].", input.Hair, input.JobIndex)
				return errors.New("chosen hair is not valid for job")
			}

			if !validHairColor(template.HairColors, input.HairColor) {
				l.Errorf("Chosen hair color [%d] is not valid for job [%d].", input.HairColor, input.JobIndex)
				return errors.New("chosen hair color is not valid for job")
			}

			if !validSkinColor(template.SkinColors, uint32(input.SkinColor)) {
				l.Errorf("Chosen skin color [%d] is not valid for job [%d]", input.SkinColor, input.JobIndex)
				return errors.New("chosen skin color is not valid for job")
			}

			if !validTop(template.Tops, input.Top) {
				l.Errorf("Chosen top [%d] is not valid for job [%d]", input.Top, input.JobIndex)
				return errors.New("chosen top is not valid for job")
			}

			if !validBottom(template.Bottoms, input.Bottom) {
				l.Errorf("Chosen bottom [%d] is not valid for job [%d]", input.Bottom, input.JobIndex)
				return errors.New("chosen bottom is not valid for job")
			}

			if !validShoes(template.Shoes, input.Shoes) {
				l.Errorf("Chosen shoes [%d] is not valid for job [%d]", input.Shoes, input.JobIndex)
				return errors.New("chosen shoes is not valid for job")
			}

			if !validWeapon(template.Weapons, input.Weapon) {
				l.Errorf("Chosen weapon [%d] is not valid for job [%d]", input.Weapon, input.JobIndex)
				return errors.New("chosen weapon is not valid for job")
			}

			asyncCreate := func(actx context.Context, rchan chan uint32, echan chan error) {
				character2.AwaitCreated(l)(input.Name)(actx, rchan, echan)
				_, err = character.Create(l)(actx)(input.AccountId, input.WorldId, input.Name, input.Gender, template.MapId, input.JobIndex, input.SubJobIndex, input.Face, input.Hair, input.HairColor, input.SkinColor)
				if err != nil {
					l.WithError(err).Errorf("Unable to create character from seed.")
					echan <- err
				}
			}

			l.Debugf("Beginning character creation for account [%d] in world [%d].", input.AccountId, input.WorldId)
			cid, err := async.Await[uint32](model.FixedProvider[async.Provider[uint32]](asyncCreate), async.SetTimeout(500*time.Millisecond), async.SetContext(ctx))()
			if err != nil {
				l.WithError(err).Errorf("Unable to create character [%s].", input.Name)
				return err
			}
			l.Debugf("Character [%d] created.", cid)

			wg := sync.WaitGroup{}
			wg.Add(1)
			go func() {
				equipment := []uint32{input.Top, input.Bottom, input.Shoes, input.Weapon}
				assetMap := make(map[inventory.Type][]uint32)
				for _, aid := range template.Items {
					it, ok := inventory.TypeFromItemId(aid)
					if !ok {
						continue
					}
					var as []uint32
					if as, ok = assetMap[it]; !ok {
						as = make([]uint32, 0)
					}
					assetMap[it] = append(as, aid)
				}

				defer wg.Done()
				var cc []async.Provider[uuid.UUID]
				for _, it := range inventory.Types {
					if it == inventory.TypeValueEquip {
						cc = append(cc, compartment.AwaitEquipableCreated(l)(cid, equipment, assetMap[it]))
					} else {
						cc = append(cc, compartment.AwaitCreated(l)(cid, it, assetMap[it]))
					}
				}
				_, err = async.AwaitSlice[uuid.UUID](model.FixedProvider[[]async.Provider[uuid.UUID]](cc), async.SetTimeout(500*time.Millisecond), async.SetContext(ctx))()
				if err != nil {
					l.WithError(err).Errorf("Inventory generation for character [%d] failed.", cid)
				}
			}()

			//wg.Add(1)
			//go func() {
			//	defer wg.Done()
			//	createEquippedItems(l)(ctx)(cid, input)
			//}()
			//wg.Add(1)
			//go func() {
			//	defer wg.Done()
			//	createInventoryItems(l)(ctx)(cid, template.Items)
			//}()
			//
			wg.Wait()
			//
			//return character.GetById(l)(ctx)(cid)

			_ = producer.ProviderImpl(l)(ctx)(seed.EnvEventTopicStatus)(seed2.CreatedEventStatusProvider(input.AccountId, cid))
			return nil
		}

	}
}

func createInventoryItems(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, items []uint32) {
	return func(ctx context.Context) func(characterId uint32, items []uint32) {
		return func(characterId uint32, items []uint32) {
			// TODO
			//l.Debugf("Beginning inventory item creation for character [%d].", characterId)
			//ip := model.FixedProvider(items)
			//_, err := async.AwaitSlice[character.ItemGained](model.SliceMap(asyncItemCreate(l)(characterId))(ip)(), async.SetTimeout(1*time.Second), async.SetContext(ctx))()
			//if err != nil {
			//	l.WithError(err).Errorf("Error creating an item for character [%d].", characterId)
			//}
		}
	}
}

func createEquippedItems(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, input RestModel) {
	return func(ctx context.Context) func(characterId uint32, input RestModel) {
		return func(characterId uint32, input RestModel) {
			// TODO
			//l.Debugf("Beginning equipped item creation for character [%d].", characterId)
			//ip := model.FixedProvider([]uint32{input.Top, input.Bottom, input.Shoes, input.Weapon})
			//items, err := async.AwaitSlice[character.ItemGained](model.SliceMap(asyncItemCreate(l)(characterId))(ip)(), async.SetTimeout(1*time.Second), async.SetContext(ctx))()
			//if err != nil {
			//	l.WithError(err).Errorf("Error creating an item for character [%d].", characterId)
			//}
			//
			//_, err = async.AwaitSlice[uint32](model.SliceMap(asyncEquipItem(l)(characterId))(model.FixedProvider(items))(), async.SetTimeout(1*time.Second), async.SetContext(ctx))()
			//if err != nil {
			//	l.WithError(err).Errorf("Error equipping an item for character [%d].", characterId)
			//}
		}
	}
}

//func asyncItemCreate(l logrus.FieldLogger) func(characterId uint32) func(itemId uint32) (async.Provider[character.ItemGained], error) {
//	return func(characterId uint32) func(itemId uint32) (async.Provider[character.ItemGained], error) {
//		return func(itemId uint32) (async.Provider[character.ItemGained], error) {
//			return func(ctx context.Context, rchan chan character.ItemGained, echan chan error) {
//				character2.AwaitItemGained(l)(characterId)(itemId)(ctx, rchan, echan)
//				_, err := character.CreateItem(l)(ctx)(characterId, itemId)
//				if err != nil {
//					l.WithError(err).Errorf("Unable to create item [%d] from seed for character [%d].", itemId, characterId)
//					echan <- err
//				}
//			}, nil
//		}
//	}
//}
//
//func asyncEquipItem(l logrus.FieldLogger) func(characterId uint32) func(character.ItemGained) (async.Provider[uint32], error) {
//	return func(characterId uint32) func(character.ItemGained) (async.Provider[uint32], error) {
//		return func(ig character.ItemGained) (async.Provider[uint32], error) {
//			return func(ctx context.Context, rchan chan uint32, echan chan error) {
//				character2.AwaitEquipChanged(l)(characterId)(ig.ItemId)(ctx, rchan, echan)
//				err := character.EquipItem(l)(ctx)(characterId, ig.ItemId, ig.Slot)
//				if err != nil {
//					l.WithError(err).Errorf("Unable to equip item [%d] for character [%d].", ig.ItemId, characterId)
//					echan <- err
//				}
//			}, nil
//		}
//	}
//}

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
