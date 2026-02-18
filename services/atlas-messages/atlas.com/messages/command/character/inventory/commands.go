package inventory

import (
	"atlas-messages/character"
	"atlas-messages/command"
	"atlas-messages/data/asset"
	_map "atlas-messages/map"
	"atlas-messages/saga"
	"context"
	"regexp"
	"strconv"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
)

func AwardItemCommandProducer(l logrus.FieldLogger) func(ctx context.Context) func(f field.Model, c character.Model, m string) (command.Executor, bool) {
	return func(ctx context.Context) func(f field.Model, c character.Model, m string) (command.Executor, bool) {
		cp := character.NewProcessor(l, ctx)
		ap := asset.NewProcessor(l, ctx)
		mp := _map.NewProcessor(l, ctx)
		return func(f field.Model, c character.Model, m string) (command.Executor, bool) {
			ch := f.Channel()
			var cn string
			var itemIdStr string
			var quantityStr string

			re := regexp.MustCompile(`@award\s+(\w+)\s+item\s+(\d+)\s+(\d+)`)
			match := re.FindStringSubmatch(m)
			if len(match) == 4 {
				cn = match[1]
				itemIdStr = match[2]
				quantityStr = match[3]
			} else {
				re = regexp.MustCompile(`@award\s+(\w+)\s+item\s+(\d+)`)
				match = re.FindStringSubmatch(m)
				if len(match) == 3 {
					cn = match[1]
					itemIdStr = match[2]
					quantityStr = "1"
				}
			}

			if len(cn) == 0 {
				return nil, false
			}

			if !c.Gm() {
				l.Debugf("Ignoring character [%d] command [%s], because they are not a gm.", c.Id(), m)
				return nil, false
			}

			var idProvider model.Provider[[]uint32]
			if match[1] == "me" {
				idProvider = model.ToSliceProvider(model.FixedProvider(c.Id()))
			} else if match[1] == "map" {
				f := field.NewBuilder(ch.WorldId(), ch.Id(), c.MapId()).Build()
				idProvider = mp.CharacterIdsInFieldProvider(f)
			} else {
				idProvider = model.ToSliceProvider(cp.IdByNameProvider(match[1]))
			}

			tItemId, err := strconv.ParseUint(itemIdStr, 10, 32)
			if err != nil {
				return nil, false
			}
			templateId := uint32(tItemId)
			exists := ap.Exists(templateId)
			if !exists {
				l.Debugf("Ignoring character [%d] command [%s], because they did not input a valid item.", c.Id(), m)
				return nil, false
			}

			tQuantity, err := strconv.ParseInt(quantityStr, 10, 16)
			if err != nil {
				return nil, false
			}
			quantity := uint32(tQuantity)

			return func(l logrus.FieldLogger) func(ctx context.Context) error {
				return func(ctx context.Context) error {
					sp := saga.NewProcessor(l, ctx)
					var cids []uint32
					cids, err = idProvider()
					if err != nil {
						return err
					}
					for _, id := range cids {
						s, buildErr := saga.NewBuilder().
							SetSagaType(saga.InventoryTransaction).
							SetInitiatedBy("atlas-messages").
							AddStep("give_item", saga.Pending, saga.AwardInventory, saga.AwardItemActionPayload{
								CharacterId: id,
								Item: saga.ItemPayload{
									TemplateId: templateId,
									Quantity:   quantity,
								},
							}).
							Build()
						if buildErr != nil {
							l.WithError(buildErr).Errorf("Unable to build saga for item award to [%d].", id)
							continue
						}
						err = sp.Create(s)
						if err != nil {
							l.WithError(err).Errorf("Unable to award [%d] with (%d) item [%d].", id, quantity, templateId)
						}
					}
					return err
				}
			}, true
		}
	}
}
