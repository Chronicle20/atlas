package writer

import (
	"atlas-channel/socket/model"

	"github.com/Chronicle20/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const (
	CharacterSkillMacro = "CharacterSkillMacro"
)

func CharacterSkillMacroBody(l logrus.FieldLogger, t tenant.Model) func(m model.Macros) BodyProducer {
	return func(m model.Macros) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			m.Encode(l, t, options)(w)
			return w.Bytes()
		}
	}
}
