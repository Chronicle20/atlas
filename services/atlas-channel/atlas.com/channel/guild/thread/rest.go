package thread

import (
	"atlas-channel/guild/thread/reply"
	"strconv"
	"time"

	"github.com/Chronicle20/atlas-model/model"
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

func Extract(rm RestModel) (Model, error) {
	rs, err := model.SliceMap(reply.Extract)(model.FixedProvider(rm.Replies))()()
	if err != nil {
		return Model{}, err
	}

	return Model{
		id:         rm.Id,
		posterId:   rm.PosterId,
		title:      rm.Title,
		message:    rm.Message,
		emoticonId: rm.EmoticonId,
		notice:     rm.Notice,
		createdAt:  rm.CreatedAt,
		replies:    rs,
	}, nil
}
