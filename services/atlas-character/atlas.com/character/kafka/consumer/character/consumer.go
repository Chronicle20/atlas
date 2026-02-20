package character

import (
	"atlas-character/character"
	consumer2 "atlas-character/kafka/consumer"
	character2 "atlas-character/kafka/message/character"
	"context"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("character_command")(character2.EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
			rf(consumer2.NewConfig(l)("character_event_status")(character2.EnvEventTopicCharacterStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
			rf(consumer2.NewConfig(l)("character_movement_command")(character2.EnvCommandTopicMovement)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
		return func(rf func(topic string, handler handler.Handler) (string, error)) error {
			var t string
			t, _ = topic.EnvProvider(l)(character2.EnvCommandTopic)()
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCreateCharacter(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleChangeMap(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleChangeJob(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleChangeHair(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleChangeFace(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleChangeSkin(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleAwardExperience(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleAwardLevel(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleRequestChangeMeso(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleRequestDropMeso(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleRequestChangeFame(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleRequestDistributeAp(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleRequestDistributeSp(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleChangeHP(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleChangeMP(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleSetHP(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleDeductExperience(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleResetStats(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleClampHP(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleClampMP(db)))); err != nil {
				return err
			}
			t, _ = topic.EnvProvider(l)(character2.EnvCommandTopicMovement)()
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleMovementEvent(db)))); err != nil {
				return err
			}
			t, _ = topic.EnvProvider(l)(character2.EnvEventTopicCharacterStatus)()
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleLevelChangedStatusEvent(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleJobChangedStatusEvent(db)))); err != nil {
				return err
			}
			return nil
		}
	}
}

func handleChangeMap(db *gorm.DB) func(l logrus.FieldLogger, ctx context.Context, c character2.Command[character2.ChangeMapBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, c character2.Command[character2.ChangeMapBody]) {
		if c.Type != character2.CommandChangeMap {
			return
		}

		f := field.NewBuilder(c.WorldId, c.Body.ChannelId, c.Body.MapId).SetInstance(c.Body.Instance).Build()
		err := character.NewProcessor(l, ctx, db).ChangeMapAndEmit(c.TransactionId, c.CharacterId, f, c.Body.PortalId)
		if err != nil {
			l.WithError(err).Errorf("Unable to change character [%d] map.", c.CharacterId)
		}
	}
}

func handleChangeJob(db *gorm.DB) func(l logrus.FieldLogger, ctx context.Context, c character2.Command[character2.ChangeJobCommandBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, c character2.Command[character2.ChangeJobCommandBody]) {
		if c.Type != character2.CommandChangeJob {
			return
		}

		cha := channel.NewModel(c.WorldId, c.Body.ChannelId)
		_ = character.NewProcessor(l, ctx, db).ChangeJobAndEmit(c.TransactionId, c.CharacterId, cha, c.Body.JobId)
	}
}

func handleChangeHair(db *gorm.DB) func(l logrus.FieldLogger, ctx context.Context, c character2.Command[character2.ChangeHairCommandBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, c character2.Command[character2.ChangeHairCommandBody]) {
		if c.Type != character2.CommandChangeHair {
			return
		}

		cha := channel.NewModel(c.WorldId, c.Body.ChannelId)
		_ = character.NewProcessor(l, ctx, db).ChangeHairAndEmit(c.TransactionId, c.CharacterId, cha, c.Body.StyleId)
	}
}

func handleChangeFace(db *gorm.DB) func(l logrus.FieldLogger, ctx context.Context, c character2.Command[character2.ChangeFaceCommandBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, c character2.Command[character2.ChangeFaceCommandBody]) {
		if c.Type != character2.CommandChangeFace {
			return
		}

		cha := channel.NewModel(c.WorldId, c.Body.ChannelId)
		_ = character.NewProcessor(l, ctx, db).ChangeFaceAndEmit(c.TransactionId, c.CharacterId, cha, c.Body.StyleId)
	}
}

func handleChangeSkin(db *gorm.DB) func(l logrus.FieldLogger, ctx context.Context, c character2.Command[character2.ChangeSkinCommandBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, c character2.Command[character2.ChangeSkinCommandBody]) {
		if c.Type != character2.CommandChangeSkin {
			return
		}

		cha := channel.NewModel(c.WorldId, c.Body.ChannelId)
		_ = character.NewProcessor(l, ctx, db).ChangeSkinAndEmit(c.TransactionId, c.CharacterId, cha, c.Body.StyleId)
	}
}

func handleAwardExperience(db *gorm.DB) func(l logrus.FieldLogger, ctx context.Context, c character2.Command[character2.AwardExperienceCommandBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, c character2.Command[character2.AwardExperienceCommandBody]) {
		if c.Type != character2.CommandAwardExperience {
			return
		}

		es, err := model.SliceMap(func(m character2.ExperienceDistributions) (character.ExperienceModel, error) {
			return character.NewExperienceModel(m.ExperienceType, m.Amount, m.Attr1), nil
		})(model.FixedProvider(c.Body.Distributions))()()
		if err != nil {
			return
		}

		cha := channel.NewModel(c.WorldId, c.Body.ChannelId)
		_ = character.NewProcessor(l, ctx, db).AwardExperienceAndEmit(c.TransactionId, c.CharacterId, cha, es)
	}
}

func handleAwardLevel(db *gorm.DB) func(l logrus.FieldLogger, ctx context.Context, c character2.Command[character2.AwardLevelCommandBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, c character2.Command[character2.AwardLevelCommandBody]) {
		if c.Type != character2.CommandAwardLevel {
			return
		}

		cha := channel.NewModel(c.WorldId, c.Body.ChannelId)
		_ = character.NewProcessor(l, ctx, db).AwardLevelAndEmit(c.TransactionId, c.CharacterId, cha, c.Body.Amount)
	}
}

func handleRequestChangeMeso(db *gorm.DB) message.Handler[character2.Command[character2.RequestChangeMesoBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c character2.Command[character2.RequestChangeMesoBody]) {
		if c.Type != character2.CommandRequestChangeMeso {
			return
		}

		_ = character.NewProcessor(l, ctx, db).RequestChangeMeso(c.TransactionId, c.CharacterId, c.Body.Amount, c.Body.ActorId, c.Body.ActorType)
	}
}

func handleRequestDropMeso(db *gorm.DB) message.Handler[character2.Command[character2.RequestDropMesoCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c character2.Command[character2.RequestDropMesoCommandBody]) {
		if c.Type != character2.CommandRequestDropMeso {
			return
		}

		f := field.NewBuilder(c.WorldId, c.Body.ChannelId, c.Body.MapId).Build()
		_ = character.NewProcessor(l, ctx, db).RequestDropMeso(c.TransactionId, f, c.CharacterId, c.Body.Amount)
	}
}

func handleRequestChangeFame(db *gorm.DB) message.Handler[character2.Command[character2.RequestChangeFameBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c character2.Command[character2.RequestChangeFameBody]) {
		if c.Type != character2.CommandRequestChangeFame {
			return
		}

		_ = character.NewProcessor(l, ctx, db).RequestChangeFame(c.TransactionId, c.CharacterId, c.Body.Amount, c.Body.ActorId, c.Body.ActorType)
	}
}

func handleRequestDistributeAp(db *gorm.DB) message.Handler[character2.Command[character2.RequestDistributeApCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c character2.Command[character2.RequestDistributeApCommandBody]) {
		if c.Type != character2.CommandRequestDistributeAp {
			return
		}

		dp := model.SliceMap(transform)(model.FixedProvider(c.Body.Distributions))()
		ds, err := model.FilteredProvider(dp, model.Filters[character.Distribution](func(d character.Distribution) bool {
			return d.Amount > 0
		}))()
		if err != nil {
			return
		}
		_ = character.NewProcessor(l, ctx, db).RequestDistributeAp(c.TransactionId, c.CharacterId, ds)
	}
}

func handleRequestDistributeSp(db *gorm.DB) message.Handler[character2.Command[character2.RequestDistributeSpCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c character2.Command[character2.RequestDistributeSpCommandBody]) {
		if c.Type != character2.CommandRequestDistributeSp {
			return
		}
		_ = character.NewProcessor(l, ctx, db).RequestDistributeSp(c.TransactionId, c.CharacterId, c.Body.SkillId, c.Body.Amount)
	}
}

func transform(m character2.DistributePair) (character.Distribution, error) {
	return character.Distribution{
		Ability: m.Ability,
		Amount:  m.Amount,
	}, nil
}

func handleChangeHP(db *gorm.DB) message.Handler[character2.Command[character2.ChangeHPBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c character2.Command[character2.ChangeHPBody]) {
		if c.Type != character2.CommandChangeHP {
			return
		}

		cha := channel.NewModel(c.WorldId, c.Body.ChannelId)
		_ = character.NewProcessor(l, ctx, db).ChangeHPAndEmit(c.TransactionId, cha, c.CharacterId, c.Body.Amount)
	}
}

func handleChangeMP(db *gorm.DB) message.Handler[character2.Command[character2.ChangeMPBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c character2.Command[character2.ChangeMPBody]) {
		if c.Type != character2.CommandChangeMP {
			return
		}

		cha := channel.NewModel(c.WorldId, c.Body.ChannelId)
		_ = character.NewProcessor(l, ctx, db).ChangeMPAndEmit(c.TransactionId, cha, c.CharacterId, c.Body.Amount)
	}
}

func handleSetHP(db *gorm.DB) message.Handler[character2.Command[character2.SetHPBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c character2.Command[character2.SetHPBody]) {
		if c.Type != character2.CommandSetHP {
			return
		}

		cha := channel.NewModel(c.WorldId, c.Body.ChannelId)
		_ = character.NewProcessor(l, ctx, db).SetHPAndEmit(c.TransactionId, cha, c.CharacterId, c.Body.Amount)
	}
}

func handleDeductExperience(db *gorm.DB) message.Handler[character2.Command[character2.DeductExperienceCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c character2.Command[character2.DeductExperienceCommandBody]) {
		if c.Type != character2.CommandDeductExperience {
			return
		}

		cha := channel.NewModel(c.WorldId, c.Body.ChannelId)
		_ = character.NewProcessor(l, ctx, db).DeductExperienceAndEmit(c.TransactionId, c.CharacterId, cha, c.Body.Amount)
	}
}

func handleLevelChangedStatusEvent(db *gorm.DB) message.Handler[character2.StatusEvent[character2.LevelChangedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.LevelChangedStatusEventBody]) {
		if e.Type != character2.StatusEventTypeLevelChanged {
			return
		}

		cha := channel.NewModel(e.WorldId, e.Body.ChannelId)
		_ = character.NewProcessor(l, ctx, db).ProcessLevelChangeAndEmit(e.TransactionId, cha, e.CharacterId, e.Body.Amount)
	}
}

func handleJobChangedStatusEvent(db *gorm.DB) message.Handler[character2.StatusEvent[character2.JobChangedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.JobChangedStatusEventBody]) {
		if e.Type != character2.StatusEventTypeJobChanged {
			return
		}
		cha := channel.NewModel(e.WorldId, e.Body.ChannelId)
		_ = character.NewProcessor(l, ctx, db).ProcessJobChangeAndEmit(e.TransactionId, cha, e.CharacterId, e.Body.JobId)
	}
}

func handleCreateCharacter(db *gorm.DB) message.Handler[character2.Command[character2.CreateCharacterCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c character2.Command[character2.CreateCharacterCommandBody]) {
		if c.Type != character2.CommandCreateCharacter {
			return
		}

		model := character.NewModelBuilder().
			SetAccountId(c.Body.AccountId).
			SetWorldId(c.Body.WorldId).
			SetName(c.Body.Name).
			SetLevel(c.Body.Level).
			SetStrength(c.Body.Strength).
			SetDexterity(c.Body.Dexterity).
			SetIntelligence(c.Body.Intelligence).
			SetLuck(c.Body.Luck).
			SetMaxHp(c.Body.MaxHp).SetHp(c.Body.MaxHp).
			SetMaxMp(c.Body.MaxMp).SetMp(c.Body.MaxMp).
			SetJobId(c.Body.JobId).
			SetGender(c.Body.Gender).
			SetHair(c.Body.Hair).
			SetFace(c.Body.Face).
			SetSkinColor(c.Body.SkinColor).
			SetMapId(c.Body.MapId).
			Build()

		_, _ = character.NewProcessor(l, ctx, db).CreateAndEmit(c.TransactionId, model)
	}
}

func handleMovementEvent(db *gorm.DB) message.Handler[character2.MovementCommand] {
	return func(l logrus.FieldLogger, ctx context.Context, c character2.MovementCommand) {
		err := character.NewProcessor(l, ctx, db).Move(uint32(c.ObjectId), c.X, c.Y, c.Stance)
		if err != nil {
			l.WithError(err).Errorf("Error processing movement for character [%d].", c.ObjectId)
		}
	}
}

func handleResetStats(db *gorm.DB) message.Handler[character2.Command[character2.ResetStatsCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c character2.Command[character2.ResetStatsCommandBody]) {
		if c.Type != character2.CommandResetStats {
			return
		}

		cha := channel.NewModel(c.WorldId, c.Body.ChannelId)
		_ = character.NewProcessor(l, ctx, db).ResetStatsAndEmit(c.TransactionId, c.CharacterId, cha)
	}
}

func handleClampHP(db *gorm.DB) message.Handler[character2.Command[character2.ClampHPBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c character2.Command[character2.ClampHPBody]) {
		if c.Type != character2.CommandClampHP {
			return
		}

		cha := channel.NewModel(c.WorldId, c.Body.ChannelId)
		_ = character.NewProcessor(l, ctx, db).ClampHPAndEmit(c.TransactionId, cha, c.CharacterId, c.Body.MaxValue)
	}
}

func handleClampMP(db *gorm.DB) message.Handler[character2.Command[character2.ClampMPBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c character2.Command[character2.ClampMPBody]) {
		if c.Type != character2.CommandClampMP {
			return
		}

		cha := channel.NewModel(c.WorldId, c.Body.ChannelId)
		_ = character.NewProcessor(l, ctx, db).ClampMPAndEmit(c.TransactionId, cha, c.CharacterId, c.Body.MaxValue)
	}
}
