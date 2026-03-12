package guild

import (
	"context"

	atlas_packet "github.com/Chronicle20/atlas-packet"
	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)

const (
	GuildOperationRequestName                    = "REQUEST_NAME"
	GuildOperationRequestAgreement               = "REQUEST_AGREEMENT"
	GuildOperationRequestEmblem                  = "REQUEST_EMBLEM"
	GuildOperationInvite                         = "INVITE"
	GuildOperationCreateErrorNameInUse           = "THE_NAME_IS_ALREADY_IN_USE_PLEASE_TRY_OTHER_ONES"
	GuildOperationCreateErrorDisagreed           = "SOMEBODY_HAS_DISAGREED_TO_FORM_A_GUILD"
	GuildOperationCreateError                    = "THE_PROBLEM_HAS_HAPPENED_DURING_THE_PROCESS_OF_FORMING_THE_GUILD_PLEASE_TRY_AGAIN"
	GuildOperationJoinSuccess                    = "JOIN_SUCCESS"
	GuildOperationJoinErrorAlreadyJoined         = "ALREADY_JOINED_THE_GUILD"
	GuildOperationJoinErrorMaxMembers            = "THE_GUILD_YOU_ARE_TRYING_TO_JOIN_HAS_ALREADY_REACHED_THE_MAX_NUMBER_OF_USERS"
	GuildOperationJoinErrorNotInChannel          = "THE_CHARACTER_CANNOT_BE_FOUND_IN_THE_CURRENT_CHANNEL"
	GuildOperationMemberQuitSuccess              = "MEMBER_QUIT_SUCCESS"
	GuildOperationMemberQuitErrorNotInGuild      = "MEMBER_QUIT_ERROR_NOT_IN_GUILD"
	GuildOperationMemberExpelledSuccess          = "MEMBER_EXPELLED_SUCCESS"
	GuildOperationMemberExpelledErrorNotInGuild  = "MEMBER_EXPELLED_ERROR_NOT_IN_GUILD"
	GuildOperationDisbandSuccess                 = "DISBAND_SUCCESS"
	GuildOperationDisbandError                   = "THE_PROBLEM_HAS_HAPPENED_DURING_THE_PROCESS_OF_DISBANDING_THE_GUILD_PLEASE_TRY_AGAIN"
	GuildOperationInviteErrorNotAcceptingInvites = "IS_CURRENTLY_NOT_ACCEPTING_GUILD_INVITE_MESSAGE"
	GuildOperationInviteErrorAnotherInvite       = "IS_TAKING_CARE_OF_ANOTHER_INVITATION"
	GuildOperationInviteDenied                   = "HAS_DENIED_YOUR_GUILD_INVITATION"
	GuildOperationCreateErrorCannotAsAdmin       = "ADMIN_CANNOT_MAKE_A_GUILD"
	GuildOperationIncreaseCapacitySuccess        = "CONGRATULATION_THE_NUMBER_OF_GUILD_MEMBERS_HAS_NOW_INCREASED_TO"
	GuildOperationIncreaseCapacityError          = "THE_PROBLEM_HAS_HAPPENED_DURING_THE_PROCESS_OF_INCREASING_THE_GUILD_PLEASE_TRY_AGAIN"
	GuildOperationMemberUpdate                   = "MEMBER_UPDATE"
	GuildOperationMemberOnline                   = "MEMBER_ONLINE"
	GuildOperationTitleUpdate                    = "TITLE_UPDATE"
	GuildOperationMemberTitleChange              = "MEMBER_TITLE_CHANGE"
	GuildOperationEmblemChange                   = "EMBLEM_CHANGE"
	GuildOperationNoticeChange                   = "NOTICE_CHANGE"
	GuildOperationShowTitles                     = "SHOW_TITLES"
	GuildOperationQuestErrorLessThanSixMembers   = "THERE_ARE_LESS_THAN_6_MEMBERS_REMAINING_SO_THE_QUEST_CANNOT_CONTINUE_YOUR_GUILD"
	GuildOperationQuestErrorDisconnected         = "THE_USER_THAT_REGISTERED_HAS_DISCONNECTED_SO_THE_QUEST_CANNOT_CONTINUE_YOUR_GUILD"
	GuildOperationQuestWaitingNotice             = "QUEST_WAITING_NOTICE"
	GuildOperationBoardAuthKeyUpdate             = "BOARD_AUTH_KEY_UPDATE"
	GuildOperationSetSkillResponse               = "SET_SKILL_RESPONSE"
)

func RequestGuildNameBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return GuildErrorBody(GuildOperationRequestName)
}

func RequestGuildEmblemBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return GuildErrorBody(GuildOperationRequestEmblem)
}

func GuildRequestAgreementBody(partyId uint32, leaderName string, guildName string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", GuildOperationRequestAgreement, func(mode byte) packet.Encoder {
		return NewRequestAgreement(mode, partyId, leaderName, guildName)
	})
}

func GuildErrorBody(code string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", code, func(mode byte) packet.Encoder {
		return NewErrorMessage(mode)
	})
}

func GuildErrorBody2(code string, target string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", code, func(mode byte) packet.Encoder {
		return NewErrorMessageWithTarget(mode, target)
	})
}

func GuildEmblemChangedBody(guildId uint32, logo uint16, logoColor byte, logoBackground uint16, logoBackgroundColor byte) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", GuildOperationEmblemChange, func(mode byte) packet.Encoder {
		return NewEmblemChange(mode, guildId, logo, logoColor, logoBackground, logoBackgroundColor)
	})
}

func GuildMemberStatusUpdatedBody(guildId uint32, characterId uint32, online bool) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", GuildOperationMemberOnline, func(mode byte) packet.Encoder {
		return NewMemberStatusUpdate(mode, guildId, characterId, online)
	})
}

func GuildMemberTitleUpdatedBody(guildId uint32, characterId uint32, title byte) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", GuildOperationMemberTitleChange, func(mode byte) packet.Encoder {
		return NewMemberTitleUpdate(mode, guildId, characterId, title)
	})
}

func GuildNoticeChangedBody(guildId uint32, notice string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", GuildOperationNoticeChange, func(mode byte) packet.Encoder {
		return NewNoticeChange(mode, guildId, notice)
	})
}

func GuildMemberLeftBody(guildId uint32, characterId uint32, name string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", GuildOperationMemberQuitSuccess, func(mode byte) packet.Encoder {
		return NewMemberLeft(mode, guildId, characterId, name)
	})
}

func GuildMemberExpelBody(guildId uint32, characterId uint32, name string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", GuildOperationMemberExpelledSuccess, func(mode byte) packet.Encoder {
		return NewMemberExpel(mode, guildId, characterId, name)
	})
}

func GuildMemberJoinedBody(guildId uint32, characterId uint32, name string, jobId uint16, level byte, title byte, online bool, allianceTitle byte) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", GuildOperationJoinSuccess, func(mode byte) packet.Encoder {
		return NewMemberJoined(mode, guildId, characterId, name, jobId, level, title, online, allianceTitle)
	})
}

func GuildInviteBody(guildId uint32, originatorName string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", GuildOperationInvite, func(mode byte) packet.Encoder {
		return NewInvite(mode, guildId, originatorName)
	})
}

func GuildTitleChangedBody(guildId uint32, titles []string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	var t [5]string
	for i := range 5 {
		t[i] = titles[i]
	}
	return atlas_packet.WithResolvedCode("operations", GuildOperationTitleUpdate, func(mode byte) packet.Encoder {
		return NewTitleChange(mode, guildId, t)
	})
}

func GuildDisbandBody(guildId uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", GuildOperationDisbandSuccess, func(mode byte) packet.Encoder {
		return NewDisband(mode, guildId)
	})
}

func GuildCapacityChangedBody(guildId uint32, capacity uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", GuildOperationIncreaseCapacitySuccess, func(mode byte) packet.Encoder {
		return NewCapacityChange(mode, guildId, capacity)
	})
}

func GuildInfoBody(inGuild bool, guildId uint32, name string, titles [5]string, members []GuildMemberInfo, capacity uint32, logoBackground uint16, logoBackgroundColor byte, logo uint16, logoColor byte, notice string, points uint32, allianceId uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return NewInfo(inGuild, guildId, name, titles, members, capacity, logoBackground, logoBackgroundColor, logo, logoColor, notice, points, allianceId).Encode
}
