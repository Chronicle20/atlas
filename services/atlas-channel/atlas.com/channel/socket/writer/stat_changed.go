package writer

import (
	"atlas-channel/socket/model"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
	"sort"
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
	return func(updates []model.StatUpdate, exclRequestSent bool) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteBool(exclRequestSent)

			sort.Slice(updates, func(i, j int) bool {
				return getStatIndex(l)(options, updates[i].Stat()) < getStatIndex(l)(options, updates[j].Stat())
			})

			updateMask := uint32(0)
			for _, u := range updates {
				index := getStatIndex(l)(options, u.Stat())
				mask := uint32(1 << index)
				updateMask |= mask
			}

			w.WriteInt(updateMask)

			for _, u := range updates {
				if u.Stat() == StatSkin || u.Stat() == StatLevel {
					w.WriteByte(byte(u.Value()))
				} else if u.Stat() == StatJob || u.Stat() == StatStrength || u.Stat() == StatDexterity || u.Stat() == StatIntelligence || u.Stat() == StatLuck || u.Stat() == StatHp || u.Stat() == StatMaxHp || u.Stat() == StatMp || u.Stat() == StatMaxMp || u.Stat() == StatAvailableAp || u.Stat() == StatFame {
					w.WriteInt16(int16(u.Value()))
				} else if u.Stat() == StatAvailableSp {
					w.WriteShort(uint16(u.Value()))
				} else if u.Stat() == StatFace || u.Stat() == StatHair || u.Stat() == StatExperience || u.Stat() == StatMeso || u.Stat() == StatGachaponExperience {
					w.WriteInt(uint32(u.Value()))
				} else if u.Stat() == StatPetSn1 || u.Stat() == StatPetSn2 || u.Stat() == StatPetSn3 {
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
