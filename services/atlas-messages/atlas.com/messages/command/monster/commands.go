package monster

import (
	"atlas-messages/character"
	"atlas-messages/command"
	"atlas-messages/data/foothold"
	monsterdata "atlas-messages/data/monster"
	"atlas-messages/kafka/message/monster"
	"atlas-messages/kafka/producer"
	"atlas-messages/message"
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	monster2 "github.com/Chronicle20/atlas/libs/atlas-constants/monster"
	"github.com/sirupsen/logrus"
)

var validStatuses = []string{
	"WEAPON_ATTACK", "WEAPON_DEFENSE", "MAGIC_ATTACK", "MAGIC_DEFENSE",
	"STUN", "FROZEN", "POISON", "SEAL", "SPEED",
	"POWER_UP", "MAGIC_UP", "POWER_GUARD_UP", "MAGIC_GUARD_UP",
	"WEAPON_ATTACK_IMMUNE", "MAGIC_ATTACK_IMMUNE",
	"HARD_SKIN", "WEAPON_COUNTER", "MAGIC_COUNTER",
	"MAGIC_CRASH",
	"SHOWDOWN", "NINJA_AMBUSH", "VENOM",
}

func MobStatusCommandProducer(l logrus.FieldLogger) func(ctx context.Context) func(f field.Model, c character.Model, m string) (command.Executor, bool) {
	return func(ctx context.Context) func(f field.Model, c character.Model, m string) (command.Executor, bool) {
		return func(f field.Model, c character.Model, m string) (command.Executor, bool) {
			ch := f.Channel()
			re := regexp.MustCompile(`^@mobstatus\s+(\S+)(?:\s+(\d+))?$`)
			match := re.FindStringSubmatch(m)
			if len(match) < 2 {
				return nil, false
			}

			if !c.Gm() {
				return nil, false
			}

			var skillId uint16
			input := match[1]
			if id, err := strconv.ParseUint(input, 10, 16); err == nil {
				skillId = uint16(id)
			} else if id, ok := monster2.SkillNameToId(strings.ToUpper(input)); ok {
				skillId = id
			} else {
				return func(l logrus.FieldLogger) func(ctx context.Context) error {
					return func(ctx context.Context) error {
						msgProc := message.NewProcessor(l, ctx)
						f := field.NewBuilder(ch.WorldId(), ch.Id(), c.MapId()).Build()
						_ = msgProc.IssuePinkText(f, 0, fmt.Sprintf("Unknown skill: %s", input), []uint32{c.Id()})
						return msgProc.IssuePinkText(f, 0, fmt.Sprintf("Valid: %s", strings.Join(monster2.SkillTypeNames(), ", ")), []uint32{c.Id()})
					}
				}, true
			}

			var skillLevel uint16 = 1
			if len(match) >= 3 && match[2] != "" {
				if lvl, err := strconv.ParseUint(match[2], 10, 16); err == nil {
					skillLevel = uint16(lvl)
				}
			}

			return func(l logrus.FieldLogger) func(ctx context.Context) error {
				return func(ctx context.Context) error {
					msgProc := message.NewProcessor(l, ctx)
					f := field.NewBuilder(ch.WorldId(), ch.Id(), c.MapId()).Build()

					err := producer.ProviderImpl(l)(ctx)(monster.EnvCommandTopic)(monster.UseSkillFieldCommandProvider(ch.WorldId(), ch.Id(), c.MapId(), f.Instance(), skillId, skillLevel))
					if err != nil {
						return msgProc.IssuePinkText(f, 0, "Failed to execute mob skill.", []uint32{c.Id()})
					}

					return msgProc.IssuePinkText(f, 0, fmt.Sprintf("Executing mob skill %d level %d on all monsters in map.", skillId, skillLevel), []uint32{c.Id()})
				}
			}, true
		}
	}
}

func MobKillAllCommandProducer(l logrus.FieldLogger) func(ctx context.Context) func(f field.Model, c character.Model, m string) (command.Executor, bool) {
	return func(ctx context.Context) func(f field.Model, c character.Model, m string) (command.Executor, bool) {
		return func(f field.Model, c character.Model, m string) (command.Executor, bool) {
			ch := f.Channel()
			re := regexp.MustCompile(`^@mob kill all$`)
			match := re.FindStringSubmatch(m)
			if match == nil {
				return nil, false
			}

			if !c.Gm() {
				return nil, false
			}

			return func(l logrus.FieldLogger) func(ctx context.Context) error {
				return func(ctx context.Context) error {
					msgProc := message.NewProcessor(l, ctx)
					f := field.NewBuilder(ch.WorldId(), ch.Id(), c.MapId()).Build()

					err := producer.ProviderImpl(l)(ctx)(monster.EnvCommandTopic)(monster.DestroyFieldCommandProvider(ch.WorldId(), ch.Id(), c.MapId(), f.Instance()))
					if err != nil {
						return msgProc.IssuePinkText(f, 0, "Failed to kill all monsters.", []uint32{c.Id()})
					}

					return msgProc.IssuePinkText(f, 0, "Killed all monsters in map.", []uint32{c.Id()})
				}
			}, true
		}
	}
}

