package guild

import (
	"context"

	"github.com/sirupsen/logrus"

	atlas_packet "github.com/Chronicle20/atlas/libs/atlas-packet"
	"github.com/Chronicle20/atlas/libs/atlas-packet/guild/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

// GuildOperation result-mode keys (CWvsContext::OnGuildResult). Each resolves to
// the per-version sub-op MODE byte via the tenant "operations" table
// (docs/packets/dispatchers/guild.yaml). The mode byte is per-tenant/version
// DATA — never a struct literal. Body functions fix the key; the constructor
// receives the RESOLVED mode (config-driven contract, like the other dispatcher
// families). v83/v84/v87/jms are byte-identical; v95 mode bytes are shifted.
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

// --- Request-prompt arms (mode-only) -----------------------------------------

func RequestGuildNameBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", GuildOperationRequestName, func(mode byte) packet.Encoder {
		return clientbound.NewRequestName(mode)
	})
}

func RequestGuildEmblemBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", GuildOperationRequestEmblem, func(mode byte) packet.Encoder {
		return clientbound.NewRequestEmblem(mode)
	})
}

// --- Structured arms ---------------------------------------------------------

func GuildRequestAgreementBody(partyId uint32, leaderName string, guildName string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", GuildOperationRequestAgreement, func(mode byte) packet.Encoder {
		return clientbound.NewRequestAgreement(mode, partyId, leaderName, guildName)
	})
}

func GuildEmblemChangedBody(guildId uint32, logo uint16, logoColor byte, logoBackground uint16, logoBackgroundColor byte) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", GuildOperationEmblemChange, func(mode byte) packet.Encoder {
		return clientbound.NewEmblemChange(mode, guildId, logo, logoColor, logoBackground, logoBackgroundColor)
	})
}

func GuildMemberStatusUpdatedBody(guildId uint32, characterId uint32, online bool) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", GuildOperationMemberOnline, func(mode byte) packet.Encoder {
		return clientbound.NewMemberStatusUpdate(mode, guildId, characterId, online)
	})
}

func GuildMemberTitleUpdatedBody(guildId uint32, characterId uint32, title byte) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", GuildOperationMemberTitleChange, func(mode byte) packet.Encoder {
		return clientbound.NewMemberTitleUpdate(mode, guildId, characterId, title)
	})
}

func GuildNoticeChangedBody(guildId uint32, notice string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", GuildOperationNoticeChange, func(mode byte) packet.Encoder {
		return clientbound.NewNoticeChange(mode, guildId, notice)
	})
}

func GuildMemberLeftBody(guildId uint32, characterId uint32, name string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", GuildOperationMemberQuitSuccess, func(mode byte) packet.Encoder {
		return clientbound.NewMemberLeft(mode, guildId, characterId, name)
	})
}

func GuildMemberExpelBody(guildId uint32, characterId uint32, name string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", GuildOperationMemberExpelledSuccess, func(mode byte) packet.Encoder {
		return clientbound.NewMemberExpel(mode, guildId, characterId, name)
	})
}

func GuildMemberJoinedBody(guildId uint32, characterId uint32, name string, jobId uint16, level byte, title byte, online bool, allianceTitle byte) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", GuildOperationJoinSuccess, func(mode byte) packet.Encoder {
		return clientbound.NewMemberJoined(mode, guildId, characterId, name, jobId, level, title, online, allianceTitle)
	})
}

func GuildInviteBody(guildId uint32, originatorName string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", GuildOperationInvite, func(mode byte) packet.Encoder {
		return clientbound.NewInvite(mode, guildId, originatorName, 0, 0)
	})
}

func GuildTitleChangedBody(guildId uint32, titles []string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	var t [5]string
	for i := range 5 {
		t[i] = titles[i]
	}
	return atlas_packet.WithResolvedCode("operations", GuildOperationTitleUpdate, func(mode byte) packet.Encoder {
		return clientbound.NewTitleChange(mode, guildId, t)
	})
}

func GuildDisbandBody(guildId uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", GuildOperationDisbandSuccess, func(mode byte) packet.Encoder {
		return clientbound.NewDisband(mode, guildId)
	})
}

func GuildCapacityChangedBody(guildId uint32, capacity byte) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", GuildOperationIncreaseCapacitySuccess, func(mode byte) packet.Encoder {
		return clientbound.NewCapacityChange(mode, guildId, capacity)
	})
}

func GuildMemberUpdateBody(guildId uint32, characterId uint32, level uint32, job uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", GuildOperationMemberUpdate, func(mode byte) packet.Encoder {
		return clientbound.NewMemberUpdate(mode, guildId, characterId, level, job)
	})
}

func GuildShowTitlesBody(guildId uint32, entries []clientbound.GuildTitleEntry) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", GuildOperationShowTitles, func(mode byte) packet.Encoder {
		return clientbound.NewShowTitles(mode, guildId, entries)
	})
}

func GuildQuestWaitingNoticeBody(channel byte, state uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", GuildOperationQuestWaitingNotice, func(mode byte) packet.Encoder {
		return clientbound.NewQuestWaitingNotice(mode, channel, state)
	})
}

