package thread

import (
	"atlas-guilds/thread/reply"
	"github.com/Chronicle20/atlas-model/model"
	"strconv"
	"time"
)

type RestModel struct {
	Id         uint32            `json:"-"`
	PosterId   uint32            `json:"posterId"`
	Title      string            `json:"title"`
	Message    string            `json:"message"`
	EmoticonId uint32            `json:"emoticonId"`
	Notice     bool              `json:"notice"`
	Replies    []reply.RestModel `json:"replies"`
	CreatedAt  time.Time         `json:"createdAt"`
}

func (r RestModel) GetName() string {
	return "threads"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *RestModel) SetID(strId string) error {
	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

func Transform(m Model) (RestModel, error) {
	rs, err := model.SliceMap(reply.Transform)(model.FixedProvider(m.replies))()()
	if err != nil {
		return RestModel{}, err
	}

	return RestModel{
		Id:         m.id,
		PosterId:   m.posterId,
		Title:      m.title,
		Message:    m.message,
		EmoticonId: m.emoticonId,
		Notice:     m.notice,
		Replies:    rs,
		CreatedAt:  m.createdAt,
	}, nil
}
