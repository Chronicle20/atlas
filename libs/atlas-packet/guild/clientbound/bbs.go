package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const GuildBBSWriter = "GuildBBS"

// Guild-BBS dispatcher mode bytes. CUIGuildBBS::OnGuildBBSPacket dispatches on
// (Decode1 - 6), so the raw leading mode bytes are 6/7/8. These are VERSION-
// STABLE across gms_v83/v84/v87/v95 (IDA-verified — guild_bbs.yaml) and are NOT
// resolved from the tenant operations table (no seed template registers a
// GuildBBS operations map). They are package consts (not config data and not
// struct-literal AP-2 footguns) passed through the body functions in bbs_body.go.
const (
	GuildBBSModeThreadList    byte = 6 // OnLoadListResult
	GuildBBSModeThread        byte = 7 // OnViewEntryResult
	GuildBBSModeEntryNotFound byte = 8 // OnEntryNotFound
)

type BBSThreadSummary struct {
	Id         uint32
	PosterId   uint32
	Title      string
	CreatedAt  int64 // ms time
	EmoticonId uint32
	ReplyCount uint32
}

type BBSReply struct {
	Id        uint32
	PosterId  uint32
	CreatedAt int64 // ms time
	Message   string
}

// BBSThreadList - thread listing
//
// CUIGuildBBS::OnGuildBBSPacket dispatches on (Decode1 - 6); this arm is mode 6
// (OnLoadListResult). The mode byte is version-stable (6 across gms_v83/84/87/95;
// jms-absent) and is NOT config-resolved — guild_bbs.yaml records that no seed
// template registers a GuildBBS operations table. It is therefore passed in via
// the constructor as a fixed package const (GuildBBSModeThreadList) rather than
// hard-coded as a struct-literal field, so no AP-2 mode:0x literal exists.
// packet-audit:fname CUIGuildBBS::OnGuildBBSPacket#BBSThreadList
type BBSThreadList struct {
	mode       byte
	hasNotice  bool
	notice     BBSThreadSummary
	threads    []BBSThreadSummary
	startIndex uint32
}

func NewBBSThreadList(mode byte, notice *BBSThreadSummary, threads []BBSThreadSummary, startIndex uint32) BBSThreadList {
	if notice != nil {
		return BBSThreadList{mode: mode, hasNotice: true, notice: *notice, threads: threads, startIndex: startIndex}
	}
	return BBSThreadList{mode: mode, hasNotice: false, threads: threads, startIndex: startIndex}
}

func (m BBSThreadList) Operation() string { return GuildBBSWriter }
func (m BBSThreadList) String() string {
	return fmt.Sprintf("bbs thread list [%d] threads", len(m.threads))
}

func (m BBSThreadList) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		if !m.hasNotice && len(m.threads) == 0 {
			w.WriteByte(0)
			w.WriteInt(0)
			return w.Bytes()
		}
		if m.hasNotice {
			w.WriteByte(1)
			w.WriteInt(m.notice.Id)
			w.WriteInt(m.notice.PosterId)
			w.WriteAsciiString(m.notice.Title)
			w.WriteInt64(m.notice.CreatedAt)
			w.WriteInt(m.notice.EmoticonId)
			w.WriteInt(m.notice.ReplyCount)
		} else {
			w.WriteByte(0)
		}
		w.WriteInt(uint32(len(m.threads)))
		if len(m.threads) > 0 {
			bound := uint32(len(m.threads)) - m.startIndex
			if bound > 10 {
				bound = 10
			}
			w.WriteInt(bound)
			for i := m.startIndex; i < m.startIndex+bound; i++ {
				w.WriteInt(m.threads[i].Id)
				w.WriteInt(m.threads[i].PosterId)
				w.WriteAsciiString(m.threads[i].Title)
				w.WriteInt64(m.threads[i].CreatedAt)
				w.WriteInt(m.threads[i].EmoticonId)
				w.WriteInt(m.threads[i].ReplyCount)
			}
		}
		return w.Bytes()
	}
}

