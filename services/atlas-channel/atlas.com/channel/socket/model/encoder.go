package model

import (
	"atlas-channel/tenant"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type Encoder interface {
	Encode(l logrus.FieldLogger, tenant tenant.Model, options map[string]interface{}) func(w *response.Writer)
}

type Decoder interface {
	Decode(l logrus.FieldLogger, tenant tenant.Model, options map[string]interface{}) func(r *request.Reader)
}

type TypeEncoder interface {
	EncodeType(w *response.Writer)
}

type EncoderDecoder interface {
	Encoder
	TypeEncoder
	Decoder
}
