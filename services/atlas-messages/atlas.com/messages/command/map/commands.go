package _map

import (
	"atlas-messages/character"
	"atlas-messages/command"
	"atlas-messages/map"
	"atlas-messages/message"
	"atlas-messages/rate"
	"atlas-messages/saga"
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	_map2 "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
)

func WarpCommandProducer(l logrus.FieldLogger) func(ctx context.Context) func(ch channel.Model, c character.Model, m string) (command.Executor, bool) {
	return func(ctx context.Context) func(ch channel.Model, c character.Model, m string) (command.Executor, bool) {
		return func(ch channel.Model, c character.Model, m string) (command.Executor, bool) {
			re := regexp.MustCompile(`@warp\s+(\w+)\s+(\d+)`)
			match := re.FindStringSubmatch(m)
			if len(match) != 3 {
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
				idProvider = _map.NewProcessor(l, ctx).CharacterIdsInMapStringProvider(ch, match[2])
			} else {
				idProvider = model.ToSliceProvider(character.NewProcessor(l, ctx).IdByNameProvider(match[1]))
			}

			return warpCommandProducer(ch, c.Id(), idProvider, match[2])

		}
	}
}

func warpCommandProducer(ch channel.Model, actorId uint32, idProvider model.Provider[[]uint32], mapStr string) (command.Executor, bool) {
	return func(l logrus.FieldLogger) func(ctx context.Context) error {
		return func(ctx context.Context) error {
			mp := _map.NewProcessor(l, ctx)
			sp := saga.NewProcessor(l, ctx)
			requestedMapId, err := strconv.ParseUint(mapStr, 10, 32)
			if err != nil {
				return errors.New("map does not exist")
			}

			exists := mp.Exists(_map2.Id(requestedMapId))
			if !exists {
				l.Debugf("Ignoring character [%d] command [%d], because they did not input a valid map.", actorId, requestedMapId)
				return errors.New("map does not exist")
			}

			ids, err := idProvider()
			if err != nil {
				return err
			}
			for _, id := range ids {
				s, buildErr := saga.NewBuilder().
					SetSagaType(saga.QuestReward).
					SetInitiatedBy("COMMAND").
					AddStep("warp_character", saga.Pending, saga.WarpToRandomPortal, saga.WarpToRandomPortalPayload{
						CharacterId: id,
						FieldId:     field.NewBuilder(ch.WorldId(), ch.Id(), _map2.Id(requestedMapId)).Build().Id(),
					}).
					Build()
				if buildErr != nil {
					l.WithError(buildErr).Errorf("Unable to build saga for warp to [%d] for character [%d].", requestedMapId, id)
					continue
				}
				err = sp.Create(s)
				if err != nil {
					l.WithError(err).Errorf("Unable to warp character [%d] via warp map command.", id)
				}
			}
			return err
		}
	}, true
}

func WhereAmICommandProducer(_ logrus.FieldLogger) func(_ context.Context) func(ch channel.Model, character character.Model, m string) (command.Executor, bool) {
	return func(_ context.Context) func(ch channel.Model, character character.Model, m string) (command.Executor, bool) {
		return func(ch channel.Model, character character.Model, m string) (command.Executor, bool) {
			re := regexp.MustCompile(`@query map`)
			match := re.FindStringSubmatch(m)
			if len(match) != 1 {
				return nil, false
			}

			return func(l logrus.FieldLogger) func(ctx context.Context) error {
				return func(ctx context.Context) error {
					f := field.NewBuilder(ch.WorldId(), ch.Id(), character.MapId()).Build()
					return message.NewProcessor(l, ctx).IssuePinkText(f, 0, "You are in map "+strconv.Itoa(int(character.MapId())), []uint32{character.Id()})
				}
			}, true
		}
	}
}

func RatesCommandProducer(_ logrus.FieldLogger) func(_ context.Context) func(ch channel.Model, character character.Model, m string) (command.Executor, bool) {
	return func(_ context.Context) func(ch channel.Model, character character.Model, m string) (command.Executor, bool) {
		return func(ch channel.Model, character character.Model, m string) (command.Executor, bool) {
			re := regexp.MustCompile(`^@query rates$`)
			match := re.FindStringSubmatch(m)
			if len(match) != 1 {
				return nil, false
			}

			if !character.Gm() {
				return nil, false
			}

			return func(l logrus.FieldLogger) func(ctx context.Context) error {
				return func(ctx context.Context) error {
					rp := rate.NewProcessor(l, ctx)
					mp := message.NewProcessor(l, ctx)
					f := field.NewBuilder(ch.WorldId(), ch.Id(), character.MapId()).Build()

					r, err := rp.GetByCharacter(ch, character.Id())
					if err != nil {
						l.WithError(err).Errorf("Unable to get rates for character [%d].", character.Id())
						return mp.IssuePinkText(f, 0, "Unable to retrieve rate information.", []uint32{character.Id()})
					}

					messages := buildRatesMessages(r)
					for _, msg := range messages {
						_ = mp.IssuePinkText(f, 0, msg, []uint32{character.Id()})
					}
					return nil
				}
			}, true
		}
	}
}

func buildRatesMessages(r rate.Model) []string {
	messages := []string{
		"=== Current Rates ===",
		fmt.Sprintf("EXP: %.2fx | Meso: %.2fx | Drop: %.2fx | Quest: %.2fx", r.ExpRate(), r.MesoRate(), r.ItemDropRate(), r.QuestExpRate()),
	}

	factors := r.Factors()
	if len(factors) > 0 {
		messages = append(messages, "=== Rate Factors ===")

		expFactors := r.FactorsByType("exp")
		mesoFactors := r.FactorsByType("meso")
		dropFactors := r.FactorsByType("item_drop")
		questExpFactors := r.FactorsByType("quest_exp")

		if len(expFactors) > 0 {
			messages = append(messages, "EXP: "+formatFactors(expFactors))
		}
		if len(mesoFactors) > 0 {
			messages = append(messages, "Meso: "+formatFactors(mesoFactors))
		}
		if len(dropFactors) > 0 {
			messages = append(messages, "Drop: "+formatFactors(dropFactors))
		}
		if len(questExpFactors) > 0 {
			messages = append(messages, "Quest: "+formatFactors(questExpFactors))
		}
	}

	return messages
}

func formatFactors(factors []rate.Factor) string {
	var parts []string
	for _, f := range factors {
		parts = append(parts, fmt.Sprintf("%s(%.2fx)", f.Source(), f.Multiplier()))
	}
	return strings.Join(parts, ", ")
}
