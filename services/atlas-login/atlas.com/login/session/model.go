package session

import (
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-socket/crypto"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
)

type Model struct {
	id          uuid.UUID
	accountId   uint32
	ch          channel.Model
	con         net.Conn
	send        crypto.AESOFB
	sendLock    *sync.Mutex
	recv        crypto.AESOFB
	encryptFunc crypto.EncryptFunc
	lastPacket  time.Time
	locale      byte
}

func NewSession(id uuid.UUID, t tenant.Model, locale byte, con net.Conn) Model {
	recvIv := []byte{byte(rand.Float64() * 255), byte(rand.Float64() * 255), byte(rand.Float64() * 255), byte(rand.Float64() * 255)}
	sendIv := []byte{byte(rand.Float64() * 255), byte(rand.Float64() * 255), byte(rand.Float64() * 255), byte(rand.Float64() * 255)}

	var send *crypto.AESOFB
	var recv *crypto.AESOFB
	if t.Region() == "GMS" && t.MajorVersion() <= 12 {
		send = crypto.NewAESOFB(sendIv, uint16(65535)-t.MajorVersion(), crypto.SetIvGenerator(crypto.FillIvZeroGenerator))
		recv = crypto.NewAESOFB(recvIv, t.MajorVersion(), crypto.SetIvGenerator(crypto.FillIvZeroGenerator))
	} else {
		send = crypto.NewAESOFB(sendIv, uint16(65535)-t.MajorVersion())
		recv = crypto.NewAESOFB(recvIv, t.MajorVersion())
	}

	hasMapleEncryption := true
	if t.Region() == "JMS" {
		hasMapleEncryption = false
	}

	return Model{
		id:          id,
		con:         con,
		send:        *send,
		sendLock:    &sync.Mutex{},
		recv:        *recv,
		encryptFunc: send.Encrypt(hasMapleEncryption, true),
		lastPacket:  time.Now(),
		locale:      locale,
	}
}

func CloneSession(s Model) Model {
	return Model{
		id:          s.id,
		accountId:   s.accountId,
		ch:          s.ch,
		con:         s.con,
		send:        s.send,
		sendLock:    s.sendLock,
		recv:        s.recv,
		encryptFunc: s.encryptFunc,
		lastPacket:  s.lastPacket,
		locale:      s.locale,
	}
}

func (s *Model) setAccountId(accountId uint32) Model {
	ns := CloneSession(*s)
	ns.accountId = accountId
	return ns
}

func (s *Model) SessionId() uuid.UUID {
	return s.id
}

func (s *Model) AccountId() uint32 {
	return s.accountId
}

func (s *Model) announceEncrypted(b []byte) error {
	s.sendLock.Lock()
	defer s.sendLock.Unlock()

	tmp := make([]byte, len(b)+4)
	copy(tmp, b)
	tmp = append([]byte{0, 0, 0, 0}, b...)
	tmp = s.encryptFunc(tmp)
	_, err := s.con.Write(tmp)
	return err
}

func (s *Model) announce(b []byte) error {
	s.sendLock.Lock()
	defer s.sendLock.Unlock()

	_, err := s.con.Write(b)
	return err
}

func (s *Model) WriteHello(majorVersion uint16, minorVersion uint16) error {
	return s.announce(WriteHello(nil)(majorVersion, minorVersion, s.send.IV(), s.recv.IV(), s.locale))
}

func (s *Model) ReceiveAESOFB() *crypto.AESOFB {
	return &s.recv
}

func (s *Model) GetRemoteAddress() net.Addr {
	return s.con.RemoteAddr()
}

func (s *Model) setWorldId(worldId world.Id) Model {
	ns := CloneSession(*s)
	ns.ch = ns.ch.Clone().SetWorldId(worldId).Build()
	return ns
}

func (s *Model) setChannelId(channelId channel.Id) Model {
	ns := CloneSession(*s)
	ns.ch = ns.ch.Clone().SetId(channelId).Build()
	return ns
}

func (s *Model) WorldId() world.Id {
	return s.Channel().WorldId()
}

func (s *Model) ChannelId() channel.Id {
	return s.Channel().Id()
}

func (s *Model) Channel() channel.Model {
	return s.ch
}

func (s *Model) updateLastRequest() Model {
	ns := CloneSession(*s)
	ns.lastPacket = time.Now()
	return ns
}

func (s *Model) LastRequest() time.Time {
	return s.lastPacket
}

func (s *Model) Disconnect() {
	_ = s.con.Close()
}
