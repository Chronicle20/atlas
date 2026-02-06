package help

import (
	"atlas-messages/character"
	"atlas-messages/command"
	"atlas-messages/message"
	"context"
	"regexp"
	"strings"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	"github.com/sirupsen/logrus"
)

var commandSyntaxList = []string{
	"@help - Display this list of available commands",
	"@warp <target> <mapId> - Warp a character to a map",
	"@query map - Display your current map ID",
	"@query rates - Display your current rates (exp, meso, drop)",
	"@award <target> experience <amount> - Award experience points",
	"@award <target> <amount> level - Award levels",
	"@award <target> meso <amount> - Award mesos (can be negative)",
	"@award <target> <currencyType> <amount> - Award currency (credit, points, prepaid)",
	"@award <target> item <itemId> [quantity] - Award items",
	"@change <target> job <jobId> - Change job",
	"@skill max <skillId> - Maximize a skill",
	"@skill reset <skillId> - Reset a skill",
	"@buff <target> <skillName> [duration] - Apply a buff by name",
	"@buff <target> #<skillId> [duration] - Apply a buff by ID",
	"@consume <target> <itemId> - Apply consumable item effects",
}

func HelpCommandProducer(_ logrus.FieldLogger) func(_ context.Context) func(ch channel.Model, c character.Model, m string) (command.Executor, bool) {
	return func(_ context.Context) func(ch channel.Model, c character.Model, m string) (command.Executor, bool) {
		return func(ch channel.Model, c character.Model, m string) (command.Executor, bool) {
			re := regexp.MustCompile(`^@help$`)
			match := re.FindStringSubmatch(m)
			if len(match) != 1 {
				return nil, false
			}

			if !c.Gm() {
				return nil, false
			}

			return func(l logrus.FieldLogger) func(ctx context.Context) error {
				return func(ctx context.Context) error {
					mp := message.NewProcessor(l, ctx)

					f := field.NewBuilder(ch.WorldId(), ch.Id(), c.MapId()).Build()
					helpText := strings.Join(commandSyntaxList, "\r\n")
					return mp.IssuePinkText(f, 0, helpText, []uint32{c.Id()})
				}
			}, true
		}
	}
}
