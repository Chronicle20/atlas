package report

import (
	report2 "atlas-channel/kafka/message/report"

	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// sueCommandProvider builds the CREATE command for a /-command sue. Legacy
// clients (v83/v84/v87) supply accusedId with an empty subCommand; v95
// clients supply accusedId=0 with subCommand populated, which is forwarded
// as AccusedName. Resolution of whichever half is missing happens in
// atlas-ban.
func sueCommandProvider(reporterId uint32, worldId world.Id, channelId channel.Id, accusedId uint32, subCommand string, flag byte, reason string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(reporterId))
	value := &report2.Command[report2.CreateCommandBody]{
		Type: report2.CommandTypeCreate,
		Body: report2.CreateCommandBody{
			Kind:        report2.KindSue,
			WorldId:     worldId,
			ChannelId:   channelId,
			ReporterId:  reporterId,
			AccusedId:   accusedId,
			AccusedName: subCommand,
			ReasonType:  flag,
			Description: reason,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// claimCommandProvider builds the CREATE command for a CUIClaim report
// window submission. Claims always carry a target name.
func claimCommandProvider(reporterId uint32, worldId world.Id, channelId channel.Id, targetName string, reasonType byte, description string, chatClaim bool, chatLog string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(reporterId))
	value := &report2.Command[report2.CreateCommandBody]{
		Type: report2.CommandTypeCreate,
		Body: report2.CreateCommandBody{
			Kind:        report2.KindClaim,
			WorldId:     worldId,
			ChannelId:   channelId,
			ReporterId:  reporterId,
			AccusedName: targetName,
			ReasonType:  reasonType,
			Description: description,
			ChatClaim:   chatClaim,
			ChatLog:     chatLog,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
