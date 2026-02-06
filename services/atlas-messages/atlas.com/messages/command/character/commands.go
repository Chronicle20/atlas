package character

import (
	"atlas-messages/character"
	"atlas-messages/command"
	_map "atlas-messages/map"
	"atlas-messages/saga"
	"regexp"
	"strconv"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-constants/job"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

func AwardExperienceCommandProducer(l logrus.FieldLogger) func(ctx context.Context) func(ch channel.Model, c character.Model, m string) (command.Executor, bool) {
	return func(ctx context.Context) func(ch channel.Model, c character.Model, m string) (command.Executor, bool) {
		cp := character.NewProcessor(l, ctx)
		mp := _map.NewProcessor(l, ctx)
		sp := saga.NewProcessor(l, ctx)
		return func(ch channel.Model, c character.Model, m string) (command.Executor, bool) {
			var cn string
			var amountStr string

			re := regexp.MustCompile(`@award\s+(\w+)\s+experience\s+(\d+)`)
			match := re.FindStringSubmatch(m)
			if len(match) == 3 {
				cn = match[1]
				amountStr = match[2]
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

			tAmount, err := strconv.ParseUint(amountStr, 10, 32)
			if err != nil {
				return nil, false
			}
			amount := uint32(tAmount)

			return func(l logrus.FieldLogger) func(ctx context.Context) error {
				return func(ctx context.Context) error {
					var cids []uint32
					cids, err = idProvider()
					if err != nil {
						return err
					}
					for _, id := range cids {
						s, buildErr := saga.NewBuilder().
							SetSagaType(saga.QuestReward).
							SetInitiatedBy("COMMAND").
							AddStep("give_experience", saga.Pending, saga.AwardExperience, saga.AwardExperiencePayload{
								CharacterId: id,
								WorldId:     ch.WorldId(),
								ChannelId:   ch.Id(),
								Distributions: []saga.ExperienceDistributions{{
									ExperienceType: "WHITE",
									Amount:         amount,
								}},
							}).
							Build()
						if buildErr != nil {
							l.WithError(buildErr).Errorf("Unable to build saga for experience award to [%d].", id)
							continue
						}
						err = sp.Create(s)
						if err != nil {
							l.WithError(err).Errorf("Unable to award [%d] with [%d] experience.", id, amount)
						}
					}
					return err
				}
			}, true
		}
	}
}

func AwardLevelCommandProducer(l logrus.FieldLogger) func(ctx context.Context) func(ch channel.Model, c character.Model, m string) (command.Executor, bool) {
	return func(ctx context.Context) func(ch channel.Model, c character.Model, m string) (command.Executor, bool) {
		cp := character.NewProcessor(l, ctx)
		mp := _map.NewProcessor(l, ctx)
		sp := saga.NewProcessor(l, ctx)
		return func(ch channel.Model, c character.Model, m string) (command.Executor, bool) {
			var cn string
			var amountStr string

			re := regexp.MustCompile(`@award\s+(\w+)\s+(\d+)\s+level`)
			match := re.FindStringSubmatch(m)
			if len(match) == 3 {
				cn = match[1]
				amountStr = match[2]
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

			tAmount, err := strconv.ParseUint(amountStr, 10, 8)
			if err != nil {
				return nil, false
			}
			amount := byte(tAmount)

			return func(l logrus.FieldLogger) func(ctx context.Context) error {
				return func(ctx context.Context) error {
					cids, err := idProvider()
					if err != nil {
						return err
					}
					for _, id := range cids {
						s, buildErr := saga.NewBuilder().
							SetSagaType(saga.QuestReward).
							SetInitiatedBy("COMMAND").
							AddStep("give_level", saga.Pending, saga.AwardLevel, saga.AwardLevelPayload{
								CharacterId: id,
								WorldId:     ch.WorldId(),
								ChannelId:   ch.Id(),
								Amount:      amount,
							}).
							Build()
						if buildErr != nil {
							l.WithError(buildErr).Errorf("Unable to build saga for level award to [%d].", id)
							continue
						}
						err = sp.Create(s)
						if err != nil {
							l.WithError(err).Errorf("Unable to award [%d] with [%d] level(s).", id, amount)
						}
					}
					return err
				}
			}, true
		}
	}
}

func ChangeJobCommandProducer(l logrus.FieldLogger) func(ctx context.Context) func(ch channel.Model, c character.Model, m string) (command.Executor, bool) {
	return func(ctx context.Context) func(ch channel.Model, c character.Model, m string) (command.Executor, bool) {
		cp := character.NewProcessor(l, ctx)
		sp := saga.NewProcessor(l, ctx)
		return func(ch channel.Model, c character.Model, m string) (command.Executor, bool) {
			var cn string
			var jobStr string

			re := regexp.MustCompile(`@change\s+(\w+)\s+job\s+(\d+)`)
			match := re.FindStringSubmatch(m)
			if len(match) == 3 {
				cn = match[1]
				jobStr = match[2]
			}

			if len(cn) == 0 {
				return nil, false
			}

			if !c.Gm() {
				l.Debugf("Ignoring character [%d] command [%s], because they are not a gm.", c.Id(), m)
				return nil, false
			}

			var idProvider model.Provider[[]uint32]
			if match[1] == "my" {
				idProvider = model.ToSliceProvider(model.FixedProvider(c.Id()))
			} else {
				idProvider = model.ToSliceProvider(cp.IdByNameProvider(match[1]))
			}

			tJobId, err := strconv.ParseUint(jobStr, 10, 16)
			if err != nil {
				return nil, false
			}
			jobId := uint16(tJobId)

			return func(l logrus.FieldLogger) func(ctx context.Context) error {
				return func(ctx context.Context) error {
					cids, err := idProvider()
					if err != nil {
						return err
					}
					for _, id := range cids {
						s, buildErr := saga.NewBuilder().
							SetSagaType(saga.QuestReward).
							SetInitiatedBy("COMMAND").
							AddStep("change_job", saga.Pending, saga.ChangeJob, saga.ChangeJobPayload{
								CharacterId: id,
								WorldId:     ch.WorldId(),
								ChannelId:   ch.Id(),
								JobId:       job.Id(jobId),
							}).
							Build()
						if buildErr != nil {
							l.WithError(buildErr).Errorf("Unable to build saga for job change for [%d].", id)
							continue
						}
						err = sp.Create(s)
						if err != nil {
							l.WithError(err).Errorf("Unable to change job for character [%d] to job [%d].", id, jobId)
						}
					}
					return err
				}
			}, true
		}
	}
}

func AwardMesoCommandProducer(l logrus.FieldLogger) func(ctx context.Context) func(ch channel.Model, c character.Model, m string) (command.Executor, bool) {
	return func(ctx context.Context) func(ch channel.Model, c character.Model, m string) (command.Executor, bool) {
		cp := character.NewProcessor(l, ctx)
		mp := _map.NewProcessor(l, ctx)
		sp := saga.NewProcessor(l, ctx)
		return func(ch channel.Model, c character.Model, m string) (command.Executor, bool) {
			var cn string
			var amountStr string

			re := regexp.MustCompile(`@award\s+(\w+)\s+meso\s+(-?\d+)`)
			match := re.FindStringSubmatch(m)
			if len(match) == 3 {
				cn = match[1]
				amountStr = match[2]
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

			tAmount, err := strconv.ParseInt(amountStr, 10, 32)
			if err != nil {
				return nil, false
			}
			amount := int32(tAmount)

			return func(l logrus.FieldLogger) func(ctx context.Context) error {
				return func(ctx context.Context) error {
					cids, err := idProvider()
					if err != nil {
						return err
					}
					for _, id := range cids {
						s, buildErr := saga.NewBuilder().
							SetSagaType(saga.QuestReward).
							SetInitiatedBy("COMMAND").
							AddStep("give_mesos", saga.Pending, saga.AwardMesos, saga.AwardMesosPayload{
								CharacterId: id,
								WorldId:     ch.WorldId(),
								ChannelId:   ch.Id(),
								ActorId:     c.Id(),
								ActorType:   "CHARACTER",
								Amount:      amount,
							}).
							Build()
						if buildErr != nil {
							l.WithError(buildErr).Errorf("Unable to build saga for meso award to [%d].", id)
							continue
						}
						err = sp.Create(s)
						if err != nil {
							l.WithError(err).Errorf("Unable to award [%d] with [%d] meso.", id, amount)
						}
					}
					return err
				}
			}, true
		}
	}
}
