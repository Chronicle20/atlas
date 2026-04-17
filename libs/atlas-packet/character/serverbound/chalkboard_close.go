package serverbound

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

const ChalkboardCloseHandle = "ChalkboardCloseHandle"

// ChalkboardClose - CUser::SendCloseChalkboard
type ChalkboardClose struct{}

func (m ChalkboardClose) Operation() string {
	return ChalkboardCloseHandle
}

func (m ChalkboardClose) String() string {
	return ""
}

func (m ChalkboardClose) Encode(_ logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	return func(options map[string]interface{}) []byte {
		return []byte{}
	}
}

func (m *ChalkboardClose) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
	}
}
