package login

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CharacterViewAllPongHandle = "CharacterViewAllPongHandle"

type AllCharacterListPong struct {
	render bool
}

func (m AllCharacterListPong) Render() bool {
	return m.render
}

func (m AllCharacterListPong) Operation() string {
	return CharacterViewAllPongHandle
}

func (m AllCharacterListPong) String() string {
	return fmt.Sprintf("render [%t]", m.render)
}

func (m AllCharacterListPong) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteBool(m.Render())
		return w.Bytes()
	}
}

func (m AllCharacterListPong) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.render = r.ReadBool()
	}
}
