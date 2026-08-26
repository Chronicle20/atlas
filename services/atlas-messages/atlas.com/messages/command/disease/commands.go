package disease

import (
	"atlas-messages/character"
	"atlas-messages/command"
	"atlas-messages/kafka/message/buff"
	_map "atlas-messages/map"
	"atlas-messages/message"
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

	"github.com/sirupsen/logrus"

	statconstant "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// validDiseases maps the user-typed GM command word to the canonical
// character.TemporaryStatType wire value. Both the disease name and the
// canonical stat name are accepted as keys where they differ (ZOMBIFY/UNDEAD,
// WEAKNESS/WEAKEN) so existing GM syntax keeps working.
var validDiseases = map[string]statconstant.TemporaryStatType{
	"SEAL":         statconstant.TemporaryStatTypeSeal,
	"DARKNESS":     statconstant.TemporaryStatTypeDarkness,
	"WEAKNESS":     statconstant.TemporaryStatTypeWeaken,
	"WEAKEN":       statconstant.TemporaryStatTypeWeaken,
	"STUN":         statconstant.TemporaryStatTypeStun,
	"CURSE":        statconstant.TemporaryStatTypeCurse,
	"POISON":       statconstant.TemporaryStatTypePoison,
	"SLOW":         statconstant.TemporaryStatTypeSlow,
	"SEDUCE":       statconstant.TemporaryStatTypeSeduce,
	"ZOMBIFY":      statconstant.TemporaryStatTypeUndead,
	"UNDEAD":       statconstant.TemporaryStatTypeUndead,
	"CONFUSE":      statconstant.TemporaryStatTypeConfuse,
	"STOP_PORTION": statconstant.TemporaryStatTypeStopPortion,
}

func DiseaseCommandProducer(l logrus.FieldLogger) func(ctx context.Context) func(f field.Model, c character.Model, m string) (command.Executor, bool) {
	return func(ctx context.Context) func(f field.Model, c character.Model, m string) (command.Executor, bool) {
		cp := character.NewProcessor(l, ctx)
		mp := _map.NewProcessor(l, ctx)
		return func(f field.Model, c character.Model, m string) (command.Executor, bool) {
			ch := f.Channel()
			re := regexp.MustCompile(`^@disease\s+(\w+)\s+(\w+)(?:\s+(-?\d+))?(?:\s+(\d+))?$`)
			match := re.FindStringSubmatch(m)
			if len(match) < 3 {
				return nil, false
			}

			if !c.Gm() {
				return nil, false
			}

			target := match[1]
			diseaseType := strings.ToUpper(match[2])

			var value int32 = 1
			if len(match) >= 4 && match[3] != "" {
				val, err := strconv.ParseInt(match[3], 10, 32)
				if err == nil {
					value = int32(val)
				}
			}

			var duration int32 = 10000
			if len(match) >= 5 && match[4] != "" {
				dur, err := strconv.ParseInt(match[4], 10, 32)
				if err == nil {
					duration = int32(dur)
				}
			}

			statName, ok := validDiseases[diseaseType]
			if !ok {
				names := make([]string, 0, len(validDiseases))
				for k := range validDiseases {
					names = append(names, k)
				}
				return func(l logrus.FieldLogger) func(ctx context.Context) error {
					return func(ctx context.Context) error {
						msgProc := message.NewProcessor(l, ctx)
						f := field.NewBuilder(ch.WorldId(), ch.Id(), f.MapId()).Build()
						_ = msgProc.IssuePinkText(f, 0, fmt.Sprintf("Unknown disease: %s", diseaseType), []uint32{c.Id()})
						return msgProc.IssuePinkText(f, 0, fmt.Sprintf("Valid: %s", strings.Join(names, ", ")), []uint32{c.Id()})
					}
				}, true
			}

			var idProvider model.Provider[[]uint32]
			if target == "me" {
				idProvider = model.ToSliceProvider(model.FixedProvider(c.Id()))
			} else if target == "map" {
				f := field.NewBuilder(ch.WorldId(), ch.Id(), f.MapId()).Build()
				idProvider = mp.CharacterIdsInFieldProvider(f)
			} else {
				idProvider = model.ToSliceProvider(cp.IdByNameProvider(target))
			}

			return func(l logrus.FieldLogger) func(ctx context.Context) error {
				return func(ctx context.Context) error {
					msgProc := message.NewProcessor(l, ctx)
					f := field.NewBuilder(ch.WorldId(), ch.Id(), f.MapId()).Build()

					ids, err := idProvider()
					if err != nil {
						return msgProc.IssuePinkText(f, 0, "Unable to resolve target.", []uint32{c.Id()})
					}

					if len(ids) == 0 {
						return msgProc.IssuePinkText(f, 0, "No targets found.", []uint32{c.Id()})
					}

					changes := []buff.StatChange{{Type: string(statName), Amount: value}}

					for _, id := range ids {
						err = producer.ProviderImpl(l)(ctx)(buff.EnvCommandTopic)(buff.ApplyCommandProvider(f, id, 0, 0, 1, duration, changes))
						if err != nil {
							l.WithError(err).Errorf("Unable to apply disease [%s] to character [%d].", statName, id)
						}
					}

					if len(ids) == 1 {
						return msgProc.IssuePinkText(f, 0, fmt.Sprintf("Applied %s to target.", diseaseType), []uint32{c.Id()})
					}
					return msgProc.IssuePinkText(f, 0, fmt.Sprintf("Applied %s to %d targets.", diseaseType, len(ids)), []uint32{c.Id()})
				}
			}, true
		}
	}
}
