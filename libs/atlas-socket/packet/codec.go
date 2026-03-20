package packet

type Codec interface {
	Encoder
	Decoder
}