func (m *BBSThreadList) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte() // mode 6 (OnLoadListResult)
		hasNotice := r.ReadByte()
		if hasNotice == 0 {
			totalCount := r.ReadUint32()
			if totalCount == 0 {
				return
			}
			m.threads = make([]BBSThreadSummary, 0)
			bound := r.ReadUint32()
			for i := uint32(0); i < bound; i++ {
				t := BBSThreadSummary{
					Id:         r.ReadUint32(),
					PosterId:   r.ReadUint32(),
					Title:      r.ReadAsciiString(),
					CreatedAt:  r.ReadInt64(),
					EmoticonId: r.ReadUint32(),
					ReplyCount: r.ReadUint32(),
				}
				m.threads = append(m.threads, t)
			}
		} else {
			m.hasNotice = true
			m.notice = BBSThreadSummary{
				Id:         r.ReadUint32(),
				PosterId:   r.ReadUint32(),
				Title:      r.ReadAsciiString(),
				CreatedAt:  r.ReadInt64(),
				EmoticonId: r.ReadUint32(),
				ReplyCount: r.ReadUint32(),
			}
			totalCount := r.ReadUint32()
			m.threads = make([]BBSThreadSummary, 0)
			if totalCount > 0 {
				bound := r.ReadUint32()
				for i := uint32(0); i < bound; i++ {
					t := BBSThreadSummary{
						Id:         r.ReadUint32(),
						PosterId:   r.ReadUint32(),
						Title:      r.ReadAsciiString(),
						CreatedAt:  r.ReadInt64(),
						EmoticonId: r.ReadUint32(),
						ReplyCount: r.ReadUint32(),
					}
					m.threads = append(m.threads, t)
				}
			}
		}
	}
}

// BBSThread - single thread detail
//
// Mode 7 (OnViewEntryResult), version-stable, NOT config-resolved (see
// BBSThreadList doc). Mode injected via constructor (GuildBBSModeThread const).
// packet-audit:fname CUIGuildBBS::OnGuildBBSPacket#BBSThread
type BBSThread struct {
	mode       byte
	id         uint32
	posterId   uint32
	createdAt  int64
	title      string
	message    string
	emoticonId uint32
	replies    []BBSReply
}

func NewBBSThread(mode byte, id uint32, posterId uint32, createdAt int64, title string, message string, emoticonId uint32, replies []BBSReply) BBSThread {
	return BBSThread{mode: mode, id: id, posterId: posterId, createdAt: createdAt, title: title, message: message, emoticonId: emoticonId, replies: replies}
}

func (m BBSThread) Operation() string { return GuildBBSWriter }
func (m BBSThread) String() string {
	return fmt.Sprintf("bbs thread id [%d]", m.id)
}

func (m BBSThread) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt(m.id)
		w.WriteInt(m.posterId)
		w.WriteInt64(m.createdAt)
		w.WriteAsciiString(m.title)
		w.WriteAsciiString(m.message)
		w.WriteInt(m.emoticonId)
		w.WriteInt(uint32(len(m.replies)))
		for _, r := range m.replies {
			w.WriteInt(r.Id)
			w.WriteInt(r.PosterId)
			w.WriteInt64(r.CreatedAt)
			w.WriteAsciiString(r.Message)
		}
		return w.Bytes()
	}
}

func (m *BBSThread) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte() // mode 7 (OnViewEntryResult)
		m.id = r.ReadUint32()
		m.posterId = r.ReadUint32()
		m.createdAt = r.ReadInt64()
		m.title = r.ReadAsciiString()
		m.message = r.ReadAsciiString()
		m.emoticonId = r.ReadUint32()
		replyCount := r.ReadUint32()
		m.replies = make([]BBSReply, replyCount)
		for i := range m.replies {
			m.replies[i] = BBSReply{
				Id:        r.ReadUint32(),
				PosterId:  r.ReadUint32(),
				CreatedAt: r.ReadInt64(),
				Message:   r.ReadAsciiString(),
			}
		}
	}
}

// BBSEntryNotFound - the (Decode1 - 6)==2 arm (OnEntryNotFound, mode 8).
// Mode-only: the sub-handler shows a "thread not found" notice with no further
// wire reads. Discrete struct per the discrete-per-mode rule. Mode injected via
// constructor (GuildBBSModeEntryNotFound); version-stable, not config-resolved.
// packet-audit:fname CUIGuildBBS::OnGuildBBSPacket#BBSEntryNotFound
type BBSEntryNotFound struct {
	mode byte
}

func NewBBSEntryNotFound(mode byte) BBSEntryNotFound { return BBSEntryNotFound{mode: mode} }
func (m BBSEntryNotFound) Operation() string         { return GuildBBSWriter }
func (m BBSEntryNotFound) String() string            { return fmt.Sprintf("bbs entry not found mode [%d]", m.mode) }
func (m BBSEntryNotFound) Encode(l logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(map[string]interface{}) []byte { w.WriteByte(m.mode); return w.Bytes() }
}
func (m *BBSEntryNotFound) Decode(_ logrus.FieldLogger, _ context.Context) func(*request.Reader, map[string]interface{}) {
	return func(r *request.Reader, _ map[string]interface{}) { m.mode = r.ReadByte() }
}
