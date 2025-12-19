package model

import (
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
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
