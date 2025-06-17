package factory

import (
	asset2 "atlas-character-factory/asset"
	"atlas-character-factory/character"
	compartment2 "atlas-character-factory/compartment"
	"atlas-character-factory/configuration"
	"atlas-character-factory/configuration/tenant/characters/template"
	"atlas-character-factory/kafka/consumer/asset"
	character2 "atlas-character-factory/kafka/consumer/character"
	"atlas-character-factory/kafka/consumer/compartment"
	"atlas-character-factory/kafka/message/seed"
	"atlas-character-factory/kafka/producer"
	seed2 "atlas-character-factory/kafka/producer/seed"
	"context"
	"errors"
	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-constants/item"
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

			// prepare assets for creation.
			assetMap := make(map[inventory.Type][]uint32)
			for _, aid := range template.Items {
				it, ok := inventory.TypeFromItemId(item.Id(aid))
				if !ok {
					continue
				}
				var as []uint32
				if as, ok = assetMap[it]; !ok {
					as = make([]uint32, 0)
				}
				assetMap[it] = append(as, aid)
			}

			wg := sync.WaitGroup{}
			var invErr error
			for _, it := range inventory.Types {
				wg.Add(1)
				go func() {
					defer wg.Done()

					ap := compartment.AwaitCreated(l)(cid, it, assetMap[it])
					compartmentId, eqpErr := async.Await[uuid.UUID](model.FixedProvider(ap), async.SetTimeout(500*time.Millisecond), async.SetContext(ctx))()
					if eqpErr != nil {
						invErr = eqpErr
					}

					l.Debugf("Compartment [%s] of type [%d] created for character [%d].", compartmentId.String(), it, cid)
					if it == inventory.TypeValueEquip {
						l.Debugf("Creating equipment for character [%d]. Starting item processing.", cid)
						equipment := []uint32{input.Top, input.Bottom, input.Shoes, input.Weapon}
						for _, aid := range equipment {
							var a asset2.Model
							a, err = async.Await[asset2.Model](model.FixedProvider(asset.AwaitCreated(l)(cid, compartmentId, aid, it)), async.SetTimeout(500*time.Millisecond), async.SetContext(ctx))()
							if err != nil {
								invErr = eqpErr
							}
							asyncEquip := func(actx context.Context, rchan chan uint32, echan chan error) {
								asset.AwaitSlotUpdate(l)(cid, compartmentId, a.Id())(actx, rchan, echan)
								err = compartment2.NewProcessor(l, actx).EquipAsset(cid, it, a.Slot(), 0)
								if err != nil {
									l.WithError(err).Errorf("Unable to equip asset [%d] character for character [%d].", a.Id(), cid)
									echan <- err
								}
							}
							_, err = async.Await[uint32](model.FixedProvider[async.Provider[uint32]](asyncEquip), async.SetTimeout(500*time.Millisecond), async.SetContext(ctx))()
							if err != nil {
								invErr = eqpErr
							}
						}
					}
					l.Debugf("Processing assets destined for compartment [%s] for character [%d].", compartmentId.String(), cid)
					for _, aid := range assetMap[it] {
						_, err = async.Await[asset2.Model](model.FixedProvider(asset.AwaitCreated(l)(cid, compartmentId, aid, it)), async.SetTimeout(500*time.Millisecond), async.SetContext(ctx))()
						if err != nil {
							invErr = eqpErr
						}
					}
				}()
			}

			wg.Wait()
			if invErr != nil {
				return invErr
			}

			_ = producer.ProviderImpl(l)(ctx)(seed.EnvEventTopicStatus)(seed2.CreatedEventStatusProvider(input.AccountId, cid))
			return nil
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
