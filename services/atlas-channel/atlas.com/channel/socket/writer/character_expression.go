package writer

import "github.com/Chronicle20/atlas-socket/response"

const CharacterExpression = "CharacterExpression"

func CharacterExpressionBody(characterId uint32, expression uint32) BodyProducer {
	return func(w *response.Writer, options map[string]interface{}) []byte {
		w.WriteInt(characterId)
		w.WriteInt(expression)
		return w.Bytes()
	}
}
