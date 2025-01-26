package writer

import (
	"atlas-channel/socket/model"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const (
	StatChanged            = "StatChanged"
	StatSkin               = "SKIN"
	StatFace               = "FACE"
	StatHair               = "HAIR"
	StatPetSn1             = "PET_SN_1"
	StatLevel              = "LEVEL"
	StatJob                = "JOB"
	StatStrength           = "STRENGTH"
	StatDexterity          = "DEXTERITY"
	StatIntelligence       = "INTELLIGENCE"
	StatLuck               = "LUCK"
	StatHp                 = "HP"
	StatMaxHp              = "MAX_HP"
	StatMp                 = "MP"
	StatMaxMp              = "MAX_MP"
	StatAvailableAp        = "AVAILABLE_AP"
	StatAvailableSp        = "AVAILABLE_SP"
	StatExperience         = "EXPERIENCE"
	StatFame               = "FAME"
	StatMeso               = "MESO"
	StatPetSn2             = "PET_SN_2"
	StatPetSn3             = "PET_SN_3"
	StatGachaponExperience = "GACHAPON_EXPERIENCE"
)

func StatChangedBody(l logrus.FieldLogger) func(updates []model.StatUpdate, exclRequestSent bool) BodyProducer {
	// TODO this should transmit stats
	return func(updates []model.StatUpdate, exclRequestSent bool) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteBool(exclRequestSent)

			updateMask := uint32(0)
			for _, u := range updates {
				index := getStatIndex(l)(options, u.Stat())
				mask := uint32(1 << index)
				updateMask |= mask
			}

			w.WriteInt(updateMask)

			for _, u := range updates {
				switch u.Stat() {
				case StatSkin:
				case StatLevel:
					w.WriteByte(byte(u.Value()))
				case StatJob:
				case StatStrength:
				case StatDexterity:
				case StatIntelligence:
				case StatLuck:
				case StatHp:
				case StatMaxHp:
				case StatMp:
				case StatMaxMp:
				case StatAvailableAp:
				case StatFame:
					w.WriteInt16(int16(u.Value()))
				case StatAvailableSp:
					w.WriteShort(uint16(u.Value()))
				case StatFace:
				case StatHair:
				case StatExperience:
				case StatMeso:
				case StatGachaponExperience:
					w.WriteInt(uint32(u.Value()))
				case StatPetSn1:
				case StatPetSn2:
				case StatPetSn3:
					w.WriteLong(uint64(u.Value()))
				}
			}
			w.WriteByte(0)
			return w.Bytes()
		}
	}
}

func getStatIndex(l logrus.FieldLogger) func(options map[string]interface{}, stat string) uint8 {
	return func(options map[string]interface{}, key string) uint8 {

		var genericCodes interface{}
		var ok bool
		if genericCodes, ok = options["statistics"]; !ok {
			l.Errorf("Code [%s] not configured for use. Defaulting to 99 which will likely cause a client crash.", key)
			return 99
		}

		var codes []string
		switch v := genericCodes.(type) {
		case string:
			codes = append(codes, v)
		case []interface{}:
			for _, item := range v {
				str, ok := item.(string)
				if !ok {
					l.Errorf("Code [%s] not configured for use. Defaulting to 99 which will likely cause a client crash.", key)
					return 99
				}
				codes = append(codes, str)
			}
		case interface{}:
			if str, ok := v.(string); ok {
				codes = append(codes, str)
			} else {
				l.Errorf("Code [%s] not configured for use. Defaulting to 99 which will likely cause a client crash.", key)
				return 99
			}
		}

		for i, code := range codes {
			if code == key {
				return uint8(i)
			}
		}
		l.Errorf("Code [%s] not configured for use. Defaulting to 99 which will likely cause a client crash.", key)
		return 99
	}
}
