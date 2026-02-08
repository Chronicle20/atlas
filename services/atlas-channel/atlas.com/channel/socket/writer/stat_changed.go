package writer

import (
	"atlas-channel/socket/model"
	"sort"

	"github.com/Chronicle20/atlas-constants/stat"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const StatChanged = "StatChanged"

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
				if u.Stat() == stat.TypeSkin || u.Stat() == stat.TypeLevel {
					w.WriteByte(byte(u.Value()))
				} else if u.Stat() == stat.TypeJob || u.Stat() == stat.TypeStrength || u.Stat() == stat.TypeDexterity || u.Stat() == stat.TypeIntelligence || u.Stat() == stat.TypeLuck || u.Stat() == stat.TypeHp || u.Stat() == stat.TypeMaxHp || u.Stat() == stat.TypeMp || u.Stat() == stat.TypeMaxMp || u.Stat() == stat.TypeAvailableAP || u.Stat() == stat.TypeFame {
					w.WriteInt16(int16(u.Value()))
				} else if u.Stat() == stat.TypeAvailableSP {
					w.WriteShort(uint16(u.Value()))
				} else if u.Stat() == stat.TypeFace || u.Stat() == stat.TypeHair || u.Stat() == stat.TypeExperience || u.Stat() == stat.TypeMeso || u.Stat() == stat.TypeGachaponExperience {
					w.WriteInt(uint32(u.Value()))
				} else if u.Stat() == stat.TypePetSn1 || u.Stat() == stat.TypePetSn2 || u.Stat() == stat.TypePetSn3 {
					w.WriteLong(uint64(u.Value()))
				}
			}
			w.WriteByte(0)
			return w.Bytes()
		}
	}
}

func getStatIndex(l logrus.FieldLogger) func(options map[string]interface{}, statType stat.Type) uint8 {
	return func(options map[string]interface{}, statType stat.Type) uint8 {
		key := string(statType)

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
