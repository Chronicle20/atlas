package character

import (
	"atlas-messages/character"
	"atlas-messages/command"
	"atlas-messages/saga"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"regexp"
	"strconv"
	"strings"
)

func AwardCurrencyCommandProducer(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, c character.Model, m string) (command.Executor, bool) {
	return func(ctx context.Context) func(worldId byte, channelId byte, c character.Model, m string) (command.Executor, bool) {
		cp := character.NewProcessor(l, ctx)
		sp := saga.NewProcessor(l, ctx)
		return func(worldId byte, channelId byte, c character.Model, m string) (command.Executor, bool) {
			var cn string
			var currencyTypeStr string
			var amountStr string

			// Case-insensitive regex: @award <target> <credit|points|prepaid> <amount>
			re := regexp.MustCompile(`(?i)@award\s+(\w+)\s+(credit|points|prepaid)\s+(-?\d+)`)
			match := re.FindStringSubmatch(m)
			if len(match) == 4 {
				cn = match[1]
				currencyTypeStr = strings.ToLower(match[2])
				amountStr = match[3]
			}

			if len(cn) == 0 {
				return nil, false
			}

			if !c.Gm() {
				l.Debugf("Ignoring character [%d] command [%s], because they are not a gm.", c.Id(), m)
				return nil, false
			}

			// Map currency type string to numeric ID
			var currencyType uint32
			switch currencyTypeStr {
			case "credit":
				currencyType = 1
			case "points":
				currencyType = 2
			case "prepaid":
				currencyType = 3
			default:
				return nil, false
			}

			// Resolve target character - NO map support for currency awards
			var targetProvider model.Provider[character.Model]
			if match[1] == "me" {
				targetProvider = model.FixedProvider(c)
			} else {
				targetProvider = func() (character.Model, error) {
					return cp.GetByName()(match[1])
				}
			}

			tAmount, err := strconv.ParseInt(amountStr, 10, 32)
			if err != nil {
				return nil, false
			}
			amount := int32(tAmount)

			return func(l logrus.FieldLogger) func(ctx context.Context) error {
				return func(ctx context.Context) error {
					target, err := targetProvider()
					if err != nil {
						l.WithError(err).Errorf("Unable to find target character [%s] for currency award.", match[1])
						return err
					}

					s, buildErr := saga.NewBuilder().
						SetSagaType(saga.QuestReward).
						SetInitiatedBy("COMMAND").
						AddStep("award_currency", saga.Pending, saga.AwardCurrency, saga.AwardCurrencyPayload{
							CharacterId:  target.Id(),
							AccountId:    target.AccountId(),
							CurrencyType: currencyType,
							Amount:       amount,
						}).
						Build()
					if buildErr != nil {
						l.WithError(buildErr).Errorf("Unable to build saga for currency award to [%d].", target.Id())
						return buildErr
					}
					err = sp.Create(s)
					if err != nil {
						l.WithError(err).Errorf("Unable to award [%d] currency type [%d] amount [%d] to character [%d].", amount, currencyType, amount, target.Id())
					}
					return err
				}
			}, true
		}
	}
}
