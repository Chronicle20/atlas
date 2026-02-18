package consumable

import (
	"atlas-messages/character"
	"atlas-messages/command"
	_map "atlas-messages/map"
	"atlas-messages/message"
	"atlas-messages/saga"
	"context"
	"fmt"
	"regexp"
	"strconv"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
)

func ConsumeCommandProducer(l logrus.FieldLogger) func(ctx context.Context) func(f field.Model, c character.Model, m string) (command.Executor, bool) {
	return func(ctx context.Context) func(f field.Model, c character.Model, m string) (command.Executor, bool) {
		cp := character.NewProcessor(l, ctx)
		mp := _map.NewProcessor(l, ctx)
		sp := saga.NewProcessor(l, ctx)
		return func(f field.Model, c character.Model, m string) (command.Executor, bool) {
			ch := f.Channel()
			// @consume <target> <itemId>
			re := regexp.MustCompile(`^@consume\s+(\w+)\s+(\d+)$`)
			match := re.FindStringSubmatch(m)
			if len(match) != 3 {
				return nil, false
			}

			if !c.Gm() {
				l.Debugf("Ignoring character [%d] command [%s], because they are not a gm.", c.Id(), m)
				return nil, false
			}

			target := match[1]
			itemIdStr := match[2]

			itemId, err := strconv.ParseUint(itemIdStr, 10, 32)
			if err != nil {
				return nil, false
			}

			var idProvider model.Provider[[]uint32]
			if target == "me" {
				idProvider = model.ToSliceProvider(model.FixedProvider(c.Id()))
			} else if target == "map" {
				f := field.NewBuilder(ch.WorldId(), ch.Id(), c.MapId()).Build()
				idProvider = mp.CharacterIdsInFieldProvider(f)
			} else {
				idProvider = model.ToSliceProvider(cp.IdByNameProvider(target))
			}

			return func(l logrus.FieldLogger) func(ctx context.Context) error {
				return func(ctx context.Context) error {
					msgProc := message.NewProcessor(l, ctx)
					f := field.NewBuilder(ch.WorldId(), ch.Id(), c.MapId()).Build()

					cids, err := idProvider()
					if err != nil {
						l.WithError(err).Errorf("Unable to resolve consume target.")
						return msgProc.IssuePinkText(f, 0, "Unable to resolve target.", []uint32{c.Id()})
					}

					if len(cids) == 0 {
						return msgProc.IssuePinkText(f, 0, "No targets found.", []uint32{c.Id()})
					}

					for _, id := range cids {
						s, buildErr := saga.NewBuilder().
							SetSagaType(saga.QuestReward).
							SetInitiatedBy("COMMAND").
							AddStep("apply_consumable_effect", saga.Pending, saga.ApplyConsumableEffect, saga.ApplyConsumableEffectPayload{
								CharacterId: id,
								WorldId:     ch.WorldId(),
								ChannelId:   ch.Id(),
								ItemId:      uint32(itemId),
							}).
							Build()
						if buildErr != nil {
							l.WithError(buildErr).Errorf("Unable to build saga for apply consumable effect to [%d].", id)
							continue
						}
						err = sp.Create(s)
						if err != nil {
							l.WithError(err).Errorf("Unable to apply item [%d] effect to character [%d].", itemId, id)
						}
					}

					if len(cids) == 1 {
						return msgProc.IssuePinkText(f, 0, fmt.Sprintf("Applied item %d effects to target.", itemId), []uint32{c.Id()})
					}
					return msgProc.IssuePinkText(f, 0, fmt.Sprintf("Applied item %d effects to %d targets.", itemId, len(cids)), []uint32{c.Id()})
				}
			}, true
		}
	}
}
