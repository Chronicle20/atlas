package writer

import (
	"atlas-channel/guild"
	"atlas-channel/socket/model"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"strconv"
)

const (
	GuildOperation                               = "GuildOperation"
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

func RequestGuildNameBody(l logrus.FieldLogger) BodyProducer {
	return GuildErrorBody(l)(GuildOperationRequestName)
}

func RequestGuildEmblemBody(l logrus.FieldLogger) BodyProducer {
	return GuildErrorBody(l)(GuildOperationRequestEmblem)
}

func GuildRequestAgreement(l logrus.FieldLogger) func(partyId uint32, leaderName string, guildName string) BodyProducer {
	return func(partyId uint32, leaderName string, guildName string) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getGuildOperation(l)(options, GuildOperationRequestAgreement))
			w.WriteInt(partyId)
			w.WriteAsciiString(leaderName)
			w.WriteAsciiString(guildName)
			return w.Bytes()
		}
	}
}

func GuildErrorBody(l logrus.FieldLogger) func(code string) BodyProducer {
	return func(code string) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getGuildOperation(l)(options, code))
			return w.Bytes()
		}
	}
}

func GuildErrorBody2(l logrus.FieldLogger) func(code string, target string) BodyProducer {
	return func(code string, target string) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getGuildOperation(l)(options, code))
			w.WriteAsciiString(target)
			return w.Bytes()
		}
	}
}

func GuildInfoBody(l logrus.FieldLogger, t tenant.Model) func(g guild.Model) BodyProducer {
	return func(g guild.Model) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(0x1A) // TODO

			inGuild := g.Id() != 0
			w.WriteBool(inGuild)
			if !inGuild {
				return w.Bytes()
			}
			w.WriteInt(g.Id())
			w.WriteAsciiString(g.Name())
			for i := range 5 {
				for _, t := range g.Titles() {
					if t.Index() == byte(i)+1 {
						w.WriteAsciiString(t.Name())
					}
				}
			}
			w.WriteByte(byte(len(g.Members())))
			for _, mm := range g.Members() {
				w.WriteInt(mm.CharacterId())
			}
			for _, mm := range g.Members() {
				gm := model.GuildMember{
					Name:          mm.Name(),
					JobId:         mm.JobId(),
					Level:         mm.Level(),
					Title:         mm.Title(),
					Online:        mm.Online(),
					Signature:     0,
					AllianceTitle: mm.AllianceTitle(),
				}
				gm.Encode(l, t, options)(w)
			}
			w.WriteInt(g.Capacity())
			w.WriteShort(g.LogoBackground())
			w.WriteByte(g.LogoBackgroundColor())
			w.WriteShort(g.Logo())
			w.WriteByte(g.LogoColor())
			w.WriteAsciiString(g.Notice())
			w.WriteInt(g.Points())
			w.WriteInt(g.AllianceId())
			return w.Bytes()
		}
	}
}

func GuildEmblemChangedBody(l logrus.FieldLogger) func(guildId uint32, logo uint16, logoColor byte, logoBackground uint16, logoBackgroundColor byte) BodyProducer {
	return func(guildId uint32, logo uint16, logoColor byte, logoBackground uint16, logoBackgroundColor byte) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getGuildOperation(l)(options, GuildOperationEmblemChange))
			w.WriteInt(guildId)
			w.WriteShort(logoBackground)
			w.WriteByte(logoBackgroundColor)
			w.WriteShort(logo)
			w.WriteByte(logoColor)
			return w.Bytes()
		}
	}
}

func GuildMemberStatusUpdatedBody(l logrus.FieldLogger) func(guildId uint32, characterId uint32, online bool) BodyProducer {
	return func(guildId uint32, characterId uint32, online bool) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getGuildOperation(l)(options, GuildOperationMemberOnline))
			w.WriteInt(guildId)
			w.WriteInt(characterId)
			w.WriteBool(online)
			return w.Bytes()
		}
	}
}

func GuildMemberTitleUpdatedBody(l logrus.FieldLogger) func(guildId uint32, characterId uint32, title byte) BodyProducer {
	return func(guildId uint32, characterId uint32, title byte) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getGuildOperation(l)(options, GuildOperationMemberTitleChange))
			w.WriteInt(guildId)
			w.WriteInt(characterId)
			w.WriteByte(title)
			return w.Bytes()
		}
	}
}