func GuildBoardAuthKeyUpdateBody(authKey string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", GuildOperationBoardAuthKeyUpdate, func(mode byte) packet.Encoder {
		return clientbound.NewBoardAuthKeyUpdate(mode, authKey)
	})
}

func GuildSetSkillResponseBody(success bool, message string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", GuildOperationSetSkillResponse, func(mode byte) packet.Encoder {
		return clientbound.NewSetSkillResponse(mode, success, message)
	})
}

// --- Target-bearing invite-error arms ({mode,target}) -------------------------

func GuildInviteErrorNotAcceptingInvitesBody(target string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", GuildOperationInviteErrorNotAcceptingInvites, func(mode byte) packet.Encoder {
		return clientbound.NewInviteErrorNotAcceptingInvites(mode, target)
	})
}

func GuildInviteErrorAnotherInviteBody(target string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", GuildOperationInviteErrorAnotherInvite, func(mode byte) packet.Encoder {
		return clientbound.NewInviteErrorAnotherInvite(mode, target)
	})
}

func GuildInviteDeniedBody(target string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", GuildOperationInviteDenied, func(mode byte) packet.Encoder {
		return clientbound.NewInviteDenied(mode, target)
	})
}

// --- Mode-only error/notice arms ---------------------------------------------

func GuildCreateErrorNameInUseBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", GuildOperationCreateErrorNameInUse, func(mode byte) packet.Encoder {
		return clientbound.NewCreateErrorNameInUse(mode)
	})
}

func GuildCreateErrorDisagreedBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", GuildOperationCreateErrorDisagreed, func(mode byte) packet.Encoder {
		return clientbound.NewCreateErrorDisagreed(mode)
	})
}

func GuildCreateErrorBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", GuildOperationCreateError, func(mode byte) packet.Encoder {
		return clientbound.NewCreateError(mode)
	})
}

func GuildJoinErrorAlreadyJoinedBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", GuildOperationJoinErrorAlreadyJoined, func(mode byte) packet.Encoder {
		return clientbound.NewJoinErrorAlreadyJoined(mode)
	})
}

func GuildJoinErrorMaxMembersBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", GuildOperationJoinErrorMaxMembers, func(mode byte) packet.Encoder {
		return clientbound.NewJoinErrorMaxMembers(mode)
	})
}

func GuildJoinErrorNotInChannelBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", GuildOperationJoinErrorNotInChannel, func(mode byte) packet.Encoder {
		return clientbound.NewJoinErrorNotInChannel(mode)
	})
}

func GuildMemberQuitErrorNotInGuildBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", GuildOperationMemberQuitErrorNotInGuild, func(mode byte) packet.Encoder {
		return clientbound.NewMemberQuitErrorNotInGuild(mode)
	})
}

func GuildMemberExpelledErrorNotInGuildBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", GuildOperationMemberExpelledErrorNotInGuild, func(mode byte) packet.Encoder {
		return clientbound.NewMemberExpelledErrorNotInGuild(mode)
	})
}

func GuildDisbandErrorBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", GuildOperationDisbandError, func(mode byte) packet.Encoder {
		return clientbound.NewDisbandError(mode)
	})
}

func GuildCreateErrorCannotAsAdminBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", GuildOperationCreateErrorCannotAsAdmin, func(mode byte) packet.Encoder {
		return clientbound.NewCreateErrorCannotAsAdmin(mode)
	})
}

func GuildIncreaseCapacityErrorBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", GuildOperationIncreaseCapacityError, func(mode byte) packet.Encoder {
		return clientbound.NewIncreaseCapacityError(mode)
	})
}

func GuildQuestErrorLessThanSixMembersBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", GuildOperationQuestErrorLessThanSixMembers, func(mode byte) packet.Encoder {
		return clientbound.NewQuestErrorLessThanSixMembers(mode)
	})
}

func GuildQuestErrorDisconnectedBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", GuildOperationQuestErrorDisconnected, func(mode byte) packet.Encoder {
		return clientbound.NewQuestErrorDisconnected(mode)
	})
}

// GuildInfoBody emits the GUILDDATA info packet (GUILD_OPERATION sub-op 0x1A).
// Info is NOT one of the OnGuildResult dispatcher arms (it has no operations
// key in guild.yaml — it is the separate GUILDDATA::Decode path); its 0x1A mode
// is written directly by clientbound.Info.Encode. Left as-is by task-103 (out of
// the 35-key OnGuildResult dispatcher scope).
func GuildInfoBody(inGuild bool, guildId uint32, name string, titles [5]string, members []clientbound.GuildMemberInfo, capacity uint32, logoBackground uint16, logoBackgroundColor byte, logo uint16, logoColor byte, notice string, points uint32, allianceId uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return clientbound.NewInfo(inGuild, guildId, name, titles, members, capacity, logoBackground, logoBackgroundColor, logo, logoColor, notice, points, allianceId).Encode
}
