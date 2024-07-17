package socket

import (
	"fmt"
	"github.com/Chronicle20/atlas-socket/crypto"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"net"
	"os"
	"time"
)

type OpReader[E uint8 | uint16] func(r *request.Reader) E

func ByteOpReader(r *request.Reader) uint8 {
	return r.ReadByte()
}

func ShortOpReader(r *request.Reader) uint16 {
	return r.ReadUint16()
}

type MessageHandlerProducer[E uint8 | uint16] func() map[E]request.Handler

type SessionCreator func(sessionId uuid.UUID, conn net.Conn)

func defaultSessionCreator(_ uuid.UUID, _ net.Conn) {
}

type SessionMessageDecryptor func(sessionId uuid.UUID, message []byte) []byte

func defaultSessionMessageDecryptor(_ uuid.UUID, message []byte) []byte {
	return message
}

type SessionDestroyer func(sessionId uuid.UUID)

func defaultSessionDestroyer(_ uuid.UUID) {
}

type serverConfiguration[E uint8 | uint16] struct {
	opReader  OpReader[E]
	creator   SessionCreator
	decryptor SessionMessageDecryptor
	destroyer SessionDestroyer
	ipAddress string
	port      int
	handlers  map[E]request.Handler
}

//goland:noinspection GoUnusedExportedFunction
func Run[E uint8 | uint16](l logrus.FieldLogger, handlerProducer MessageHandlerProducer[E], configurators ...ServerConfigurator[E]) error {
	config := &serverConfiguration[E]{
		creator:   defaultSessionCreator,
		decryptor: defaultSessionMessageDecryptor,
		destroyer: defaultSessionDestroyer,
		ipAddress: "0.0.0.0",
		port:      5000,
		handlers:  handlerProducer(),
	}

	for _, configurator := range configurators {
		configurator(config)
	}

	l.Infof("Starting tcp server on %s:%d", config.ipAddress, config.port)
	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", config.ipAddress, config.port))
	if err != nil {
		l.WithError(err).Errorln("Error listening:", err.Error())
		os.Exit(1)
	}
	defer lis.Close()

	for {
		conn, err := lis.Accept()
		if err != nil {
			l.WithError(err).Errorln("Error connecting:", err.Error())
			return err
		}

		l.Infof("Client %s connected.", conn.RemoteAddr().String())

		go run[E](l)(config, conn, uuid.New(), 4)
	}
}

func run[E uint8 | uint16](l logrus.FieldLogger) func(config *serverConfiguration[E], conn net.Conn, sessionId uuid.UUID, headerSize int) {
	return func(config *serverConfiguration[E], conn net.Conn, sessionId uuid.UUID, headerSize int) {

		defer func(conn net.Conn) {
			err := conn.Close()
			if err != nil {
			}
		}(conn)

		config.creator(sessionId, conn)

		header := true
		readSize := headerSize

		fl := l.WithField("session", sessionId.String())

		for {
			buffer := make([]byte, readSize)

			if _, err := conn.Read(buffer); err != nil {
				break
			}

			if header {
				readSize = crypto.PacketLength(buffer)
			} else {
				readSize = headerSize

				result := buffer
				result = config.decryptor(sessionId, buffer)
				handle[E](fl)(config, sessionId, result)
			}

			header = !header
		}

		fl.Infof("Exiting read loop.")
		config.destroyer(sessionId)
	}
}

func handle[E uint8 | uint16](l logrus.FieldLogger) func(config *serverConfiguration[E], sessionId uuid.UUID, p request.Request) {
	return func(config *serverConfiguration[E], sessionId uuid.UUID, p request.Request) {
		go func(sessionId uuid.UUID, reader request.Reader) {
			op := config.opReader(&reader)
			if h, ok := config.handlers[op]; ok {
				h(sessionId, reader)
			} else {
				l.Infof("Read a unhandled message with op 0x%02X.", op&0xFF)
			}
		}(sessionId, request.NewRequestReader(&p, time.Now().Unix()))
	}
}
