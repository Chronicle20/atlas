package clientbound

import (
	"context"
	"fmt"
	"sort"

	constants "github.com/Chronicle20/atlas/libs/atlas-constants/stat"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
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

func (m *Changed) Decode(l logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.exclRequestSent = r.ReadBool()
		updateMask := r.ReadUint32()

		statTypes := getStatTypes(l)(options)
		m.updates = nil
		for i := uint8(0); i < 32; i++ {
			if updateMask&(1<<i) == 0 {
				continue
			}
			if int(i) >= len(statTypes) {
				continue
			}
			st := statTypes[i]
			var val int64
			switch st {
			case constants.TypeSkin, constants.TypeLevel:
				val = int64(r.ReadByte())
			case constants.TypeJob, constants.TypeStrength, constants.TypeDexterity, constants.TypeIntelligence, constants.TypeLuck,
				constants.TypeHp, constants.TypeMaxHp, constants.TypeMp, constants.TypeMaxMp, constants.TypeAvailableAP, constants.TypeFame:
				val = int64(r.ReadInt16())
			case constants.TypeAvailableSP:
				val = int64(r.ReadUint16())
			case constants.TypeFace, constants.TypeHair, constants.TypeExperience, constants.TypeMeso, constants.TypeGachaponExperience:
				val = int64(r.ReadUint32())
			case constants.TypePetSn1, constants.TypePetSn2, constants.TypePetSn3:
				val = int64(r.ReadUint64())
			default:
				continue
			}
			m.updates = append(m.updates, Update{statType: st, value: val})
		}
		_ = r.ReadByte() // trailing zero
	}
}

func getStatTypes(l logrus.FieldLogger) func(options map[string]interface{}) []constants.Type {
	return func(options map[string]interface{}) []constants.Type {
		genericCodes, ok := options["statistics"]
		if !ok {
			l.Error("statistics not configured in options")
			return nil
		}
		var codes []string
		switch v := genericCodes.(type) {
		case []interface{}:
			for _, item := range v {
				if str, ok := item.(string); ok {
					codes = append(codes, str)
				}
			}
		}
		result := make([]constants.Type, len(codes))
		for i, c := range codes {
			result[i] = constants.Type(c)
		}
		return result
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
