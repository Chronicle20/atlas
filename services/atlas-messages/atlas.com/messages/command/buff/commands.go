package buff

import (
	"atlas-messages/buff"
	"atlas-messages/character"
	"atlas-messages/command"
	"atlas-messages/data/skill"
	"atlas-messages/map"
	"atlas-messages/message"
	"context"
	"fmt"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
	"regexp"
	"strconv"
	"strings"
)

func BuffCommandProducer(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, c character.Model, m string) (command.Executor, bool) {
	return func(ctx context.Context) func(worldId byte, channelId byte, c character.Model, m string) (command.Executor, bool) {
		cp := character.NewProcessor(l, ctx)
		sdp := skill.NewProcessor(l, ctx)
		mp := _map.NewProcessor(l, ctx)
		return func(worldId byte, channelId byte, c character.Model, m string) (command.Executor, bool) {
			re := regexp.MustCompile(`^@buff\s+(\w+)\s+"?([^"]+)"?(?:\s+(\d+))?$`)
			match := re.FindStringSubmatch(m)
			if len(match) < 3 {
				return nil, false
			}

			if !c.Gm() {
				l.Debugf("Ignoring character [%d] command [%s], because they are not a gm.", c.Id(), m)
				return nil, false
			}

			target := match[1]
			skillName := strings.TrimSpace(match[2])
			var durationOverride int32 = 0
			if len(match) >= 4 && match[3] != "" {
				dur, err := strconv.Atoi(match[3])
				if err == nil {
					durationOverride = int32(dur)
				}
			}

			skills, err := sdp.GetByName(skillName)
			if err != nil || len(skills) == 0 {
				return func(l logrus.FieldLogger) func(ctx context.Context) error {
					return func(ctx context.Context) error {
						msgProc := message.NewProcessor(l, ctx)
						return msgProc.IssuePinkText(worldId, channelId, c.MapId(), 0, fmt.Sprintf("Unknown skill: %s", skillName), []uint32{c.Id()})
					}
				}, true
			}

			foundSkill := skills[0]
			maxLevel := byte(len(foundSkill.Effects()))
			if maxLevel == 0 {
				return func(l logrus.FieldLogger) func(ctx context.Context) error {
					return func(ctx context.Context) error {
						msgProc := message.NewProcessor(l, ctx)
						return msgProc.IssuePinkText(worldId, channelId, c.MapId(), 0, fmt.Sprintf("Skill %s has no effects", foundSkill.Name()), []uint32{c.Id()})
					}
				}, true
			}

			var idProvider model.Provider[[]uint32]
			if target == "me" {
				idProvider = model.ToSliceProvider(model.FixedProvider(c.Id()))
			} else if target == "map" {
				idProvider = mp.CharacterIdsInMapProvider(worldId, channelId, c.MapId())
			} else {
				idProvider = model.ToSliceProvider(cp.IdByNameProvider(target))
			}

			return func(l logrus.FieldLogger) func(ctx context.Context) error {
				return func(ctx context.Context) error {
					bp := buff.NewProcessor(l, ctx)
					msgProc := message.NewProcessor(l, ctx)

					ids, err := idProvider()
					if err != nil {
						l.WithError(err).Errorf("Unable to resolve buff target.")
						return msgProc.IssuePinkText(worldId, channelId, c.MapId(), 0, "Unable to resolve target.", []uint32{c.Id()})
					}

					if len(ids) == 0 {
						return msgProc.IssuePinkText(worldId, channelId, c.MapId(), 0, "No targets found.", []uint32{c.Id()})
					}

					for _, id := range ids {
						err = bp.Apply(worldId, channelId, id, c.Id(), foundSkill.Id(), maxLevel, durationOverride)
						if err != nil {
							l.WithError(err).Errorf("Unable to apply buff [%d] to character [%d].", foundSkill.Id(), id)
						}
					}

					if len(ids) == 1 {
						return msgProc.IssuePinkText(worldId, channelId, c.MapId(), 0, fmt.Sprintf("Applied %s to target.", foundSkill.Name()), []uint32{c.Id()})
					}
					return msgProc.IssuePinkText(worldId, channelId, c.MapId(), 0, fmt.Sprintf("Applied %s to %d targets.", foundSkill.Name(), len(ids)), []uint32{c.Id()})
				}
			}, true
		}
	}
}
