package pet

import (
	"atlas-messages/character"
	"atlas-messages/command"
	"atlas-messages/kafka/message/pet"
	petlookup "atlas-messages/pet"
	"context"
	"regexp"
	"strconv"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

// AwardTamenessCommandProducer handles the GM command
// "@award <petName> tameness <amount>", raising the tameness/closeness of the
// acting character's pet identified by name (rather than all of their pets) by
// reusing atlas-pets' additive AWARD_CLOSENESS command. The pet name may contain
// spaces (e.g. "Baby Dragon"). Used to exercise the pet-evolution tameness gate.
func AwardTamenessCommandProducer(l logrus.FieldLogger) func(ctx context.Context) func(f field.Model, c character.Model, m string) (command.Executor, bool) {
	return func(ctx context.Context) func(f field.Model, c character.Model, m string) (command.Executor, bool) {
		pp := petlookup.NewProcessor(l, ctx)
		return func(f field.Model, c character.Model, m string) (command.Executor, bool) {
			re := regexp.MustCompile(`^@award\s+(.+?)\s+tameness\s+(\d+)$`)
			match := re.FindStringSubmatch(m)
			if len(match) != 3 {
				return nil, false
			}

			if !c.Gm() {
				l.Debugf("Ignoring character [%d] command [%s], because they are not a gm.", c.Id(), m)
				return nil, false
			}

			petName := match[1]

			tAmount, err := strconv.ParseUint(match[2], 10, 16)
			if err != nil {
				return nil, false
			}
			amount := uint16(tAmount)

			characterId := c.Id()
			return func(l logrus.FieldLogger) func(ctx context.Context) error {
				return func(ctx context.Context) error {
					petIds, err := pp.GetPetIdsByName(characterId, petName)
					if err != nil {
						return err
					}
					if len(petIds) == 0 {
						l.Warnf("Character [%d] has no pet named [%s]; nothing to award tameness to.", characterId, petName)
						return nil
					}
					for _, petId := range petIds {
						emitErr := producer.ProviderImpl(l)(ctx)(pet.EnvCommandTopic)(pet.AwardClosenessCommandProvider(petId, amount))
						if emitErr != nil {
							l.WithError(emitErr).Errorf("Unable to award [%d] tameness to pet [%d] of character [%d].", amount, petId, characterId)
						}
					}
					return nil
				}
			}, true
		}
	}
}