func GuildNoticeChangedBody(l logrus.FieldLogger) func(guildId uint32, notice string) BodyProducer {
	return func(guildId uint32, notice string) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getGuildOperation(l)(options, GuildOperationNoticeChange))
			w.WriteInt(guildId)
			w.WriteAsciiString(notice)
			return w.Bytes()
		}
	}
}

func GuildMemberLeftBody(l logrus.FieldLogger) func(guildId uint32, characterId uint32, name string) BodyProducer {
	return func(guildId uint32, characterId uint32, name string) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getGuildOperation(l)(options, GuildOperationMemberQuitSuccess))
			w.WriteInt(guildId)
			w.WriteInt(characterId)
			w.WriteAsciiString(name)
			return w.Bytes()
		}
	}
}

func GuildMemberExpelBody(l logrus.FieldLogger) func(guildId uint32, characterId uint32, name string) BodyProducer {
	return func(guildId uint32, characterId uint32, name string) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getGuildOperation(l)(options, GuildOperationMemberExpelledSuccess))
			w.WriteInt(guildId)
			w.WriteInt(characterId)
			w.WriteAsciiString(name)
			return w.Bytes()
		}
	}
}

func GuildMemberJoinedBody(l logrus.FieldLogger, t tenant.Model) func(guildId uint32, characterId uint32, name string, jobId uint16, level byte, title byte, online bool, allianceTitle byte) BodyProducer {
	return func(guildId uint32, characterId uint32, name string, jobId uint16, level byte, title byte, online bool, allianceTitle byte) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getGuildOperation(l)(options, GuildOperationJoinSuccess))
			w.WriteInt(guildId)
			w.WriteInt(characterId)
			gm := model.GuildMember{
				Name:          name,
				JobId:         jobId,
				Level:         level,
				Title:         title,
				Online:        online,
				Signature:     0,
				AllianceTitle: allianceTitle,
			}
			gm.Encode(l, t, options)(w)
			return w.Bytes()
		}
	}
}

func GuildInviteBody(l logrus.FieldLogger) func(guildId uint32, originatorName string) BodyProducer {
	return func(guildId uint32, originatorName string) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getGuildOperation(l)(options, GuildOperationInvite))
			w.WriteInt(guildId)
			w.WriteAsciiString(originatorName)
			return w.Bytes()
		}
	}
}

func GuildTitleChangedBody(l logrus.FieldLogger) func(guildId uint32, titles []string) BodyProducer {
	return func(guildId uint32, titles []string) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getGuildOperation(l)(options, GuildOperationTitleUpdate))
			w.WriteInt(guildId)
			for i := range 5 {
				w.WriteAsciiString(titles[i])
			}
			return w.Bytes()
		}
	}
}

func GuildDisbandBody(l logrus.FieldLogger) func(guildId uint32) BodyProducer {
	return func(guildId uint32) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getGuildOperation(l)(options, GuildOperationDisbandSuccess))
			w.WriteInt(guildId)
			return w.Bytes()
		}
	}
}

func GuildCapacityChangedBody(l logrus.FieldLogger) func(guildId uint32, capacity uint32) BodyProducer {
	return func(guildId uint32, capacity uint32) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getGuildOperation(l)(options, GuildOperationIncreaseCapacitySuccess))
			w.WriteInt(guildId)
			w.WriteInt(capacity)
			return w.Bytes()
		}
	}
}

func getGuildOperation(l logrus.FieldLogger) func(options map[string]interface{}, key string) byte {
	return func(options map[string]interface{}, key string) byte {
		var genericCodes interface{}
		var ok bool
		if genericCodes, ok = options["operations"]; !ok {
			l.Errorf("Code [%s] not configured for use. Defaulting to 99 which will likely cause a client crash.", key)
			return 99
		}

		var codes map[string]interface{}
		if codes, ok = genericCodes.(map[string]interface{}); !ok {
			l.Errorf("Code [%s] not configured for use. Defaulting to 99 which will likely cause a client crash.", key)
			return 99
		}

		var code interface{}
		if code, ok = codes[key]; !ok {
			l.Errorf("Code [%s] not configured for use. Defaulting to 99 which will likely cause a client crash.", key)
			return 99
		}

		op, err := strconv.ParseUint(code.(string), 0, 16)
		if err != nil {
			l.Errorf("Code [%s] not configured for use. Defaulting to 99 which will likely cause a client crash.", key)
			return 99
		}
		return byte(op)
	}
}
