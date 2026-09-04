package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const CharacterUseStoredExperienceHandle = "CharacterUseStoredExperienceHandle"

// StoredExperienceUse - USE_GACHA_EXP. Redeems the whole stored-EXP counter
// (GW_CharacterStat::nTempEXP, persisted by Atlas as the misnamed
// `gachapon_experience` column) into real character EXP.
//
// CUIStatusBar::TryUseTempExp is the sole caller of the sender; it gates on the
// EXP-bar click rect, characterLevel <= 50, nTempEXP > 0 and a YesNo confirm.
// The sender itself re-checks the job guard and characterLevel <= 50. None of
// that is on the wire — the body is the tick and nothing else, on every version
// that carries the op, so there is no version gate here.
//
// packet-audit:fname CWvsContext::SendTempExpUseRequest
type StoredExperienceUse struct {
	updateTime uint32
}

func (m StoredExperienceUse) UpdateTime() uint32 { return m.updateTime }

func (m StoredExperienceUse) Operation() string {
	return CharacterUseStoredExperienceHandle
}

func (m StoredExperienceUse) String() string {
	return fmt.Sprintf("updateTime [%d]", m.updateTime)
}

func (m StoredExperienceUse) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.updateTime)
		return w.Bytes()
	}
}

func (m *StoredExperienceUse) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.updateTime = r.ReadUint32()
	}
}
