package guild

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type SetTitleNames struct {
	titles []string
}

func (m SetTitleNames) Titles() []string { return m.titles }

func (m SetTitleNames) Operation() string { return "SetTitleNames" }

func (m SetTitleNames) String() string {
	return fmt.Sprintf("titles %v", m.titles)
}

func (m SetTitleNames) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		for _, title := range m.titles {
			w.WriteAsciiString(title)
		}
		return w.Bytes()
	}
}

func (m *SetTitleNames) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.titles = make([]string, 5)
		for i := 0; i < 5; i++ {
			m.titles[i] = r.ReadAsciiString()
		}
	}
}
