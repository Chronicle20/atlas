package guild

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const GuildBBSWriter = "GuildBBS"

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
type BBSThreadList struct {
	hasNotice  bool
	notice     BBSThreadSummary
	threads    []BBSThreadSummary
	startIndex uint32
}

func NewBBSThreadList(notice *BBSThreadSummary, threads []BBSThreadSummary, startIndex uint32) BBSThreadList {
	if notice != nil {
		return BBSThreadList{hasNotice: true, notice: *notice, threads: threads, startIndex: startIndex}
	}
	return BBSThreadList{hasNotice: false, threads: threads, startIndex: startIndex}
}

func (m BBSThreadList) Operation() string { return GuildBBSWriter }
func (m BBSThreadList) String() string {
	return fmt.Sprintf("bbs thread list [%d] threads", len(m.threads))
}

func (m BBSThreadList) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(0x06)
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
		_ = r.ReadByte() // 0x06
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
type BBSThread struct {
	id         uint32
	posterId   uint32
	createdAt  int64
	title      string
	message    string
	emoticonId uint32
	replies    []BBSReply
}

func NewBBSThread(id uint32, posterId uint32, createdAt int64, title string, message string, emoticonId uint32, replies []BBSReply) BBSThread {
	return BBSThread{id: id, posterId: posterId, createdAt: createdAt, title: title, message: message, emoticonId: emoticonId, replies: replies}
}

func (m BBSThread) Operation() string { return GuildBBSWriter }
func (m BBSThread) String() string {
	return fmt.Sprintf("bbs thread id [%d]", m.id)
}

func (m BBSThread) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(0x07)
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
		_ = r.ReadByte() // 0x07
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
