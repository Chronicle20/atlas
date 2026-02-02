package help

import (
	"atlas-messages/character"
	"atlas-messages/command"
	"atlas-messages/message"
	"context"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"
)

var commandSyntaxList = []string{
	"@help - Display this list of available commands",
	"@warp <target> <mapId> - Warp a character to a map",
	"@query map - Display your current map ID",
	"@award <target> experience <amount> - Award experience points",
	"@award <target> <amount> level - Award levels",
	"@award <target> meso <amount> - Award mesos (can be negative)",
	"@award <target> <currencyType> <amount> - Award currency (credit, points, prepaid)",
	"@award <target> item <itemId> [quantity] - Award items",
	"@change <target> job <jobId> - Change job",
	"@skill max <skillId> - Maximize a skill",
	"@skill reset <skillId> - Reset a skill",
	"@buff <target> <skillName> [duration] - Apply a buff (target: me, map, name)",
}

func HelpCommandProducer(_ logrus.FieldLogger) func(_ context.Context) func(worldId byte, channelId byte, c character.Model, m string) (command.Executor, bool) {
	return func(_ context.Context) func(worldId byte, channelId byte, c character.Model, m string) (command.Executor, bool) {
		return func(worldId byte, channelId byte, c character.Model, m string) (command.Executor, bool) {
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

					helpText := strings.Join(commandSyntaxList, "\r\n")
					return mp.IssuePinkText(worldId, channelId, c.MapId(), 0, helpText, []uint32{c.Id()})
				}
			}, true
		}
	}
}