func MobClearCommandProducer(l logrus.FieldLogger) func(ctx context.Context) func(f field.Model, c character.Model, m string) (command.Executor, bool) {
	return func(ctx context.Context) func(f field.Model, c character.Model, m string) (command.Executor, bool) {
		return func(f field.Model, c character.Model, m string) (command.Executor, bool) {
			ch := f.Channel()
			re := regexp.MustCompile(`^@mobclear(?:\s+(\w+))?$`)
			match := re.FindStringSubmatch(m)
			if match == nil {
				return nil, false
			}

			if !c.Gm() {
				return nil, false
			}

			var statusTypes []string
			if len(match) >= 2 && match[1] != "" {
				st := strings.ToUpper(match[1])
				if !isValidStatus(st) {
					return func(l logrus.FieldLogger) func(ctx context.Context) error {
						return func(ctx context.Context) error {
							msgProc := message.NewProcessor(l, ctx)
							f := field.NewBuilder(ch.WorldId(), ch.Id(), c.MapId()).Build()
							return msgProc.IssuePinkText(f, 0, fmt.Sprintf("Unknown status: %s", st), []uint32{c.Id()})
						}
					}, true
				}
				statusTypes = []string{st}
			}

			return func(l logrus.FieldLogger) func(ctx context.Context) error {
				return func(ctx context.Context) error {
					msgProc := message.NewProcessor(l, ctx)
					f := field.NewBuilder(ch.WorldId(), ch.Id(), c.MapId()).Build()

					err := producer.ProviderImpl(l)(ctx)(monster.EnvCommandTopic)(monster.CancelStatusFieldCommandProvider(ch.WorldId(), ch.Id(), c.MapId(), f.Instance(), statusTypes))
					if err != nil {
						return msgProc.IssuePinkText(f, 0, "Failed to clear monster statuses.", []uint32{c.Id()})
					}

					if len(statusTypes) > 0 {
						return msgProc.IssuePinkText(f, 0, fmt.Sprintf("Cleared %s from all monsters in map.", statusTypes[0]), []uint32{c.Id()})
					}
					return msgProc.IssuePinkText(f, 0, "Cleared all statuses from all monsters in map.", []uint32{c.Id()})
				}
			}, true
		}
	}
}

func isValidStatus(s string) bool {
	for _, v := range validStatuses {
		if v == s {
			return true
		}
	}
	return false
}

const spawnCountCap = 20

var spawnRe = regexp.MustCompile(`^@mob spawn\s+(\d+)(?:\s+(\d+))?$`)

// parseSpawnArgs extracts the template id and raw count from a "@mob spawn"
// message. ok is false when the message is not a spawn command. A non-numeric
// or overflowing count is normalized to spawnCountCap+1 so it clamps downstream.
func parseSpawnArgs(m string) (templateId uint32, rawCount int, ok bool) {
	match := spawnRe.FindStringSubmatch(m)
	if match == nil {
		return 0, 0, false
	}
	id, err := strconv.ParseUint(match[1], 10, 32)
	if err != nil {
		return 0, 0, false
	}
	rawCount = 1
	if match[2] != "" {
		c, cerr := strconv.Atoi(match[2])
		if cerr != nil {
			c = spawnCountCap + 1
		}
		rawCount = c
	}
	return uint32(id), rawCount, true
}

// normalizeCount validates and clamps the requested spawn count. valid is false
// when below 1; capped is true when the request exceeded the cap.
func normalizeCount(raw int) (count int, capped bool, valid bool) {
	if raw < 1 {
		return 0, false, false
	}
	if raw > spawnCountCap {
		return spawnCountCap, true, true
	}
	return raw, false, true
}

func MobSpawnCommandProducer(l logrus.FieldLogger) func(ctx context.Context) func(f field.Model, c character.Model, m string) (command.Executor, bool) {
	return func(ctx context.Context) func(f field.Model, c character.Model, m string) (command.Executor, bool) {
		return func(f field.Model, c character.Model, m string) (command.Executor, bool) {
			templateId, rawCount, ok := parseSpawnArgs(m)
			if !ok {
				return nil, false
			}

			if !c.Gm() {
				return nil, false
			}

			return func(l logrus.FieldLogger) func(ctx context.Context) error {
				return func(ctx context.Context) error {
					msgProc := message.NewProcessor(l, ctx)

					count, capped, valid := normalizeCount(rawCount)
					if !valid {
						return msgProc.IssuePinkText(f, 0, "Count must be at least 1.", []uint32{c.Id()})
					}

					mon, err := monsterdata.NewProcessor(l, ctx).GetById(templateId)
					if err != nil {
						return msgProc.IssuePinkText(f, 0, fmt.Sprintf("Unknown monster template: %d", templateId), []uint32{c.Id()})
					}

					var fh int16
					if fhModel, ferr := foothold.NewProcessor(l, ctx).GetBelow(f.MapId(), c.X(), c.Y()); ferr != nil {
						l.WithError(ferr).Warnf("Unable to resolve foothold below (%d, %d) on map [%d]; spawning with fh=0.", c.X(), c.Y(), uint32(f.MapId()))
					} else {
						fh = int16(fhModel.Id())
					}

					err = producer.ProviderImpl(l)(ctx)(monster.EnvCommandTopic)(monster.SpawnFieldCommandProvider(f.WorldId(), f.ChannelId(), f.MapId(), f.Instance(), templateId, c.X(), c.Y(), fh, 0, count))
					if err != nil {
						return msgProc.IssuePinkText(f, 0, fmt.Sprintf("Failed to spawn monster %d.", templateId), []uint32{c.Id()})
					}

					text := fmt.Sprintf("Spawned %dx monster %d (%s) at (%d, %d).", count, templateId, mon.Name(), c.X(), c.Y())
					if capped {
						text += fmt.Sprintf(" Capped to %d.", spawnCountCap)
					}
					return msgProc.IssuePinkText(f, 0, text, []uint32{c.Id()})
				}
			}, true
		}
	}
}
