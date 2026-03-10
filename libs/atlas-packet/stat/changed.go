package stat

import (
	"context"
	"fmt"
	"sort"

	constants "github.com/Chronicle20/atlas-constants/stat"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const StatChangedWriter = "StatChanged"

type Update struct {
	statType constants.Type
	value    int64
}

func NewUpdate(statType constants.Type, value int64) Update {
	return Update{statType: statType, value: value}
}

func (u Update) Stat() constants.Type { return u.statType }
func (u Update) Value() int64         { return u.value }

type Changed struct {
	exclRequestSent bool
	updates         []Update
}

func NewStatChanged(updates []Update, exclRequestSent bool) Changed {
	return Changed{exclRequestSent: exclRequestSent, updates: updates}
}

func (m Changed) ExclRequestSent() bool { return m.exclRequestSent }
func (m Changed) Updates() []Update     { return m.updates }
func (m Changed) Operation() string     { return StatChangedWriter }
func (m Changed) String() string {
	return fmt.Sprintf("exclRequestSent [%t], updates [%d]", m.exclRequestSent, len(m.updates))
}

func (m Changed) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteBool(m.exclRequestSent)

		updates := make([]Update, len(m.updates))
		copy(updates, m.updates)
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
			switch u.Stat() {
			case constants.TypeSkin, constants.TypeLevel:
				w.WriteByte(byte(u.Value()))
			case constants.TypeJob, constants.TypeStrength, constants.TypeDexterity, constants.TypeIntelligence, constants.TypeLuck,
				constants.TypeHp, constants.TypeMaxHp, constants.TypeMp, constants.TypeMaxMp, constants.TypeAvailableAP, constants.TypeFame:
				w.WriteInt16(int16(u.Value()))
			case constants.TypeAvailableSP:
				w.WriteShort(uint16(u.Value()))
			case constants.TypeFace, constants.TypeHair, constants.TypeExperience, constants.TypeMeso, constants.TypeGachaponExperience:
				w.WriteInt(uint32(u.Value()))
			case constants.TypePetSn1, constants.TypePetSn2, constants.TypePetSn3:
				w.WriteLong(uint64(u.Value()))
			}
		}
		w.WriteByte(0)
		return w.Bytes()
	}
}

func (m *Changed) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		// server-send only — stat index resolution requires options["statistics"]
	}
}

func getStatIndex(l logrus.FieldLogger) func(options map[string]interface{}, statType constants.Type) uint8 {
	return func(options map[string]interface{}, statType constants.Type) uint8 {
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
